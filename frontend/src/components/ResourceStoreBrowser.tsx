import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import {
  DownloadRemoteStoreProject,
  GetCatalogStoreSettings,
  GetInstanceDetails,
  HasCurseForgeAPIKey,
  OpenBrowserURL,
  SearchRemoteStore,
} from "../../wailsjs/go/main/App";
import { launcher } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ArrowLeft, Check, ChevronDown, Loader2 } from "lucide-react";
import { toast } from "sonner";

export type StoreCategory = "modpacks" | "mods" | "datapacks" | "resourcepacks" | "shaderpacks";

type StoreSourceToggle = "curseforge" | "modrinth" | "both";

type StoreSort = "popularity" | "downloads";

type RemoteHitSide = {
  projectId: string;
  slug: string;
  pageUrl: string;
  downloads: number;
};

type RemoteHit = {
  source: string;
  projectId: string;
  slug: string;
  title: string;
  summary: string;
  iconUrl: string;
  pageUrl: string;
  downloads: number;
  cf?: RemoteHitSide;
  mr?: RemoteHitSide;
};

const CF_KEY_HINT =
  "Укажите API-ключ CurseForge в настройках лаунчера для установки через CurseForge и полнотекстового поиска по их API.";
/** Когда ключа нет, бэкенд строит выдачу через Modrinth + CFWidget (только совпадение slug). */
const CF_BROWSE_WITHOUT_KEY =
  "Без API-ключа CurseForge список строится через Modrinth и CFWidget: в выдаче остаются проекты, у которых на CurseForge совпадает slug с Modrinth. Установка с CurseForge по-прежнему требует ключа.";

const CF_CATALOG_OFF = "Включите CurseForge в настройках лаунчера (раздел «CurseForge»).";
const MR_CATALOG_OFF = "Включите Modrinth в настройках лаунчера (раздел «Modrinth»).";
const BOTH_CATALOG_OFF = "Включите оба каталога в настройках лаунчера.";

const CATEGORY_LABEL: Record<StoreCategory, string> = {
  mods: "Моды",
  modpacks: "Модпаки",
  datapacks: "Датапаки",
  resourcepacks: "Ресурспаки",
  shaderpacks: "Шейдерпаки",
};

function normalizeStoreSlug(s: string): string {
  return s.trim().toLowerCase();
}

/** Для сравнения названий: нижний регистр, скобки с уточнениями отбрасываются. */
function normalizeCatalogTitle(t: string): string {
  let s = t.trim().toLowerCase().replace(/\s+/g, " ");
  const paren = s.indexOf("(");
  if (paren > 0) s = s.slice(0, paren).trim();
  return s;
}

/**
 * Совпадение карточки каталога с remote-installs без привязки к источнику строки:
 * учитываются projectId/slug с CurseForge и Modrinth с карточки и нормализованный title
 * (Architectury с CF совпадёт со строкой только Modrinth при разных slug).
 */
function catalogRowMatchesRemoteInstall(
  category: StoreCategory,
  map: Record<string, launcher.RemoteInstallMeta>,
  row: {
    title: string;
    cfProjectId: string;
    cfSlug: string;
    mrProjectId: string;
    mrSlug: string;
  }
): boolean {
  const projectIds = new Set<string>();
  const slugs = new Set<string>();
  for (const p of [row.cfProjectId, row.mrProjectId]) {
    const t = p?.trim() ?? "";
    if (t) projectIds.add(t);
  }
  for (const s of [row.cfSlug, row.mrSlug]) {
    const t = normalizeStoreSlug(s);
    if (t) slugs.add(t);
  }
  const titleNorm = normalizeCatalogTitle(row.title);
  const useTitle = titleNorm.length >= 3;

  for (const rec of Object.values(map)) {
    if ((rec.category ?? "") !== category) continue;
    const rp = rec.projectId?.trim() ?? "";
    if (rp && projectIds.has(rp)) return true;
    const rs = normalizeStoreSlug(rec.slug ?? "");
    if (rs && slugs.has(rs)) return true;
    if (useTitle) {
      const rt = normalizeCatalogTitle(rec.title ?? "");
      if (rt && rt === titleNorm) return true;
    }
  }
  return false;
}

