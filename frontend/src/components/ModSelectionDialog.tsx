import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";

interface ModMetadata {
  optional: boolean;
  /** When true, QMAdmin turned off delivery to QMLauncher — always excluded from sync. */
  launcher_disabled?: boolean;
  display_name?: string;
  icon_url?: string;
  description?: string;
  curseforge_url?: string;
  modrinth_url?: string;
  depends_on?: string[];
  depends_on_required?: string[];
  incompatible_with?: string[];
  load_order?: number;
}

interface ModPreset {
  id: string;
  name: string;
  mods_to_disable: string[];
  mods_to_enable: string[];
  resourcepacks_to_enable?: string[];
  shaderpacks_to_enable?: string[];
  is_default?: boolean;
}

interface ModConfig {
  mods_metadata: Record<string, ModMetadata>;
  mod_presets: ModPreset[];
  mod_paths?: string[];
  resourcepacks_metadata?: Record<string, ModMetadata>;
  resourcepacks_paths?: string[];
  shaderpacks_metadata?: Record<string, ModMetadata>;
  shaderpacks_paths?: string[];
}

const STORAGE_KEY = (sid: number) => `qm-server-mod-selection-${sid}`;

interface ModSelectionData {
  presetId: string;
  optionalEnabled: Record<string, boolean>;
  optionalResourcepacksEnabled: Record<string, boolean>;
  optionalShaderpacksEnabled: Record<string, boolean>;
}

interface ModSelectionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  serverID: number;
  serverName: string;
  apiBase: string;
  onConfirm: (disabledPaths: string[], enabledResourcepacksOrder?: string[]) => void;
  onCancel: () => void;
}