export function ResourceStoreBrowser({
  instanceName,
  category,
  onClose,
  onInstallSuccess,
}: {
  instanceName: string;
  category: StoreCategory;
  onClose: () => void;
  /** Вызывается после успешной установки из каталога (обновить список ресурсов). */
  onInstallSuccess?: () => void;
}) {
  const [source, setSource] = useState<StoreSourceToggle>("both");
  const [query, setQuery] = useState("");
  const [storeSort, setStoreSort] = useState<StoreSort>("popularity");
  const [page, setPage] = useState(0);
  const [hits, setHits] = useState<RemoteHit[]>([]);
  const [loading, setLoading] = useState(false);
  const [instanceMeta, setInstanceMeta] = useState("");
  const [catalogError, setCatalogError] = useState("");
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [remoteInstallMap, setRemoteInstallMap] = useState<Record<string, launcher.RemoteInstallMeta>>({});
  const [cfKeyOk, setCfKeyOk] = useState(false);
  const [catalogCfEnabled, setCatalogCfEnabled] = useState(true);
  const [catalogMrEnabled, setCatalogMrEnabled] = useState(true);

  const refreshCfKey = useCallback(() => {
    void HasCurseForgeAPIKey()
      .then(setCfKeyOk)
      .catch(() => setCfKeyOk(false));
  }, []);

  const refreshCatalogSettings = useCallback(() => {
    void GetCatalogStoreSettings()
      .then((s) => {
        setCatalogCfEnabled(s?.curseforge_enabled !== false);
        setCatalogMrEnabled(s?.modrinth_enabled !== false);
      })
      .catch(() => {
        setCatalogCfEnabled(true);
        setCatalogMrEnabled(true);
      });
  }, []);

  useEffect(() => {
    void refreshCfKey();
    void refreshCatalogSettings();
    const unsubCf = EventsOn("curseforge-settings-changed", () => {
      void refreshCfKey();
    });
    const unsubCat = EventsOn("catalog-store-settings-changed", () => {
      void refreshCatalogSettings();
    });
    const onFocus = () => {
      void refreshCfKey();
      void refreshCatalogSettings();
    };
    window.addEventListener("focus", onFocus);
    return () => {
      unsubCf?.();
      unsubCat?.();
      window.removeEventListener("focus", onFocus);
    };
  }, [refreshCfKey, refreshCatalogSettings]);

  useEffect(() => {
    const canCf = catalogCfEnabled;
    const canMr = catalogMrEnabled;
    let valid = false;
    if (source === "curseforge") valid = canCf;
    else if (source === "modrinth") valid = canMr;
    else if (source === "both") valid = canCf && canMr;
    if (valid) return;
    if (canMr) setSource("modrinth");
    else if (canCf) setSource("curseforge");
  }, [source, catalogCfEnabled, catalogMrEnabled]);

  const refreshInstanceInstalls = useCallback(() => {
    GetInstanceDetails(instanceName)
      .then((d) => {
        if (d) {
          setInstanceMeta(`Minecraft ${d.gameVersion ?? "?"} · ${d.loader ?? "vanilla"}`);
          setRemoteInstallMap((d.remoteInstalls as Record<string, launcher.RemoteInstallMeta> | undefined) ?? {});
        } else {
          setInstanceMeta("");
          setRemoteInstallMap({});
        }
      })
      .catch(() => {
        setInstanceMeta("");
        setRemoteInstallMap({});
      });
  }, [instanceName]);

  useEffect(() => {
    refreshInstanceInstalls();
  }, [refreshInstanceInstalls]);

  useEffect(() => {
    setPage(0);
  }, [query, source, storeSort, category, instanceName]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const res = await SearchRemoteStore(
          instanceName,
          category,
          source,
          query.trim(),
          storeSort,
          storeSort,
          page
        );
        if (!cancelled) {
          const err = (res?.error ?? "").trim();
          const rawHits = res?.hits;
          if (err) {
            setCatalogError(err);
            setHits([]);
          } else {
            setCatalogError("");
            setHits(Array.isArray(rawHits) ? (rawHits as unknown as RemoteHit[]) : []);
          }
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    const t = window.setTimeout(load, 320);
    return () => {
      cancelled = true;
      window.clearTimeout(t);
    };
  }, [instanceName, category, source, query, storeSort, page]);

  const openPage = useCallback(async (u: string) => {
    const trimmed = u?.trim();
    if (!trimmed) return;
    const err = await OpenBrowserURL(trimmed);
    if (err) toast.error(err);
  }, []);

  const download = useCallback(
    async (
      from: "curseforge" | "modrinth",
      projectId: string,
      slug: string,
      title: string,
      iconUrl?: string
    ) => {
      const pid = projectId?.trim() ?? "";
      const sg = slug?.trim() ?? "";
      const key = `${from}:${pid}:${sg}`;
      setBusyKey(key);
      try {
        const err = await DownloadRemoteStoreProject(
          instanceName,
          category,
          from,
          pid,
          sg,
          title?.trim() ?? "",
          iconUrl?.trim() ?? ""
        );
        if (err) toast.error(err);
        else {
          onInstallSuccess?.();
          refreshInstanceInstalls();
          toast.success("Установлено в инстанс", {
            description: "Файл добавлен в папку ресурсов инстанса.",
          });
        }
      } finally {
        setBusyKey(null);
      }
    },
    [instanceName, category, onInstallSuccess, refreshInstanceInstalls]
  );

  const pageSize = 20;

  const canInstallCf = catalogCfEnabled && cfKeyOk;
  const canInstallMr = catalogMrEnabled;

  if (!instanceName.trim()) {
    return (
      <div className="min-h-screen bg-background flex flex-col items-center justify-center gap-4 p-8">
        <p className="text-muted-foreground text-sm">Не выбран инстанс. Откройте каталог из вкладки «Ресурсы».</p>
        <Button type="button" variant="outline" onClick={onClose}>
          Назад
        </Button>
      </div>
    );
  }

  return (
    <TooltipProvider delayDuration={300}>
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <header className="shrink-0 border-b border-border px-4 py-3 flex flex-wrap items-center justify-between gap-3 bg-card/40">
        <div className="min-w-0">
          <h1 className="text-lg font-semibold tracking-tight">Каталог ресурсов</h1>
          <p className="text-xs text-muted-foreground truncate">
            {instanceName} · {CATEGORY_LABEL[category]}
            {instanceMeta ? ` · ${instanceMeta}` : ""}
          </p>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={onClose} className="shrink-0">
          <ArrowLeft className="w-4 h-4 mr-2" />
          Закрыть
        </Button>
      </header>

      <div className="shrink-0 border-b border-border px-4 py-3 space-y-3 bg-muted/20">
        <div className="space-y-1">
          <Label className="text-xs text-muted-foreground">Источник</Label>
          <div className="flex flex-wrap gap-2">
            {(
              [
                ["curseforge", "CurseForge"],
                ["modrinth", "Modrinth"],
                ["both", "Оба источника"],
              ] as const
            ).map(([id, label]) => {
              let blocked = false;
              let hint = CF_KEY_HINT;
              if (id === "curseforge") {
                blocked = !catalogCfEnabled;
                hint = CF_CATALOG_OFF;
              } else if (id === "modrinth") {
                blocked = !catalogMrEnabled;
                hint = MR_CATALOG_OFF;
              } else {
                blocked = !catalogCfEnabled || !catalogMrEnabled;
                hint =
                  !catalogCfEnabled && !catalogMrEnabled
                    ? BOTH_CATALOG_OFF
                    : !catalogCfEnabled
                      ? CF_CATALOG_OFF
                      : MR_CATALOG_OFF;
              }
              const inner = (
                <Button
                  type="button"
                  size="sm"
                  variant={source === id ? "secondary" : "outline"}
                  disabled={blocked}
                  onClick={() => setSource(id)}
                >
                  {label}
                </Button>
              );
              if (!blocked) return <span key={id}>{inner}</span>;
              return (
                <Tooltip key={id}>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">{inner}</span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom" className="max-w-xs">
                    {hint}
                  </TooltipContent>
                </Tooltip>
              );
            })}
          </div>
          {catalogCfEnabled && !cfKeyOk && (source === "curseforge" || source === "both") ? (
            <p className="text-xs text-muted-foreground max-w-3xl leading-relaxed">{CF_BROWSE_WITHOUT_KEY}</p>
          ) : null}
        </div>

        <div className="flex flex-col sm:flex-row gap-3 sm:items-end">
          <div className="flex-1 space-y-1 min-w-0">
            <Label htmlFor="store-search" className="text-xs text-muted-foreground">
              Поиск
            </Label>
            <Input
              id="store-search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Название или ключевые слова…"
              className="max-w-xl"
            />
          </div>
          <div className="space-y-1 w-full sm:w-52 shrink-0">
            <Label htmlFor="store-sort" className="text-xs text-muted-foreground">
              Сортировка
            </Label>
            <NativeSelect
              id="store-sort"
              value={storeSort}
              onChange={(e) => {
                const v = e.target.value;
                if (v === "popularity" || v === "downloads") setStoreSort(v);
              }}
            >
              <NativeSelectOption value="popularity">Популярность</NativeSelectOption>
              <NativeSelectOption value="downloads">Загрузки</NativeSelectOption>
            </NativeSelect>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-3 py-4 sm:px-5">
        {loading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="w-4 h-4 animate-spin" />
            Загрузка…
          </div>
        ) : hits.length === 0 ? (
          catalogError ? (
            <p className="text-sm text-destructive">{catalogError}</p>
          ) : (
            <p className="text-sm text-muted-foreground">Ничего не найдено. Измените запрос или страницу.</p>
          )
        ) : (
          <div className="grid w-full max-w-none gap-3">
            {hits.map((h) => {
              const isBoth = h.source === "both" && h.cf && h.mr;
              const fromCf = h.source === "curseforge" || isBoth;
              const fromMr = h.source === "modrinth" || isBoth;
              const cfPid = isBoth ? h.cf!.projectId : h.projectId;
              const cfSlug = isBoth ? h.cf!.slug : h.slug;
              const mrPid = isBoth ? h.mr!.projectId : h.projectId;
              const mrSlug = isBoth ? h.mr!.slug : h.slug;
              const dlCf = `curseforge:${cfPid}:${cfSlug}`;
              const dlMr = `modrinth:${mrPid}:${mrSlug}`;
              const cardKey = isBoth ? `both-${cfPid}-${mrPid}` : `${h.source}-${h.projectId}-${h.slug}`;
              const busyBoth = busyKey === dlCf || busyKey === dlMr;

              const installedFromCatalog = catalogRowMatchesRemoteInstall(category, remoteInstallMap, {
                title: h.title,
                cfProjectId: isBoth ? h.cf!.projectId : h.source === "curseforge" ? h.projectId : "",
                cfSlug: isBoth ? h.cf!.slug : h.source === "curseforge" ? h.slug : "",
                mrProjectId: isBoth ? h.mr!.projectId : h.source === "modrinth" ? h.projectId : "",
                mrSlug: isBoth ? h.mr!.slug : h.source === "modrinth" ? h.slug : "",
              });

              const installedLabel = (
                <div
                  className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400"
                  title="Уже добавлено в этот инстанс из каталога (CurseForge или Modrinth)"
                >
                  <Check className="w-4 h-4 shrink-0" strokeWidth={2.5} aria-hidden />
                  <span className="text-xs whitespace-nowrap">Уже в инстансе</span>
                </div>
              );

              const addInstallActions = installedFromCatalog ? (
                installedLabel
              ) : isBoth ? (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={busyBoth}
                      className="inline-flex items-center gap-2"
                    >
                      {busyBoth ? <Loader2 className="w-4 h-4 animate-spin shrink-0" /> : null}
                      <span>Добавить в инстанс</span>
                      <ChevronDown className="w-4 h-4 opacity-70" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="min-w-56">
                    <DropdownMenuItem
                      disabled={busyKey === dlCf || !canInstallCf}
                      title={
                        !canInstallCf
                          ? !catalogCfEnabled
                            ? CF_CATALOG_OFF
                            : CF_KEY_HINT
                          : undefined
                      }
                      onSelect={() => download("curseforge", cfPid, cfSlug, h.title, h.iconUrl)}
                    >
                      Установить с CurseForge
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      disabled={busyKey === dlMr || !canInstallMr}
                      title={!canInstallMr ? MR_CATALOG_OFF : undefined}
                      onSelect={() => download("modrinth", mrPid, mrSlug, h.title, h.iconUrl)}
                    >
                      Установить с Modrinth
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : h.source === "curseforge" ? (
                !canInstallCf ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled
                          className="inline-flex items-center gap-2"
                        >
                          <span>Установить с CurseForge</span>
                        </Button>
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left" className="max-w-xs">
                      {!catalogCfEnabled ? CF_CATALOG_OFF : CF_KEY_HINT}
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={busyKey === dlCf}
                    onClick={() => download("curseforge", h.projectId, h.slug, h.title, h.iconUrl)}
                    className="inline-flex items-center gap-2"
                  >
                    {busyKey === dlCf ? <Loader2 className="w-4 h-4 animate-spin shrink-0" /> : null}
                    <span>Установить с CurseForge</span>
                  </Button>
                )
              ) : !canInstallMr ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <Button
                        type="button"
                        size="sm"
                        variant="default"
                        disabled
                        className="inline-flex items-center gap-2"
                      >
                        <span>Установить с Modrinth</span>
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="left" className="max-w-xs">
                    {MR_CATALOG_OFF}
                  </TooltipContent>
                </Tooltip>
              ) : (
                <Button
                  type="button"
                  size="sm"
                  variant="default"
                  disabled={busyKey === dlMr}
                  onClick={() => download("modrinth", h.projectId, h.slug, h.title, h.iconUrl)}
                  className="inline-flex items-center gap-2"
                >
                  {busyKey === dlMr ? <Loader2 className="w-4 h-4 animate-spin shrink-0" /> : null}
                  <span>Установить с Modrinth</span>
                </Button>
              );

              return (
                <div
                  key={cardKey}
                  className="w-full min-w-0 rounded-lg border border-border bg-card/50 p-3 flex flex-col sm:flex-row gap-3"
                >
                  <div className="shrink-0 w-14 h-14 rounded-md overflow-hidden bg-muted border border-border">
                    {h.iconUrl ? (
                      <img src={h.iconUrl} alt="" className="w-full h-full object-cover" loading="lazy" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center text-[10px] text-muted-foreground">
                        —
                      </div>
                    )}
                  </div>
                  <div className="flex-1 min-w-0 space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-foreground">{h.title}</span>
                      {fromCf ? (
                        <span className="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded border border-amber-600/50 text-amber-700 dark:text-amber-400">
                          CurseForge
                        </span>
                      ) : null}
                      {fromMr ? (
                        <span className="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded border border-emerald-600/50 text-emerald-700 dark:text-emerald-400">
                          Modrinth
                        </span>
                      ) : null}
                      {typeof h.downloads === "number" && h.downloads > 0 ? (
                        <span className="text-xs text-muted-foreground">
                          {h.downloads.toLocaleString()} загрузок
                        </span>
                      ) : null}
                    </div>
                    {h.summary ? (
                      <p className="text-xs text-muted-foreground line-clamp-3">{h.summary}</p>
                    ) : null}
                    <div className="flex flex-wrap gap-x-3 gap-y-1">
                      {isBoth ? (
                        <>
                          {h.cf?.pageUrl ? (
                            <button
                              type="button"
                              className="text-xs text-primary hover:underline"
                              onClick={() => openPage(h.cf!.pageUrl)}
                            >
                              CurseForge — страница
                            </button>
                          ) : null}
                          {h.mr?.pageUrl ? (
                            <button
                              type="button"
                              className="text-xs text-primary hover:underline"
                              onClick={() => openPage(h.mr!.pageUrl)}
                            >
                              Modrinth — страница
                            </button>
                          ) : null}
                        </>
                      ) : h.pageUrl ? (
                        <button
                          type="button"
                          className="text-xs text-primary hover:underline"
                          onClick={() => openPage(h.pageUrl)}
                        >
                          Страница проекта
                        </button>
                      ) : null}
                    </div>
                  </div>
                  <div className="shrink-0 flex flex-col gap-2 sm:items-end justify-center">
                    {addInstallActions}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        <div className="mt-6 flex w-full max-w-none flex-wrap items-center gap-2 border-t border-border pt-4">
          <Button type="button" size="sm" variant="outline" disabled={page <= 0 || loading} onClick={() => setPage((p) => Math.max(0, p - 1))}>
            Назад
          </Button>
          <span className="text-xs text-muted-foreground">Страница {page + 1}</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={loading || hits.length < pageSize}
            onClick={() => setPage((p) => p + 1)}
          >
            Далее
          </Button>
        </div>
      </div>
    </div>
    </TooltipProvider>
  );
}