export function ModSelectionDialog({
  open,
  onOpenChange,
  serverID,
  serverName,
  apiBase,
  onConfirm,
  onCancel,
}: ModSelectionDialogProps) {
  const [config, setConfig] = useState<ModConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedPresetId, setSelectedPresetId] = useState<string>("");
  const [optionalEnabled, setOptionalEnabled] = useState<Record<string, boolean>>({});
  const [optionalResourcepacksEnabled, setOptionalResourcepacksEnabled] = useState<Record<string, boolean>>({});
  const [optionalShaderpacksEnabled, setOptionalShaderpacksEnabled] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (!open || serverID <= 0 || !apiBase) return;
    setLoading(true);
    const url = `${apiBase.replace(/\/$/, "")}/servers/${serverID}/mod-config`;
    fetch(url)
      .then((res) => res.ok ? res.json() : Promise.reject(new Error("Failed to fetch")))
      .then((data: ModConfig) => {
        setConfig(data);
        // Загрузить сохранённый выбор, если есть
        const saved = loadSavedSelection(serverID);
        if (saved) {
          const presetExists = !saved.presetId || data?.mod_presets?.some((p) => p.id === saved.presetId);
          const modPaths = data?.mod_paths ?? [];
          const rpPaths = data?.resourcepacks_paths ?? [];
          const spPaths = data?.shaderpacks_paths ?? [];
          // Применяем только если пресет существует и пути всё ещё есть на сервере
          if (presetExists) {
            setSelectedPresetId(saved.presetId ?? "");
            const optMod = modPaths.filter((p) => data?.mods_metadata?.[p]?.optional);
            const optRp = rpPaths.filter((p) => data?.resourcepacks_metadata?.[p]?.optional);
            const optSp = spPaths.filter((p) => data?.shaderpacks_metadata?.[p]?.optional);
            setOptionalEnabled(filterByPaths(saved.optionalEnabled, optMod));
            setOptionalResourcepacksEnabled(filterByPaths(saved.optionalResourcepacksEnabled, optRp));
            setOptionalShaderpacksEnabled(filterByPaths(saved.optionalShaderpacksEnabled, optSp));
          } else {
            applyDefaults(data);
          }
        } else {
          applyDefaults(data);
        }
      })
      .catch(() => {
        setConfig({ mods_metadata: {}, mod_presets: [], mod_paths: [], resourcepacks_paths: [], shaderpacks_paths: [] });
        setOptionalEnabled({});
        setOptionalResourcepacksEnabled({});
        setOptionalShaderpacksEnabled({});
        setSelectedPresetId("");
      })
      .finally(() => setLoading(false));
  }, [open, serverID, apiBase]);

  function loadSavedSelection(sid: number): ModSelectionData | null {
    try {
      const raw = localStorage.getItem(STORAGE_KEY(sid));
      if (!raw) return null;
      const parsed = JSON.parse(raw) as ModSelectionData;
      if (parsed && typeof parsed.presetId === "string") return parsed;
    } catch {
      /* ignore */
    }
    return null;
  }

  function filterByPaths(rec: Record<string, boolean>, validPaths: string[]): Record<string, boolean> {
    const out: Record<string, boolean> = {};
    for (const p of validPaths) {
      if (rec[p] === true) out[p] = true;
    }
    return out;
  }

  function applyDefaults(data: ModConfig | null) {
    setOptionalEnabled({});
    setOptionalResourcepacksEnabled({});
    setOptionalShaderpacksEnabled({});
    const defaultPreset = data?.mod_presets?.find((p) => p.is_default);
    setSelectedPresetId(defaultPreset?.id ?? "");
  }

  const modPaths = config?.mod_paths ?? [];
  const optionalModPaths = modPaths.filter(
    (p) => config?.mods_metadata?.[p]?.optional
  );
  const rpPaths = config?.resourcepacks_paths ?? [];
  const optionalRpPaths = rpPaths.filter(
    (p) => config?.resourcepacks_metadata?.[p]?.optional
  );
  const spPaths = config?.shaderpacks_paths ?? [];
  const optionalSpPaths = spPaths.filter(
    (p) => config?.shaderpacks_metadata?.[p]?.optional
  );

  const preset = config?.mod_presets?.find((p) => p.id === selectedPresetId);
  const presetDisabledSet = new Set(preset?.mods_to_disable ?? []);

  const isLauncherAdminDisabled = (path: string, kind: "mod" | "rp" | "sp"): boolean => {
    if (kind === "mod") return Boolean(config?.mods_metadata?.[path]?.launcher_disabled);
    if (kind === "rp") return Boolean(config?.resourcepacks_metadata?.[path]?.launcher_disabled);
    return Boolean(config?.shaderpacks_metadata?.[path]?.launcher_disabled);
  };

  /** Моды, отключённые пресетом — нельзя переключать, всегда выключены */
  const isLockedByPreset = (path: string): boolean =>
    Boolean(selectedPresetId && presetDisabledSet.has(path));

  /** Моды, несовместимые с этим (из metadata) */
  const getIncompatibleWith = (path: string): string[] =>
    config?.mods_metadata?.[path]?.incompatible_with ?? [];

  /** Мод заблокирован: другой включённый мод помечен этот как несовместимый */
  const isModLockedByIncompatible = (path: string): boolean => {
    for (const other of optionalModPaths) {
      if (other === path) continue;
      const otherEnabled = optionalEnabled[other] ?? defaultOptionalEnabled;
      if (otherEnabled && getIncompatibleWith(other).includes(path)) return true;
    }
    return false;
  };

  /** Название мода, из-за которого этот заблокирован (несовместимость) */
  const getConflictingModName = (path: string): string | null => {
    for (const other of optionalModPaths) {
      if (other === path) continue;
      const otherEnabled = optionalEnabled[other] ?? defaultOptionalEnabled;
      if (otherEnabled && getIncompatibleWith(other).includes(path)) {
        return config?.mods_metadata?.[other]?.display_name ?? other.split("/").pop() ?? other;
      }
    }
    return null;
  };

  /** Без пресета: всё отключено. С пресетом: моды — включены, кроме mods_to_disable; рп/шп — только если перечислены в списках (отсутствие или пустой массив = ничего не вкл). */
  const defaultOptionalEnabled = Boolean(selectedPresetId);
  const defaultResourcepackEnabled = (path: string): boolean => {
    if (!selectedPresetId || !preset) return false;
    const list = preset.resourcepacks_to_enable;
    return Array.isArray(list) && list.length > 0 && list.includes(path);
  };
  const defaultShaderpackEnabled = (path: string): boolean => {
    if (!selectedPresetId || !preset) return false;
    const list = preset.shaderpacks_to_enable;
    return Array.isArray(list) && list.length > 0 && list.includes(path);
  };

  /** Транзитивные обязательные зависимости мода */
  const getModTransitiveRequiredDeps = (path: string): string[] => {
    const meta = config?.mods_metadata?.[path];
    const deps = meta?.depends_on_required ?? [];
    const result = new Set<string>(deps);
    deps.forEach((d) => getModTransitiveRequiredDeps(d).forEach((x) => result.add(x)));
    return Array.from(result);
  };

  /** Все зависимости мода (обязательные + опциональные, транзитивно) */
  const getModTransitiveDeps = (path: string): string[] => {
    const meta = config?.mods_metadata?.[path];
    const req = meta?.depends_on_required ?? [];
    const opt = meta?.depends_on ?? [];
    const result = new Set<string>([...req, ...opt]);
    [...req, ...opt].forEach((d) => getModTransitiveDeps(d).forEach((x) => result.add(x)));
    return Array.from(result);
  };

  /** Мод заблокирован как обязательная зависимость включённого мода */
  const isModLockedAsRequiredDep = (path: string): boolean => {
    for (const other of optionalModPaths) {
      if (other === path) continue;
      const otherEnabled = optionalEnabled[other] ?? defaultOptionalEnabled;
      if (otherEnabled && getModTransitiveRequiredDeps(other).includes(path)) return true;
    }
    return false;
  };

  /** Мод считается включённым: явно, как обязательная или опциональная зависимость включённого */
  const isModEffectivelyEnabled = (path: string): boolean => {
    const explicit = optionalEnabled[path] ?? defaultOptionalEnabled;
    if (explicit) return true;
    for (const other of optionalModPaths) {
      if (other === path) continue;
      const otherEnabled = optionalEnabled[other] ?? defaultOptionalEnabled;
      if (otherEnabled && getModTransitiveDeps(other).includes(path)) return true;
    }
    return false;
  };

  /** Все транзитивные зависимости ресурспака (рекурсивно) */
  const getRpTransitiveDeps = (path: string): string[] => {
    const meta = config?.resourcepacks_metadata?.[path];
    const deps = meta?.depends_on ?? [];
    const result = new Set<string>(deps);
    deps.forEach((d) => getRpTransitiveDeps(d).forEach((x) => result.add(x)));
    return Array.from(result);
  };

  /** Ресурспак считается включённым: явно пользователем или как зависимость включённого */
  const isResourcepackEffectivelyEnabled = (path: string): boolean => {
    if (isLauncherAdminDisabled(path, "rp")) return false;
    const explicit = optionalResourcepacksEnabled[path] ?? defaultResourcepackEnabled(path);
    if (explicit) return true;
    for (const other of optionalRpPaths) {
      if (other === path) continue;
      const otherEnabled = optionalResourcepacksEnabled[other] ?? defaultResourcepackEnabled(other);
      if (otherEnabled && getRpTransitiveDeps(other).includes(path)) return true;
    }
    return false;
  };

  /** Мод включён (будет использоваться) */
  const isModEnabled = (path: string) =>
    isLauncherAdminDisabled(path, "mod")
      ? false
      : isLockedByPreset(path)
        ? false
        : isModEffectivelyEnabled(path);

  /** Мод заблокирован для переключения (пресет, обязательная зависимость или несовместимость) */
  const isModSwitchLocked = (path: string) =>
    isLauncherAdminDisabled(path, "mod") ||
    isLockedByPreset(path) ||
    isModLockedAsRequiredDep(path) ||
    isModLockedByIncompatible(path);

  /** Отсортированный список: сначала включённые, потом выключенные */
  const sortedOptionalPaths = [...optionalModPaths].sort((a, b) => {
    const aEnabled = isModEnabled(a);
    const bEnabled = isModEnabled(b);
    if (aEnabled === bEnabled) return 0;
    return aEnabled ? -1 : 1;
  });

  const computeDisabledPaths = (): string[] => {
    const disabled = new Set<string>();
    modPaths.forEach((p) => {
      if (config?.mods_metadata?.[p]?.launcher_disabled) disabled.add(p);
    });
    rpPaths.forEach((p) => {
      if (config?.resourcepacks_metadata?.[p]?.launcher_disabled) disabled.add(p);
    });
    spPaths.forEach((p) => {
      if (config?.shaderpacks_metadata?.[p]?.launcher_disabled) disabled.add(p);
    });
    if (preset?.mods_to_disable) {
      preset.mods_to_disable.forEach((path) => disabled.add(path));
    }
    optionalModPaths.forEach((path) => {
      if (isLockedByPreset(path) || !isModEffectivelyEnabled(path)) {
        disabled.add(path);
      }
    });
    optionalRpPaths.forEach((path) => {
      if (!isResourcepackEffectivelyEnabled(path)) {
        disabled.add(path);
      }
    });
    optionalSpPaths.forEach((path) => {
      if (!(optionalShaderpacksEnabled[path] ?? defaultShaderpackEnabled(path))) {
        disabled.add(path);
      }
    });
    return Array.from(disabled);
  };

  const handleConfirm = () => {
    const disabled = computeDisabledPaths();
    const enabledRp = rpPaths.filter((p) => !disabled.includes(p));
    const enabledRpOrdered = [...enabledRp].sort((a, b) => {
      const orderA = config?.resourcepacks_metadata?.[a]?.load_order ?? 0;
      const orderB = config?.resourcepacks_metadata?.[b]?.load_order ?? 0;
      return orderA - orderB;
    });
    // Сохранить выбор для сервера
    if (serverID > 0) {
      try {
        const data: ModSelectionData = {
          presetId: selectedPresetId,
          optionalEnabled: optionalEnabled,
          optionalResourcepacksEnabled: optionalResourcepacksEnabled,
          optionalShaderpacksEnabled: optionalShaderpacksEnabled,
        };
        localStorage.setItem(STORAGE_KEY(serverID), JSON.stringify(data));
      } catch {
        /* ignore */
      }
    }
    onConfirm(disabled, enabledRpOrdered.length > 0 ? enabledRpOrdered : undefined);
    onOpenChange(false);
  };

  const hasOptionalMods = optionalModPaths.length > 0;
  const hasPresets = (config?.mod_presets?.length ?? 0) > 0;
  const hasOptionalResourcepacks = optionalRpPaths.length > 0;
  const hasOptionalShaderpacks = optionalSpPaths.length > 0;
  const hasAnyOptional = hasOptionalMods || hasPresets || hasOptionalResourcepacks || hasOptionalShaderpacks;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex flex-col min-w-[20rem] max-w-[min(95vw,900px)] max-h-[85vh] overflow-hidden p-4 sm:p-6">
        <DialogHeader className="flex-shrink-0">
          <DialogTitle>Настройка модов и ресурсов</DialogTitle>
          <DialogDescription>
            Сервер: {serverName}. Выберите пресет, опциональные моды, ресурспаки и шейдеры для подключения.
          </DialogDescription>
        </DialogHeader>
        <div className="flex-1 min-h-0 overflow-auto space-y-6 py-4">
          {loading ? (
            <p className="text-sm text-muted-foreground">Загрузка настроек...</p>
          ) : !hasAnyOptional ? (
            <p className="text-sm text-muted-foreground">
              Нет опциональных модов, ресурспаков или шейдеров для настройки. Нажмите «Запустить» для продолжения.
            </p>
          ) : (
            <>
              {hasPresets && (
                <div className="space-y-2">
                  <Label>Пресет модов и ресурсов</Label>
                  <NativeSelect
                    value={selectedPresetId}
                    onChange={(e) => setSelectedPresetId(e.target.value)}
                  >
                    <NativeSelectOption value="">Без пресета</NativeSelectOption>
                    {config?.mod_presets?.map((p) => (
                      <NativeSelectOption key={p.id} value={p.id}>
                        {p.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </div>
              )}
              <div className="flex flex-col sm:flex-row gap-6 min-w-0">
                {hasOptionalMods && (
                <div className="flex-1 min-w-0 space-y-2">
                  <Label>Опциональные моды</Label>
                  <p className="text-xs text-muted-foreground mb-2">
                    Включите моды, которые хотите использовать
                  </p>
                  <div className="space-y-2 max-h-48 overflow-y-auto scrollbar-hide">
                    {sortedOptionalPaths.map((path) => {
                      const meta = config?.mods_metadata?.[path];
                      const name = meta?.display_name || path.split("/").pop() || path;
                      const locked = isModSwitchLocked(path);
                      const isEnabled = isModEnabled(path);
                      const isDepOnly = isEnabled && !(optionalEnabled[path] ?? defaultOptionalEnabled);
                      return (
                        <div
                          key={path}
                          className={`flex items-center justify-between gap-2 rounded-lg border px-3 py-2 ${locked ? "opacity-75 bg-muted/50" : isDepOnly ? "opacity-90 bg-muted/30" : ""}`}
                        >
                          <div className="flex min-w-0 flex-1 items-center gap-2">
                            {meta?.icon_url ? (
                              <img src={meta.icon_url} alt="" className="size-6 shrink-0 rounded object-cover" />
                            ) : (
                              <div className="size-6 shrink-0 rounded bg-muted" />
                            )}
                            <span
                              className="text-sm truncate"
                              title={
                                isModLockedByIncompatible(path)
                                  ? `Несовместим с ${getConflictingModName(path) ?? "другим модом"}`
                                  : meta?.description ?? path
                              }
                            >
                              {name}
                              {isDepOnly && !locked && (
                                <span className="ml-2 text-xs text-muted-foreground">(зависимость)</span>
                              )}
                              {isModLockedByIncompatible(path) && (
                                <span className="ml-2 text-xs text-muted-foreground">(несовместим)</span>
                              )}
                            </span>
                          </div>
                          <Switch
                            checked={isEnabled}
                            disabled={locked}
                            onCheckedChange={(v) => {
                              const next = { ...optionalEnabled, [path]: v };
                              if (v) {
                                getModTransitiveDeps(path).forEach((dep) => { next[dep] = true; });
                                getIncompatibleWith(path).forEach((inc) => { next[inc] = false; });
                              }
                              setOptionalEnabled(next);
                            }}
                          />
                        </div>
                      );
                    })}
                  </div>
                </div>
                )}
                {hasOptionalResourcepacks && (
                <div className="flex-1 min-w-0 space-y-2">
                  <Label>Опциональные ресурспаки</Label>
                  <p className="text-xs text-muted-foreground mb-2">
                    {selectedPresetId ? "Включите ресурспаки, которые хотите использовать" : "Без пресета: отключены по умолчанию. Включите нужные."}
                  </p>
                  <div className="space-y-2 max-h-48 overflow-y-auto scrollbar-hide">
                    {optionalRpPaths.map((path) => {
                      const meta = config?.resourcepacks_metadata?.[path];
                      const name = meta?.display_name || path.split("/").pop() || path;
                      const adminOff = isLauncherAdminDisabled(path, "rp");
                      const isEnabled = isResourcepackEffectivelyEnabled(path);
                      const isDepOnly = isEnabled && !(optionalResourcepacksEnabled[path] ?? defaultResourcepackEnabled(path));
                      return (
                        <div
                          key={path}
                          className={`flex items-center justify-between gap-2 rounded-lg border px-3 py-2 ${isDepOnly ? "opacity-90 bg-muted/30" : ""}`}
                        >
                          <div className="flex min-w-0 flex-1 items-center gap-2">
                            {meta?.icon_url ? (
                              <img src={meta.icon_url} alt="" className="size-6 shrink-0 rounded object-cover" />
                            ) : (
                              <div className="size-6 shrink-0 rounded bg-muted" />
                            )}
                            <span className="text-sm truncate" title={meta?.description ?? path}>
                              {name}
                              {isDepOnly && (
                                <span className="ml-2 text-xs text-muted-foreground">(зависимость)</span>
                              )}
                            </span>
                          </div>
                          <Switch
                            checked={isEnabled}
                            disabled={adminOff}
                            onCheckedChange={(v) => {
                              const next = { ...optionalResourcepacksEnabled, [path]: v };
                              if (v) {
                                getRpTransitiveDeps(path).forEach((dep) => { next[dep] = true; });
                              }
                              setOptionalResourcepacksEnabled(next);
                            }}
                          />
                        </div>
                      );
                    })}
                  </div>
                </div>
                )}
              </div>
              {hasOptionalShaderpacks && (
                <div className="space-y-2">
                  <Label>Опциональные шейдеры</Label>
                  <p className="text-xs text-muted-foreground mb-2">
                    {selectedPresetId ? "Включите шейдеры, которые хотите использовать" : "Без пресета: отключены по умолчанию. Включите нужные."}
                  </p>
                  <div className="space-y-2 max-h-36 overflow-y-auto scrollbar-hide">
                    {optionalSpPaths.map((path) => {
                      const meta = config?.shaderpacks_metadata?.[path];
                      const name = meta?.display_name || path.split("/").pop() || path;
                      const adminOff = isLauncherAdminDisabled(path, "sp");
                      const isEnabled =
                        !adminOff &&
                        (optionalShaderpacksEnabled[path] ?? defaultShaderpackEnabled(path));
                      return (
                        <div
                          key={path}
                          className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2"
                        >
                          <div className="flex min-w-0 flex-1 items-center gap-2">
                            {meta?.icon_url ? (
                              <img src={meta.icon_url} alt="" className="size-6 shrink-0 rounded object-cover" />
                            ) : (
                              <div className="size-6 shrink-0 rounded bg-muted" />
                            )}
                            <span className="text-sm truncate" title={meta?.description ?? path}>
                              {name}
                            </span>
                          </div>
                          <Switch
                            checked={isEnabled}
                            disabled={adminOff}
                            onCheckedChange={(v) =>
                              setOptionalShaderpacksEnabled((prev) => ({ ...prev, [path]: v }))
                            }
                          />
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
        <DialogFooter className="flex-shrink-0">
          <Button variant="outline" onClick={onCancel}>
            Отмена
          </Button>
          <Button onClick={handleConfirm} disabled={loading}>
            Запустить
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
