import { useState, useEffect, useRef, useMemo, lazy, Suspense } from "react";
import type { ChangeEvent, CSSProperties } from "react";
import { Button } from "@/components/ui/button";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { LucideIcon } from "lucide-react";
import {
  Play,
  Plus,
  Trash2,
  LogIn,
  LogOut,
  CloudUpload,
  ArrowLeft,
  Settings,
  FolderOpen,
  Cloud,
  Check,
  User,
  Package,
  Box,
  Image,
  Sparkles,
  Database,
  Archive,
  FileJson,
  FileText,
  Boxes,
  ChevronDown,
} from "lucide-react";
import { launcher } from "../wailsjs/go/models";
import { GetInstances, GetInstanceDetails, LaunchInstanceWithAccount, GetRecentServers, GetQMServersError, InvalidateQMServersCache, EnsureInstanceForServer, GetAccounts, LoginAccount, LogoutAccount, CreateLocalAccount, DeleteLocalAccount, SetDefaultAccount, GetCurrentAccount, GetCloudProfile, OpenBrowserForQMServerCloud, OpenBrowserForMicrosoft, GetMicrosoftAuthAvailable, LogoutCloudAccount, SyncLocalAccountToCloud, SyncMicrosoftAccountToCloud, GetCloudGameAccounts, UpdateCloudGameAccount, DeleteCloudGameAccount, GetSkinProviderConfig, GetCloudElyLinked, GetNews, GetQMServerAPIBase, GetLauncherAPITarget, SetLauncherAPITarget, GetLauncherDebug, SetLauncherDebug, GetCurseForgeKeySettings, SetCurseForgeSettingsKey, GetCatalogStoreSettings, SetCatalogStoreSettings, SetLang, GetLang, GetLauncherVersion, GetLauncherAboutInfo, CheckLauncherUpdateAvailable, CreateCloudGameAccount, SetInstanceMemory, GetGameAccountInventory, CreateInstance, OpenPath, ApplyLauncherUpdate, DeleteInstance, GetCreateInstanceMinecraftVersions, GetCreateInstanceLoaderVersions, SetInstanceResourceEnabled, DeleteInstanceResource, ResolveInstanceResourceStoreLinks, OpenBrowserURL } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { AppSidebar } from "./components/app-sidebar";
import { ResourceStoreBrowser } from "./components/ResourceStoreBrowser";
import { SiteHeader } from "./components/site-header";
import {
  SidebarInset,
  SidebarProvider,
} from "./components/ui/sidebar";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Progress } from "@/components/ui/progress";
import { preloadTranslations, useTranslate } from "./hooks/use-translate";
import { ModeToggle } from "./components/mode-toggle";
import {
  ModSelectionDialog,
  type LaunchConfirmMeta,
} from "./components/ModSelectionDialog";
import { ServerModList, ModDetailDialog } from "./components/ServerModList";

const SkinPreview3d = lazy(() =>
  import("./components/SkinPreview3d").then((m) => ({ default: m.SkinPreview3d }))
);

const PROVIDER_URLS: Record<string, (username: string, uuid?: string) => string> = {
  ely_by: (u) => `https://skinsystem.ely.by/skins/${u || "Player"}.png`,
};

const PROVIDER_LABELS: Record<string, string> = {
  ely_by: "Ely.by",
};

function detectProviderFromUrl(url: string | null | undefined): string {
  if (!url) return "custom";
  if (url.includes("skinsystem.ely.by")) return "ely_by";
  return "custom";
}

interface ServerInfo {
  id: string;
  name: string;
  address: string;
  port: number;
  players: number;
  maxPlayers: number;
  version: string;
  online: boolean;
  /** From QMServer profile; false = administrator disabled the game server. */
  enabled?: boolean;
  /** false = Minecraft process did not respond to Server List Ping (QMServer ≥1.5.65). */
  gameServerOnline?: boolean;
  modLoader?: string;
  modLoaderVersion?: string;
  isPremium?: boolean;
  serverID?: number;
}

/** Minecraft process reachability for ping badge (legacy API: infer from player fields). */
function mcServerProcessPingOnline(server: ServerInfo): boolean {
  if (server.enabled === false) return false;
  if (server.gameServerOnline === true) return true;
  if (server.gameServerOnline === false) return false;
  return server.maxPlayers > 0 || server.players > 0;
}

function McServerPingBadge({
  server,
  onlineLabel,
  offlineLabel,
}: {
  server: ServerInfo;
  onlineLabel: string;
  offlineLabel: string;
}) {
  if (server.enabled === false) return null;
  const online = mcServerProcessPingOnline(server);
  return (
    <span
      className={
        online
          ? "text-xs px-2 py-0.5 rounded-md border border-emerald-500/50 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300 shrink-0 font-medium select-none"
          : "text-xs px-2 py-0.5 rounded-md border border-muted-foreground/40 text-muted-foreground shrink-0 font-medium select-none"
      }
    >
      {online ? onlineLabel : offlineLabel}
    </span>
  );
}

import { ThemeProvider } from "./components/theme-provider"
import { Toaster } from "@/components/ui/sonner"

interface AccountInfo {
  type: string;
  username: string;
  status: string;
  isDefault: boolean;
  skinModel?: string;
  skinUuid?: string;
  skinUrl?: string;
  email?: string;
  gameAccountId?: number;
}

/** Identities that can be chosen to launch the game (offline, QMServer Cloud game profile, or Microsoft license). */
function isLaunchableGameAccount(a: AccountInfo): boolean {
  if (a.type === "local" || a.type === "cloud_game") return true;
  return a.type === "microsoft" && Boolean(a.username?.trim());
}

const launchAccountTypePriority: Record<string, number> = { microsoft: 3, cloud_game: 2, local: 1 };

/** One row per username for the account picker; if names collide, Microsoft wins over cloud_game over local (matches launch resolution order). */
function dedupeLaunchableAccounts(accs: AccountInfo[]): AccountInfo[] {
  const byUser = new Map<string, AccountInfo>();
  for (const a of accs) {
    if (!isLaunchableGameAccount(a)) continue;
    const u = a.username;
    const cur = byUser.get(u);
    const pr = launchAccountTypePriority[a.type] ?? 0;
    if (!cur || pr > (launchAccountTypePriority[cur.type] ?? 0)) {
      byUser.set(u, a);
    }
  }
  return [...byUser.values()];
}

function normalizeMinecraftUuid(u: string | undefined): string {
  if (!u?.trim()) return "";
  return u.replace(/-/g, "").toLowerCase();
}

/** Microsoft row: hide "Link to QMServer Cloud" when this profile is already a cloud_game (same Mojang UUID, or same Minecraft username). */
function isMicrosoftLinkedToCloudGame(accounts: AccountInfo[]): boolean {
  const ms = accounts.find((a) => a.type === "microsoft");
  if (!ms) return false;
  const needle = normalizeMinecraftUuid(ms.skinUuid);
  const msUser = ms.username?.trim().toLowerCase() ?? "";
  if (needle) {
    const byUuid = accounts.some(
      (a) =>
        a.type === "cloud_game" && normalizeMinecraftUuid(a.skinUuid) === needle
    );
    if (byUuid) return true;
  }
  if (msUser) {
    return accounts.some(
      (a) =>
        a.type === "cloud_game" &&
        a.username?.trim().toLowerCase() === msUser
    );
  }
  return false;
}

function normalizeQMServerErrorMessage(msg: string): string {
  const m = (msg || "").trim();
  if (!m) return "";
  const lower = m.toLowerCase();
  if (lower.includes("does not serve qmlauncher") || lower.includes("qmlauncher integration is disabled")) {
    return "Выбранный QMServer не обслуживает интеграцию QMLauncher.";
  }
  return m;
}

function getAccountSkinUrl(account: AccountInfo): string {
  if (account.skinUuid) {
    return `https://mc-heads.net/avatar/${account.skinUuid}/32`;
  }
  if (account.skinUrl?.trim()) {
    return account.skinUrl.trim();
  }
  const model = account.skinModel === "alex" ? "Alex" : "Steve";
  return `https://mc-heads.net/avatar/MHF_${model}/32`;
}

/** True if URL is a skin texture (Ely.by etc) — needs crop to show head (Minecraft 64x64: head at 8,8-16,16) */
function isSkinTextureUrl(url: string): boolean {
  return !url.includes("mc-heads.net") && (url.includes("skinsystem.ely.by") || url.includes("/skins/") || /\.png$/i.test(url));
}

/** Minecraft 64x64 skin: head at 8,8-16,16 — background-image crop to show head only */
function skinTextureHeadStyle(url: string, sizePx: number = 32): CSSProperties {
  const scale = sizePx / 8; // head is 8x8, scale to display size
  const bgSize = 64 * scale;
  const offset = 8 * scale;
  return {
    backgroundImage: `url(${url})`,
    backgroundSize: `${bgSize}px ${bgSize}px`,
    backgroundPosition: `-${offset}px -${offset}px`,
    imageRendering: "pixelated",
  };
}

/** Proxied URL for skin/avatar — avoids CORS and hotlinking (uses QMServer proxy) */
function proxiedUrl(rawUrl: string, apiBase: string): string {
  if (!rawUrl?.trim() || !apiBase) return rawUrl || "";
  return `${apiBase}/skins/proxy?url=${encodeURIComponent(rawUrl.trim())}`;
}

/** Full 64x64 skin URL for SkinPreview3d when editing a cloud game account (default Steve/Alex). */
function getCloudSkinEditPreviewUrl(
  acc: { username: string; skinUrl: string },
  editSkinUrl: string,
  editProvider: string,
  editModel: "steve" | "alex",
): string {
  const fromForm =
    editSkinUrl.trim() || (editProvider === "ely_by" ? PROVIDER_URLS.ely_by(acc.username) : "");
  if (fromForm) return fromForm;
  const model = editModel === "alex" ? "Alex" : "Steve";
  return `https://mc-heads.net/skin/MHF_${model}`;
}

/** Статика из `frontend/public/` → в сборке URL с корня (встраивается в Wails вместе с `frontend/dist`). Замените файл на свой PNG/WebP и при смене имени обновите путь. */
const INSTANCE_CARD_COVER_PLACEHOLDER = "/assets/instance-placeholder.png";

/** Показ названия загрузчика в UI (Vanilla, Fabric, NeoForge, …). */
const LOADER_DISPLAY_NAME: Record<string, string> = {
  vanilla: "Vanilla",
  fabric: "Fabric",
  quilt: "Quilt",
  forge: "Forge",
  neoforge: "NeoForge",
};

function formatLoaderDisplayName(loader: string): string {
  const raw = (loader || "").trim();
  if (!raw) return raw;
  const key = raw.toLowerCase();
  if (LOADER_DISPLAY_NAME[key]) return LOADER_DISPLAY_NAME[key];
  return raw.charAt(0).toUpperCase() + raw.slice(1);
}

function formatLoaderLabel(loader: string, loaderVersion?: string): string {
  const raw = loader || "vanilla";
  if (raw.toLowerCase() === "vanilla") return "Vanilla";
  const name = formatLoaderDisplayName(raw);
  const ver = loaderVersion?.trim();
  return ver ? `${name} ${ver}` : name;
}

function isModdedInstanceLoader(loader: string): boolean {
  return !!loader && loader.toLowerCase() !== "vanilla";
}

/** Wails / JSON may yield null; non-array breaks React when calling .map → white screen. */
function normalizeInstancesList(data: unknown): any[] {
  return Array.isArray(data) ? data : [];
}

const RESOURCE_DISABLED_SUFFIX = ".disabled";

/** mods/*.jar и mods/*.jar.disabled */
function isInstanceModJarFile(filename: string): boolean {
  const l = filename.toLowerCase();
  return l.endsWith(".jar.disabled") || (l.endsWith(".jar") && !l.endsWith(".jar.disabled"));
}

function resourceEntryEnabled(storageKey: string): boolean {
  return !storageKey.toLowerCase().endsWith(RESOURCE_DISABLED_SUFFIX);
}

function resourceStripDisabledKey(storageKey: string): string {
  return resourceEntryEnabled(storageKey) ? storageKey : storageKey.slice(0, -RESOURCE_DISABLED_SUFFIX.length);
}

function parseNameVersionStem(stem: string): { title: string; version: string } {
  const m = stem.match(/^(.+)-(\d[\w.\-+]*)$/);
  if (m) return { title: m[1].replace(/-/g, " "), version: m[2] };
  return { title: stem.replace(/-/g, " "), version: "—" };
}

/** Trailing semver-like chunk: "-1.2.3", "-v1.2.3", pre-release tail (NeoForge/Gradle). */
const RE_ZIP_RESOURCE_VERSION_TAIL =
  /[-_.](?:(?:v\d+(?:\.\d+)*)|(?:\d+(?:\.\d+)*))(?:[-+_.][+_.a-zA-Z0-9-]*)*$/i;

/** Strip "-neoforge-1.20" style segments before a version (same idea as QMServer humanDisplayName). */
function stripLoaderSegmentBeforeVersionZipStem(stem: string): string {
  let t = stem;
  for (let i = 0; i < 8; i++) {
    const n = t
      .replace(/-(?:neoforge|forge|fabric|quilt)-((?:mc)?[0-9]\S*)/gi, "-$1")
      .replace(/_(?:neoforge|forge|fabric|quilt)_((?:mc)?[0-9]\S*)/gi, "_$1");
    if (n === t) break;
    t = n;
  }
  return t;
}

/** Remove repeated version tails, optional leading MC version prefix, lone "v" in title. */
function parseZipLikeResourceStem(stem: string): { title: string; version: string } {
  let s = stripLoaderSegmentBeforeVersionZipStem(stem.trim());
  const versions: string[] = [];
  for (let i = 0; i < 24; i++) {
    const m = s.match(RE_ZIP_RESOURCE_VERSION_TAIL);
    if (!m) break;
    const chunk = m[0];
    versions.unshift(chunk.replace(/^[-_.]/, ""));
    s = s.slice(0, -chunk.length).replace(/[-_.]+$/g, "");
    if (!s) break;
  }
  let base = s;
  base = base.replace(/^(?:\d+\.\d+(?:\.\d+)?)(?:[-+_.][-a-zA-Z0-9.+]*)?[-_.]?/i, "").trim();
  base = base.replace(/^v[-_.\s]+/i, "").trim();
  base = base.replace(/[-_.]+v$/i, "").trim();
  let title = base.replace(/[-_]+/g, " ").replace(/\s+/g, " ").trim();
  title = title.replace(/\s+v$/i, "").trim();
  title = title.replace(/^v\s+/i, "").trim();
  if (!title) {
    title = stem.replace(/[-_]+/g, " ").replace(/\s+/g, " ").trim();
  }
  const version = versions.length ? versions.join(" · ") : "—";
  return { title, version };
}

function parseInstanceModRow(storageName: string): { title: string; version: string; enabled: boolean } {
  const enabled = resourceEntryEnabled(storageName);
  const baseJar = resourceStripDisabledKey(storageName);
  const stem = baseJar.replace(/\.jar$/i, "");
  const { title, version } = parseNameVersionStem(stem);
  return { title, version, enabled };
}

/** resourcepacks / shaderpacks: zip/mcpack + .disabled */
function parseInstanceZipPackRow(storageName: string): { title: string; version: string; enabled: boolean } {
  const enabled = resourceEntryEnabled(storageName);
  const base = resourceStripDisabledKey(storageName);
  const stem = base.replace(/\.(zip|mcpack)$/i, "");
  const { title, version } = parseZipLikeResourceStem(stem);
  return { title, version, enabled };
}

const MODPACK_MARKER_LABELS: Record<string, string> = {
  "manifest.json": "CurseForge (manifest.json)",
  "modrinth.index.json": "Modrinth (modrinth.index.json)",
  "pack.toml": "Packwiz (pack.toml)",
  "minecraftinstance.json": "GDLauncher (minecraftinstance.json)",
};

function parseInstanceModpackMarkerRow(storageName: string): { title: string; version: string; enabled: boolean } {
  const enabled = resourceEntryEnabled(storageName);
  const base = resourceStripDisabledKey(storageName);
  const key = base.toLowerCase();
  const title = MODPACK_MARKER_LABELS[key] ?? base;
  return { title, version: "—", enabled };
}

function parseInstanceDatapackRow(relPath: string): { title: string; version: string; enabled: boolean } {
  const enabled = resourceEntryEnabled(relPath);
  const active = resourceStripDisabledKey(relPath);
  const base = active.split("/").pop() || active;
  const stem = base.replace(/\.(zip|mcpack)$/i, "");
  const { title, version } = parseNameVersionStem(stem);
  return { title, version, enabled };
}

function instanceResourceBusyKey(category: string, resourcePath: string): string {
  return `${category}:${resourcePath}`;
}

type InstanceResourcesSubTab = "modpacks" | "mods" | "datapacks" | "resourcepacks" | "shaderpacks";

/** Ключ как в QMLauncher remote-installs.json: "category/basename" (.disabled снят). */
function remoteInstallRowKey(category: InstanceResourcesSubTab, storagePath: string): string {
  const base = storagePath.split("/").pop() ?? storagePath;
  const active = resourceStripDisabledKey(base);
  return `${category}/${active}`;
}

/** Иконка строки в таблице ресурсов (всегда перед названием). */
function getInstanceResourceRowIcon(category: InstanceResourcesSubTab, storagePath: string): LucideIcon {
  const baseName = resourceStripDisabledKey(storagePath.split("/").pop() || storagePath);
  const lower = baseName.toLowerCase();

  switch (category) {
    case "mods":
      return Package;
    case "resourcepacks":
      return Image;
    case "shaderpacks":
      return Sparkles;
    case "datapacks":
      if (lower.endsWith(".zip") || lower.endsWith(".mcpack")) {
        return Archive;
      }
      return Database;
    case "modpacks":
      if (lower === "pack.toml") {
        return FileText;
      }
      if (lower.endsWith(".json")) {
        return FileJson;
      }
      return Boxes;
    default:
      return Box;
  }
}

/** Row thumbnail: catalog icon when remote-installs has iconUrl, else Lucide glyph. */
function InstanceResourceThumb({
  iconUrl,
  RowIcon,
}: {
  iconUrl?: string;
  RowIcon: LucideIcon;
}) {
  const [broken, setBroken] = useState(false);
  const u = iconUrl?.trim();
  if (!u || broken) {
    return <RowIcon className="w-[18px] h-[18px] text-muted-foreground" strokeWidth={1.75} />;
  }
  return (
    <img
      src={u}
      alt=""
      className="w-[18px] h-[18px] rounded object-cover"
      loading="lazy"
      referrerPolicy="no-referrer"
      onError={() => setBroken(true)}
    />
  );
}

type AppMainTab =
  | "news"
  | "servers"
  | "instances"
  | "instance"
  | "instance-settings"
  | "instance-resources"
  | "resource-store"
  | "accounts"
  | "game-accounts"
  | "settings";

function parseLocationHash(hash: string): {
  tab: AppMainTab;
  instanceNameFromRoute?: string;
  resourceStoreCategory?: InstanceResourcesSubTab;
} {
  const h = hash.replace(/^#/, "").trim();
  if (!h) return { tab: "news" };
  if (h.startsWith("instance-settings/")) {
    const enc = h.slice("instance-settings/".length);
    if (!enc) return { tab: "instances" };
    try {
      return { tab: "instance-settings", instanceNameFromRoute: decodeURIComponent(enc) };
    } catch {
      return { tab: "instances" };
    }
  }
  if (h.startsWith("resource-store/")) {
    const rest = h.slice("resource-store/".length);
    const slash = rest.indexOf("/");
    if (slash <= 0) return { tab: "instances" };
    const encInst = rest.slice(0, slash);
    const cat = rest.slice(slash + 1);
    try {
      const instanceNameFromRoute = decodeURIComponent(encInst);
      const allowed: InstanceResourcesSubTab[] = [
        "modpacks",
        "mods",
        "datapacks",
        "resourcepacks",
        "shaderpacks",
      ];
      const resourceStoreCategory = (allowed.includes(cat as InstanceResourcesSubTab)
        ? cat
        : "mods") as InstanceResourcesSubTab;
      return { tab: "resource-store", instanceNameFromRoute, resourceStoreCategory };
    } catch {
      return { tab: "instances" };
    }
  }
  if (h.startsWith("instance-resources/")) {
    const enc = h.slice("instance-resources/".length);
    if (!enc) return { tab: "instances" };
    try {
      return { tab: "instance-resources", instanceNameFromRoute: decodeURIComponent(enc) };
    } catch {
      return { tab: "instances" };
    }
  }
  const base = h.split("/")[0];
  const allowed: AppMainTab[] = [
    "news",
    "servers",
    "instances",
    "instance",
    "accounts",
    "game-accounts",
    "settings",
  ];
  if (allowed.includes(base as AppMainTab)) {
    return { tab: base as AppMainTab };
  }
  return { tab: "news" };
}

function InstanceMemoryControls({
  instanceName,
  minMemoryMB,
  maxMemoryMB,
  showAlert,
  onAfterSave,
}: {
  instanceName: string;
  minMemoryMB: number;
  maxMemoryMB: number;
  showAlert: (title: string, message: string) => void;
  onAfterSave: () => void;
}) {
  const clampPair = (rawMin: number, rawMax: number) => {
    const mn = Math.max(128, Math.min(32768, rawMin));
    const mx = Math.max(512, Math.min(32768, rawMax));
    return { finalMin: Math.min(mn, mx), finalMax: Math.max(mn, mx) };
  };

  const normFromProps = () => {
    const mn = Math.max(128, Math.min(32768, minMemoryMB));
    const mx = Math.max(512, Math.min(32768, Math.max(mn, maxMemoryMB)));
    return { mn, mx };
  };

  const { mn: initMin, mx: initMax } = normFromProps();
  const [minV, setMinV] = useState(initMin);
  const [maxV, setMaxV] = useState(initMax);
  const minRef = useRef(initMin);
  const maxRef = useRef(initMax);
  const committedRef = useRef({ min: initMin, max: initMax });
  const draggingRef = useRef(false);

  useEffect(() => {
    const { mn, mx } = normFromProps();
    committedRef.current = { min: mn, max: mx };
    minRef.current = mn;
    maxRef.current = mx;
    setMinV(mn);
    setMaxV(mx);
  }, [instanceName, minMemoryMB, maxMemoryMB]);

  const flush = () => {
    const { finalMin, finalMax } = clampPair(minRef.current, maxRef.current);
    const c = committedRef.current;
    if (finalMin === c.min && finalMax === c.max) return;
    SetInstanceMemory(instanceName, finalMin, finalMax).then((err) => {
      if (err) showAlert("Ошибка", err);
      else {
        committedRef.current = { min: finalMin, max: finalMax };
        minRef.current = finalMin;
        maxRef.current = finalMax;
        setMinV(finalMin);
        setMaxV(finalMax);
        onAfterSave();
      }
    });
  };

  const MemRow = ({
    label,
    value,
    which,
  }: {
    label: string;
    value: number;
    which: "min" | "max";
  }) => (
    <div className="space-y-2">
      <Label className="text-sm text-muted-foreground">{label}</Label>
      <div className="flex items-center gap-3">
        <input
          type="range"
          min={128}
          max={16384}
          step={512}
          value={value}
          onPointerDown={() => {
            draggingRef.current = true;
            const onWindowUp = () => {
              window.removeEventListener("pointerup", onWindowUp);
              window.removeEventListener("pointercancel", onWindowUp);
              const wasDragging = draggingRef.current;
              draggingRef.current = false;
              if (wasDragging) flush();
            };
            window.addEventListener("pointerup", onWindowUp);
            window.addEventListener("pointercancel", onWindowUp);
          }}
          onChange={(e) => {
            const v = parseInt(e.target.value, 10);
            if (Number.isNaN(v)) return;
            if (which === "min") {
              minRef.current = v;
              setMinV(v);
            } else {
              maxRef.current = v;
              setMaxV(v);
            }
            if (!draggingRef.current) {
              queueMicrotask(flush);
            }
          }}
          className="flex-1 h-2 rounded-lg appearance-none bg-muted cursor-pointer accent-primary"
        />
        <Input
          type="number"
          min={128}
          max={32768}
          step={512}
          value={value}
          className="w-20 h-9"
          onChange={(e) => {
            const v = parseInt(e.target.value, 10);
            if (Number.isNaN(v)) return;
            const clamped = Math.max(128, Math.min(32768, v));
            if (which === "min") {
              minRef.current = clamped;
              setMinV(clamped);
            } else {
              maxRef.current = clamped;
              setMaxV(clamped);
            }
          }}
          onBlur={flush}
          onKeyDown={(e) => {
            if (e.key === "Enter") flush();
          }}
        />
        <span className="text-sm text-muted-foreground w-12">{(value / 1024).toFixed(1)} ГБ</span>
      </div>
    </div>
  );

  return (
    <div className="space-y-4">
      <MemRow label="-Xms (мин. ОЗУ)" value={minV} which="min" />
      <MemRow label="-Xmx (макс. ОЗУ)" value={maxV} which="max" />
    </div>
  );
}

function curseForgeEffectiveSourceLabel(src: string): string {
  switch (src) {
    case "env":
      return "переменная окружения CURSEFORGE_API_KEY";
    case "file":
      return "файл ~/.qmlauncher/settings.json";
    case "cloud":
      return "QMServer (облако)";
    default:
      return "не задан";
  }
}

function App() {
  const [activeTab, setActiveTab] = useState<AppMainTab>(() => parseLocationHash(window.location.hash).tab);
  const [resourceStoreCategory, setResourceStoreCategory] = useState<InstanceResourcesSubTab>(
    () => parseLocationHash(window.location.hash).resourceStoreCategory ?? "mods"
  );
  const [servers, setServers] = useState<ServerInfo[]>([]);
  const [serversLoadError, setServersLoadError] = useState<string>("");
  const [instances, setInstances] = useState<any[]>([]);
  const [accounts, setAccounts] = useState<AccountInfo[]>([]);
  const launchableAccounts = useMemo(
    () => dedupeLaunchableAccounts(accounts),
    [accounts]
  );
  const [microsoftAuthAvailable, setMicrosoftAuthAvailable] = useState(false);
  const [currentAccount, setCurrentAccount] = useState<{ name: string; email: string }>({ name: "User", email: "user@qmlauncher.local" });
  const [currentAccountType, setCurrentAccountType] = useState<string>("");
  const [cloudProfileAvatar, setCloudProfileAvatar] = useState<string | undefined>(undefined);
  const [cloudIsPremium, setCloudIsPremium] = useState(false);

  const applySidebarForCloudAccount = (account: any | null | undefined, opts?: { requireDefault?: boolean; emailFallback?: string }) => {
    const requireDefault = opts?.requireDefault ?? false;
    const emailFallback = opts?.emailFallback ?? "account@qmlauncher.local";
    const ok = account?.name && (!requireDefault || account.isDefault);
    if (ok) {
      setCurrentAccount({
        name: account.name,
        email: account.email || emailFallback,
      });
      setCurrentAccountType(account.type || "");
      if (account.type === "cloud") {
        GetCloudProfile()
          .then((profile: any) => {
            if (profile?.avatar_url) setCloudProfileAvatar(profile.avatar_url);
            else setCloudProfileAvatar(undefined);
            setCloudIsPremium(!!profile?.is_premium);
          })
          .catch(() => {
            setCloudProfileAvatar(undefined);
            setCloudIsPremium(false);
          });
      } else {
        setCloudProfileAvatar(undefined);
        setCloudIsPremium(false);
      }
    } else {
      setCurrentAccount({ name: "User", email: "user@qmlauncher.local" });
      setCurrentAccountType("");
      setCloudProfileAvatar(undefined);
      setCloudIsPremium(false);
    }
  };
  const [launcherVersion, setLauncherVersion] = useState("");
  const [_translationsLoaded, setTranslationsLoaded] = useState(false);
  const [showNoAccountDialog, setShowNoAccountDialog] = useState(false);
  const [showCreateAccountDialog, setShowCreateAccountDialog] = useState(false);
  const [newAccountName, setNewAccountName] = useState("");
  const [newAccountSkinModel, setNewAccountSkinModel] = useState<"steve" | "alex">("steve");
  const [showLaunchProgressDialog, setShowLaunchProgressDialog] = useState(false);
  const [launchProgress, setLaunchProgress] = useState<{
    message: string;
    progress?: number;
    type?: string;
    phase?: string;
    currentFile?: string;
  }>({ message: "" });

  const [syncConfigFromServer, setSyncConfigFromServer] = useState(false);
  const [selectedServerDetails, setSelectedServerDetails] = useState<ServerInfo | null>(null);
  const [selectedModDetail, setSelectedModDetail] = useState<{ path: string; name: string; meta?: { description?: string; icon_url?: string; curseforge_url?: string; modrinth_url?: string } } | null>(null);
  const [selectedInstanceDetails, setSelectedInstanceDetails] = useState<any | null>(null);
  const [instanceDetails, setInstanceDetails] = useState<any | null>(null);
  const [selectedInstanceName, setSelectedInstanceName] = useState<string>(
    () => parseLocationHash(window.location.hash).instanceNameFromRoute ?? ""
  );

  // Confirm/Alert dialogs (replace browser confirm/alert)
  const [showDeleteAccountDialog, setShowDeleteAccountDialog] = useState(false);
  const [accountToDelete, setAccountToDelete] = useState<AccountInfo | null>(null);
  const [showDeleteInstanceDialog, setShowDeleteInstanceDialog] = useState(false);
  const [instanceNameToDelete, setInstanceNameToDelete] = useState<string | null>(null);
  const [instanceResourcesSubTab, setInstanceResourcesSubTab] = useState<InstanceResourcesSubTab>("mods");
  const [instanceResourceBusy, setInstanceResourceBusy] = useState<string | null>(null);
  const [showSyncSkinDialog, setShowSyncSkinDialog] = useState<AccountInfo | null>(null);
  const [syncSkinUrl, setSyncSkinUrl] = useState("");
  const [showEditCloudSkinDialog, setShowEditCloudSkinDialog] = useState(false);
  const [cloudGameAccounts, setCloudGameAccounts] = useState<{ id: number; username: string; uuid: string; skinModel: string; skinUrl: string }[]>([]);
  const [editCloudAccountId, setEditCloudAccountId] = useState<number | null>(null);
  const [editCloudSkinUrl, setEditCloudSkinUrl] = useState("");
  const [editCloudSkinModel, setEditCloudSkinModel] = useState<"steve" | "alex">("steve");
  const [skinProviders, setSkinProviders] = useState<Record<string, boolean>>({ ely_by: true });
  const [_cloudElyLinked, setCloudElyLinked] = useState(false);
  const [syncSkinProvider, setSyncSkinProvider] = useState("");
  const [editCloudSkinProvider, setEditCloudSkinProvider] = useState("");
  const [news, setNews] = useState<{ id: number; title: string; content: string; createdAt: string }[]>([]);
  const [showLogoutConfirmDialog, setShowLogoutConfirmDialog] = useState(false);
  const [showAlertDialog, setShowAlertDialog] = useState(false);
  const [alertContent, setAlertContent] = useState<{ title: string; message: string } | null>(null);
  const [showUpdateDialog, setShowUpdateDialog] = useState(false);
  const [updateApplying, setUpdateApplying] = useState(false);
  const [showAboutDialog, setShowAboutDialog] = useState(false);
  const [aboutInfo, setAboutInfo] = useState<{ version: string; os: string; arch: string } | null>(null);
  const [aboutLoading, setAboutLoading] = useState(false);
  const [aboutCheckLoading, setAboutCheckLoading] = useState(false);
  const [apiBase, setApiBase] = useState<string>("");
  const [launcherApiUseCloud, setLauncherApiUseCloud] = useState(true);
  const [launcherApiCustom, setLauncherApiCustom] = useState("");
  const [launcherApiEffective, setLauncherApiEffective] = useState("");
  const [launcherApiSaving, setLauncherApiSaving] = useState(false);
  const [launcherDebug, setLauncherDebug] = useState(false);
  const [launcherDebugSaving, setLauncherDebugSaving] = useState(false);
  const [curseForgeKeyInput, setCurseForgeKeyInput] = useState("");
  const [curseForgeKeySaving, setCurseForgeKeySaving] = useState(false);
  const [curseForgeKeySavedInFile, setCurseForgeKeySavedInFile] = useState(false);
  const [curseForgeUseMyKeyDefault, setCurseForgeUseMyKeyDefault] = useState(false);
  const [curseForgeEffectiveSource, setCurseForgeEffectiveSource] = useState("none");
  const [catalogCurseforgeEnabled, setCatalogCurseforgeEnabled] = useState(true);
  const [catalogModrinthEnabled, setCatalogModrinthEnabled] = useState(true);
  const [currentLang, setCurrentLang] = useState<string>("ru");

  // Диалог выбора типа аккаунта для создания
  const [showAccountTypeDialog, setShowAccountTypeDialog] = useState(false);

  // Диалог выбора модов перед запуском (при подключении к серверу)
  const [showModSelectionDialog, setShowModSelectionDialog] = useState(false);
  const [pendingLaunch, setPendingLaunch] = useState<{
    instanceName: string;
    serverAddress: string;
    serverID: number;
    serverName?: string;
    syncConfigFromServer: boolean;
    selectedAccountName: string;
  } | null>(null);

  // Диалог создания игрового аккаунта QMServer Cloud
  const [showCloudGameAccountDialog, setShowCloudGameAccountDialog] = useState(false);
  const [cloudGameAccountName, setCloudGameAccountName] = useState("");
  const [showInventoryDialog, setShowInventoryDialog] = useState(false);
  const [inventoryAccount, setInventoryAccount] = useState<{ username: string; gameAccountId: number } | null>(null);
  const [showAccountSummaryDialog, setShowAccountSummaryDialog] = useState(false);
  const [accountSummaryTarget, setAccountSummaryTarget] = useState<{ type: string; username?: string; email?: string } | null>(null);
  const [showCreateInstanceDialog, setShowCreateInstanceDialog] = useState(false);
  const [createInstanceName, setCreateInstanceName] = useState("");
  const [createInstanceVersion, setCreateInstanceVersion] = useState("");
  const [createInstanceLoader, setCreateInstanceLoader] = useState("vanilla");
  const [createInstanceLoaderVersion, setCreateInstanceLoaderVersion] = useState("");
  const [createInstanceMcVersions, setCreateInstanceMcVersions] = useState<string[]>([]);
  const [createInstanceMcLoading, setCreateInstanceMcLoading] = useState(false);
  const [createInstanceLoaderVersions, setCreateInstanceLoaderVersions] = useState<string[]>([]);
  const [createInstanceLoaderVersionsLoading, setCreateInstanceLoaderVersionsLoading] = useState(false);
  const [createInstanceSubmitting, setCreateInstanceSubmitting] = useState(false);
  const [inventoryData, setInventoryData] = useState<{ inventories: Array<{
    server_id: string; server_name?: string; server_version?: string; player_name: string; timestamp: number;
    main: Array<{ slot: number; item: string; count: number }>;
    armor: Array<{ slot: number; item: string; count: number }>;
    offhand?: { slot: number; item: string; count: number };
  }>; error?: string } | null>(null);
  const [cloudGameAccountSkinModel, setCloudGameAccountSkinModel] = useState<"steve" | "alex">("steve");

  const showAlert = (title: string, message: string) => {
    setAlertContent({ title, message });
    setShowAlertDialog(true);
  };

  const persistCatalogStoreSettings = async (nextCf: boolean, nextMr: boolean) => {
    const err = await SetCatalogStoreSettings(nextCf, nextMr);
    if (err) {
      showAlert("Каталоги", err);
      return;
    }
    setCatalogCurseforgeEnabled(nextCf);
    setCatalogModrinthEnabled(nextMr);
  };

  const aboutAppTitle = useTranslate("ui.about_app");
  const checkUpdatesLabel = useTranslate("ui.check_updates");

  useEffect(() => {
    if (!showAboutDialog) {
      setAboutInfo(null);
      return;
    }
    setAboutLoading(true);
    void GetLauncherAboutInfo()
      .then((info) => {
        setAboutInfo({
          version: info?.version ?? "",
          os: info?.os ?? "",
          arch: info?.arch ?? "",
        });
      })
      .catch(() => setAboutInfo(null))
      .finally(() => setAboutLoading(false));
  }, [showAboutDialog]);

  // Use hooks for translations at top level
  const rawNewsTitle = useTranslate("ui.news");
  const rawServersTitle = useTranslate("ui.servers");
  const rawInstancesTitle = useTranslate("ui.instances");
  const rawAccountTitle = useTranslate("ui.account");
  const rawGameAccountsTitle = useTranslate("ui.game_accounts");

  const t = {
    // Убираем emoji из заголовков, если они есть
    news: rawNewsTitle.replace(/^📰\s*/, ""),
    servers: rawServersTitle.replace(/^🌐\s*/, ""),
    instances: rawInstancesTitle.replace(/^📦\s*/, ""),
    account: rawAccountTitle.replace(/^👤\s*/, ""),
    gameAccounts: rawGameAccountsTitle.replace(/^👤\s*/, ""),
    serversSelect: useTranslate("ui.recent_servers.select"),
    serversNotFound: useTranslate("servers.not_found"),
    serversTableName: useTranslate("ui.recent_servers.table.name"),
    serversTableIp: useTranslate("ui.recent_servers.table.ip"),
    serversTableStatus: useTranslate("ui.auth.table.status"),
    serversTablePlayers: useTranslate("ui.table.players"),
    serversTableVersion: useTranslate("search.table.version"),
    serversConnect: useTranslate("ui.server.connect"),
    serverDisabledError: useTranslate("ui.server.disabled_error"),
    serverDisabledShort: useTranslate("ui.server.disabled_short"),
    gameProcessOffline: useTranslate("ui.game_server.process_offline"),
    gameProcessOfflineDetail: useTranslate("ui.game_server.process_offline_detail"),
    badgeOnline: useTranslate("ui.game_server.badge_online"),
    badgeOffline: useTranslate("ui.game_server.badge_offline"),
    premium: useTranslate("ui.premium"),
    statusOnline: useTranslate("ui.status.online"),
    statusOffline: useTranslate("ui.status.offline"),
    instance: useTranslate("instance"),
    instancesEmptyTitle: useTranslate("ui.instances.empty_title"),
    instancesEmptyHint: useTranslate("ui.instances.empty_hint"),
    createInstance: useTranslate("ui.create_instance"),
    modloader: useTranslate("ui.form.modloader"),
    launchInstance: useTranslate("ui.instance.launch"),
    settings: useTranslate("ui.settings"),
    settingsMenu: useTranslate("ui.settings_menu"),
    settingsLanguage: useTranslate("ui.settings_language"),
    settingsTheme: useTranslate("ui.settings_theme"),
    settingsDebug: useTranslate("ui.settings_debug"),
    settingsDebugDesc: useTranslate("ui.settings_debug_desc"),
    settingsTabGeneral: useTranslate("ui.settings.tab.general"),
    settingsTabMinecraft: useTranslate("ui.settings.tab.minecraft"),
    loading: useTranslate("ui.loading"),
    logout: useTranslate("ui.logout"),
    authMenu: useTranslate("ui.auth_menu"),
    authLogin: useTranslate("ui.auth_login"),
    authLogout: useTranslate("ui.auth_logout"),
    authCreate: useTranslate("ui.auth_create"),
    authAddCloud: useTranslate("ui.auth_add_cloud"),
    authDelete: useTranslate("ui.auth_delete"),
    authTableType: useTranslate("ui.auth.table.type"),
    authTableUsername: useTranslate("ui.auth.table.username"),
    authTableStatus: useTranslate("ui.auth.table.status"),
    authTableActions: useTranslate("ui.auth.table.actions"),
    authStatusConnected: useTranslate("ui.auth.status.connected"),
    authStatusExpired: useTranslate("ui.auth.status.expired"),
    authStatusExpiredTooltip: useTranslate("ui.auth.status.expired_tooltip"),
    authConnectedAccounts: useTranslate("ui.auth.connected_accounts"),
    authSyncMicrosoftCloudBtn: useTranslate("ui.auth.sync_ms_cloud_btn"),
    authSyncMicrosoftCloudLinked: useTranslate("ui.auth.sync_ms_cloud_linked"),
    authSyncMicrosoftCloudResultTitle: useTranslate("ui.auth.sync_ms_cloud_result_title"),
    authMicrosoft: useTranslate("ui.auth.mojang_ms"),
    authLocal: useTranslate("ui.auth.local_status"),
    authCloud: "QMServer Cloud",
    statusExpired: useTranslate("ui.status.expired"),
    statusNone: useTranslate("ui.status.none"),
    statusDefault: useTranslate("ui.status.default"),
    noAccountTitle: useTranslate("ui.no_account.title"),
    noAccountDescription: useTranslate("ui.no_account.description"),
    noAccountAdd: useTranslate("ui.no_account.add"),
    noAccountCancel: useTranslate("ui.no_account.cancel"),
    createAccountTitle: useTranslate("ui.create_account.title"),
    createAccountDescription: useTranslate("ui.create_account.description"),
    createAccountSkinModel: useTranslate("ui.create_account.skin_model"),
    createAccountSkinSteve: useTranslate("ui.create_account.skin_steve"),
    createAccountSkinAlex: useTranslate("ui.create_account.skin_alex"),
    createAccountCreate: useTranslate("ui.create_account.create"),
  };

  // Preload all translations
  useEffect(() => {
    const translationKeys = [
      "ui.news",
      "ui.servers",
      "ui.instances",
      "ui.recent_servers.select",
      "ui.recent_servers.table.name",
      "ui.recent_servers.table.ip",
      "ui.auth.table.status",
      "ui.table.players",
      "search.table.version",
      "ui.server.connect",
      "ui.server.disabled_error",
      "ui.server.disabled_short",
      "ui.game_server.process_offline",
      "ui.game_server.process_offline_detail",
      "servers.not_found",
      "ui.premium",
      "ui.cloud_premium",
      "ui.status.online",
      "ui.status.offline",
      "ui.status.expired",
      "ui.status.none",
      "ui.status.default",
      "instance",
      "ui.instances.empty_title",
      "ui.instances.empty_hint",
      "ui.create_instance",
      "ui.form.modloader",
      "ui.instance.launch",
      "ui.settings",
      "ui.settings_menu",
      "ui.settings_language",
      "ui.settings_theme",
      "ui.settings_debug",
      "ui.settings_debug_desc",
      "ui.settings.tab.general",
      "ui.settings.tab.minecraft",
      "ui.loading",
      "ui.account",
      "ui.game_accounts",
      "ui.logout",
      "ui.auth_menu",
      "ui.auth_login",
      "ui.auth_logout",
      "ui.auth_create",
      "ui.auth_add_cloud",
      "ui.auth_delete",
      "ui.auth.table.type",
      "ui.auth.table.username",
      "ui.auth.table.status",
      "ui.auth.table.actions",
      "ui.auth.status.connected",
      "ui.auth.status.expired",
      "ui.auth.status.expired_tooltip",
      "ui.auth.connected_accounts",
      "ui.auth.sync_ms_cloud_btn",
      "ui.auth.sync_ms_cloud_linked",
      "ui.auth.sync_ms_cloud_result_title",
      "ui.auth.mojang_ms",
      "ui.auth.local_status",
      "ui.create_account.skin_model",
      "ui.create_account.skin_steve",
      "ui.create_account.skin_alex",
      "ui.about_app",
      "ui.check_updates",
    ];

    preloadTranslations(translationKeys).then(() => {
      setTranslationsLoaded(true);
    });

    GetInstances()
      .then((raw) => setInstances(normalizeInstancesList(raw)))
      .catch(() => setInstances([]));
    GetRecentServers()
      .then((list) => {
        setServers(list);
        if (list.length === 0) {
          GetQMServersError().then((err) => setServersLoadError(normalizeQMServerErrorMessage(err || "")));
        } else {
          setServersLoadError("");
        }
      })
      .catch((error) => {
        console.error('Failed to fetch servers:', error);
        setServers([]);
        setServersLoadError(normalizeQMServerErrorMessage(String(error)));
      });
    GetNews().then((items) => {
      const list = Array.isArray(items) ? items : [];
      setNews(list.map((n: any, i: number) => ({
        id: n?.id ?? i,
        title: n?.title ?? "",
        content: n?.content ?? "",
        createdAt: n?.created_at ?? n?.createdAt ?? "",
      })));
    }).catch(() => setNews([]));
    GetQMServerAPIBase().then(setApiBase).catch(() => setApiBase(""));
    GetLauncherAPITarget()
      .then((s) => {
        setLauncherApiUseCloud(s.use_qmserver_cloud !== false);
        setLauncherApiCustom(s.custom_api_base || "");
        setLauncherApiEffective(s.effective_api_base || "");
      })
      .catch(() => {});
    GetLauncherDebug()
      .then((on) => setLauncherDebug(!!on))
      .catch(() => setLauncherDebug(false));
    GetCurseForgeKeySettings()
      .then((s) => {
        setCurseForgeKeySavedInFile(!!s?.key_saved_in_file);
        setCurseForgeUseMyKeyDefault(!!s?.use_my_key_default);
        setCurseForgeEffectiveSource(typeof s?.effective_source === "string" ? s.effective_source : "none");
      })
      .catch(() => {
        setCurseForgeKeySavedInFile(false);
        setCurseForgeUseMyKeyDefault(false);
        setCurseForgeEffectiveSource("none");
      });
    GetCatalogStoreSettings()
      .then((s) => {
        setCatalogCurseforgeEnabled(s?.curseforge_enabled !== false);
        setCatalogModrinthEnabled(s?.modrinth_enabled !== false);
      })
      .catch(() => {
        setCatalogCurseforgeEnabled(true);
        setCatalogModrinthEnabled(true);
      });
    GetLang().then(setCurrentLang).catch(() => setCurrentLang("ru"));
    GetLauncherVersion()
      .then((v) => setLauncherVersion(typeof v === "string" ? v : ""))
      .catch(() => setLauncherVersion(""));
    GetMicrosoftAuthAvailable().then(setMicrosoftAuthAvailable).catch(() => setMicrosoftAuthAvailable(false));
    GetAccounts().then((accountsList) => {
      setAccounts(accountsList);
      // Check if no accounts exist
      if (accountsList.length === 0) {
        setShowNoAccountDialog(true);
      }
    }).catch((error) => {
      console.error('Failed to fetch accounts:', error);
      setAccounts([]);
      // Show dialog if failed to fetch (assume no accounts)
      setShowNoAccountDialog(true);
    });

    // Load current account
    GetCurrentAccount().then((account: any) => {
      applySidebarForCloudAccount(account, { requireDefault: true });
    }).catch((error: any) => {
      console.error('Failed to fetch current account:', error);
      setCurrentAccount({ name: "User", email: "user@qmlauncher.local" });
      setCurrentAccountType("");
      setCloudProfileAvatar(undefined);
      setCloudIsPremium(false);
    });
  }, []);

  /** Периодически подтягиваем публичные данные QMServer (серверы, новости, флаги) без перезапуска лаунчера */
  useEffect(() => {
    const POLL_MS = 90_000;
    const refreshCloudSnapshot = () => {
      void (async () => {
        try {
          await InvalidateQMServersCache();
        } catch {
          /* ignore */
        }
        try {
          const list = await GetRecentServers();
          setServers(list);
          if (list.length === 0) {
            const err = await GetQMServersError();
            setServersLoadError(normalizeQMServerErrorMessage(err || ""));
          } else {
            setServersLoadError("");
          }
        } catch (error) {
          console.error("QMServer poll: servers", error);
          setServers([]);
          setServersLoadError(normalizeQMServerErrorMessage(String(error)));
        }
        try {
          const items = await GetNews();
          const list = Array.isArray(items) ? items : [];
          setNews(
            list.map((n: any, i: number) => ({
              id: n?.id ?? i,
              title: n?.title ?? "",
              content: n?.content ?? "",
              createdAt: n?.created_at ?? n?.createdAt ?? "",
            }))
          );
        } catch {
          /* оставляем предыдущие новости */
        }
        try {
          const skin = await GetSkinProviderConfig();
          if (skin && typeof skin === "object") setSkinProviders(skin as Record<string, boolean>);
        } catch {
          /* keep */
        }
        try {
          const s = await GetCatalogStoreSettings();
          setCatalogCurseforgeEnabled(s?.curseforge_enabled !== false);
          setCatalogModrinthEnabled(s?.modrinth_enabled !== false);
        } catch {
          /* keep */
        }
        try {
          const cf = await GetCurseForgeKeySettings();
          setCurseForgeKeySavedInFile(!!cf?.key_saved_in_file);
          setCurseForgeUseMyKeyDefault(!!cf?.use_my_key_default);
          setCurseForgeEffectiveSource(
            typeof cf?.effective_source === "string" ? cf.effective_source : "none"
          );
        } catch {
          /* keep */
        }
        try {
          setMicrosoftAuthAvailable(await GetMicrosoftAuthAvailable());
        } catch {
          /* keep */
        }
      })();
    };
    const id = window.setInterval(refreshCloudSnapshot, POLL_MS);
    return () => window.clearInterval(id);
  }, []);

  useEffect(() => {
    if (!showCreateInstanceDialog) return;
    let cancelled = false;
    setCreateInstanceMcLoading(true);
    GetCreateInstanceMinecraftVersions()
      .then((list) => {
        if (cancelled) return;
        const versions = Array.isArray(list) ? list : [];
        setCreateInstanceMcVersions(versions);
        setCreateInstanceVersion((prev) =>
          prev && versions.includes(prev) ? prev : versions[0] ?? ""
        );
      })
      .catch(() => {
        if (!cancelled) {
          setCreateInstanceMcVersions([]);
        }
      })
      .finally(() => {
        if (!cancelled) setCreateInstanceMcLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [showCreateInstanceDialog]);

  useEffect(() => {
    if (!showCreateInstanceDialog) return;
    if (createInstanceLoader === "vanilla") {
      setCreateInstanceLoaderVersions([]);
      setCreateInstanceLoaderVersion("");
      setCreateInstanceLoaderVersionsLoading(false);
      return;
    }
    const gv = createInstanceVersion.trim();
    if (!gv) {
      setCreateInstanceLoaderVersions([]);
      setCreateInstanceLoaderVersion("");
      setCreateInstanceLoaderVersionsLoading(false);
      return;
    }
    let cancelled = false;
    setCreateInstanceLoaderVersionsLoading(true);
    GetCreateInstanceLoaderVersions(createInstanceLoader, gv)
      .then((list) => {
        if (cancelled) return;
        const versions = Array.isArray(list) ? list : [];
        setCreateInstanceLoaderVersions(versions);
        setCreateInstanceLoaderVersion((prev) =>
          prev && versions.includes(prev) ? prev : versions[0] ?? ""
        );
      })
      .catch(() => {
        if (!cancelled) {
          setCreateInstanceLoaderVersions([]);
          setCreateInstanceLoaderVersion("");
        }
      })
      .finally(() => {
        if (!cancelled) setCreateInstanceLoaderVersionsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [showCreateInstanceDialog, createInstanceLoader, createInstanceVersion]);

  useEffect(() => {
    const unsubLaunchProgress = EventsOn("launch-progress", (ev: any) => {
      if (ev && typeof ev === "object") {
        const msg = ev.message ?? ev.Message ?? "";
        const progress = ev.progress ?? ev.Progress;
        const typ = ev.type ?? ev.Type;
        const phase = ev.phase ?? ev.Phase;
        const file = ev.currentFile ?? ev.current_file ?? "";
        setLaunchProgress((prev) => ({
          ...prev,
          message: msg || prev.message,
          progress: progress !== undefined ? progress : prev.progress,
          type: typ !== undefined ? typ : prev.type,
          phase: phase !== undefined ? phase : prev.phase,
          currentFile: file !== undefined ? file : prev.currentFile,
        }));
      }
    });
    const unsubCloudSuccess = EventsOn("cloud-auth-success", () => {
      GetAccounts().then(setAccounts);
      GetCurrentAccount().then((account: any) => {
        applySidebarForCloudAccount(account, { requireDefault: false, emailFallback: "" });
      });
    });
    const unsubCloudError = EventsOn("cloud-auth-error", (msg: string) => {
      showAlert("Ошибка входа", msg || "Не удалось войти в QMServer Cloud");
    });
    const unsubMicrosoftSuccess = EventsOn("microsoft-auth-success", () => {
      GetAccounts().then(setAccounts);
      GetCurrentAccount().then((account: any) => {
        applySidebarForCloudAccount(account, { requireDefault: false, emailFallback: "" });
      });
    });
    const unsubMicrosoftError = EventsOn("microsoft-auth-error", (msg: string) => {
      showAlert("Ошибка входа", msg || "Не удалось войти через Microsoft");
    });
    const unsubLauncherUpdate = EventsOn("launcher-update-available", () => {
      setShowUpdateDialog(true);
    });
    return () => {
      unsubLaunchProgress?.();
      unsubCloudSuccess?.();
      unsubCloudError?.();
      unsubMicrosoftSuccess?.();
      unsubMicrosoftError?.();
      unsubLauncherUpdate?.();
    };
  }, []);

  const handleLaunchInstance = async (instanceName: string) => {
    setSyncConfigFromServer(false);
    if (launchableAccounts.length === 0) {
      showAlert("Ошибка", "Добавьте игровой аккаунт для запуска");
      return;
    }
    if (launchableAccounts.length === 1) {
      await launchWithSelectedGameAccount(
        { instanceName, serverAddress: "", serverID: 0 },
        launchableAccounts[0].username,
        false
      );
      return;
    }
    const defaultUser =
      launchableAccounts.find((a) => a.isDefault)?.username ??
      launchableAccounts[0]?.username ??
      "";
    setPendingLaunch({
      instanceName,
      serverAddress: "",
      serverID: 0,
      syncConfigFromServer: false,
      selectedAccountName: defaultUser,
    });
    setShowModSelectionDialog(true);
  };

  const connectToServer = async (server: ServerInfo) => {
    console.log('Connecting to server:', server);

    if (!server.address || !server.port) {
      showAlert("Ошибка", "Не указан адрес или порт сервера");
      return;
    }

    if (server.enabled === false) {
      showAlert(t.serverDisabledShort, t.serverDisabledError);
      return;
    }
    if (server.gameServerOnline === false) {
      showAlert(t.gameProcessOffline, t.gameProcessOfflineDetail);
      return;
    }

    // Do NOT show launch progress here — it appears only after account selection and "Запустить"
    try {
      // Step 1: Ensure instance exists (create or get) - exact copy of TUI
      const instanceName = await EnsureInstanceForServer(
        server.name,
        `${server.address}:${server.port}`,
        server.version || "release",
        server.modLoader || "vanilla",
        server.modLoaderVersion || "latest",
        server.serverID ?? 0
      );

      if (instanceName.startsWith("Error:")) {
        setShowLaunchProgressDialog(true);
        setLaunchProgress({ message: instanceName, type: "error" });
        setTimeout(() => setShowLaunchProgressDialog(false), 3000);
        return;
      }

      const serverAddress = `${server.address}:${server.port}`;
      setSyncConfigFromServer(false);
      if (launchableAccounts.length === 0) {
        showAlert("Ошибка", "Добавьте игровой аккаунт для подключения к серверу");
        return;
      }
      if (launchableAccounts.length === 1) {
        await launchWithSelectedGameAccount(
          {
            instanceName,
            serverAddress,
            serverID: server.serverID || 0,
            serverName: server.name,
          },
          launchableAccounts[0].username,
          false
        );
      } else {
        const defaultUser =
          launchableAccounts.find((a) => a.isDefault)?.username ??
          launchableAccounts[0]?.username ??
          "";
        setPendingLaunch({
          instanceName,
          serverAddress,
          serverID: server.serverID || 0,
          serverName: server.name,
          syncConfigFromServer: false,
          selectedAccountName: defaultUser,
        });
        setShowModSelectionDialog(true);
      }

      // Refresh instances list after creation
      GetInstances()
      .then((raw) => setInstances(normalizeInstancesList(raw)))
      .catch(() => setInstances([]));
    } catch (error) {
      console.error('Connection error:', error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      setShowLaunchProgressDialog(true);
      setLaunchProgress({ message: `Ошибка: ${errorMessage}`, type: "error" });
      setTimeout(() => setShowLaunchProgressDialog(false), 3000);
    }
  };

  const formatNewsDate = (s: string) => {
    try {
      return new Date(s).toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit", year: "numeric" });
    } catch {
      return s;
    }
  };

  const handleLanguageChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const lang = e.target.value;
    await SetLang(lang);
    window.location.reload();
  };

  const renderSettings = () => (
    <div className="space-y-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold">{t.settingsMenu}</h2>
        <p className="text-sm text-muted-foreground">Настройки лаунчера</p>
      </div>
      <Tabs defaultValue="general" className="w-full">
        <TabsList className="grid w-full max-w-md grid-cols-2">
          <TabsTrigger value="general">{t.settingsTabGeneral}</TabsTrigger>
          <TabsTrigger value="minecraft">{t.settingsTabMinecraft}</TabsTrigger>
        </TabsList>
        <TabsContent value="general" className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.settingsLanguage}</CardTitle>
        </CardHeader>
        <CardContent>
          <NativeSelect value={currentLang} onChange={handleLanguageChange}>
            <NativeSelectOption value="ru">Русский</NativeSelectOption>
            <NativeSelectOption value="en">English</NativeSelectOption>
          </NativeSelect>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.settingsTheme}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">Светлая / Тёмная тема</span>
            <ModeToggle />
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.settingsDebug}</CardTitle>
          <CardDescription>{t.settingsDebugDesc}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">Debug</span>
            <Switch
              checked={launcherDebug}
              disabled={launcherDebugSaving}
              onCheckedChange={(on) => {
                setLauncherDebugSaving(true);
                void (async () => {
                  try {
                    const err = await SetLauncherDebug(on);
                    if (err) {
                      showAlert(t.settingsDebug, err);
                      return;
                    }
                    setLauncherDebug(on);
                  } finally {
                    setLauncherDebugSaving(false);
                  }
                })();
              }}
            />
          </div>
        </CardContent>
      </Card>
        </TabsContent>
        <TabsContent value="minecraft" className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Адрес QMServer API</CardTitle>
          <CardDescription>
            Список серверов в лаунчере и запросы к API идут на этот базовый URL. Включите{' '}
            <strong>QMServer Cloud</strong> для адреса по умолчанию (или значения{' '}
            <code className="text-xs">QMSERVER_API_BASE</code> у процесса лаунчера). Выключите
            и укажите свой инстанс с суффиксом <code className="text-xs">/api/v1</code>.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between gap-4">
            <div className="min-w-0">
              <p className="text-sm font-medium">Использовать QMServer Cloud</p>
              <p className="text-xs text-muted-foreground">
                Публичное облако (см. документацию проекта) или переопределение через окружение.
              </p>
            </div>
            <Switch checked={launcherApiUseCloud} onCheckedChange={setLauncherApiUseCloud} />
          </div>
          {!launcherApiUseCloud ? (
            <div className="grid gap-2">
              <Label htmlFor="launcher-api-base">Свой базовый URL API</Label>
              <Input
                id="launcher-api-base"
                value={launcherApiCustom}
                onChange={(e) => setLauncherApiCustom(e.target.value)}
                placeholder="https://example.com/api/v1"
                className="font-mono text-sm"
              />
            </div>
          ) : null}
          <p className="text-xs text-muted-foreground break-all">
            Активный адрес:{' '}
            <span className="font-mono text-foreground">{launcherApiEffective || "—"}</span>
          </p>
          <Button
            type="button"
            disabled={launcherApiSaving}
            onClick={() => {
              setLauncherApiSaving(true);
              void (async () => {
                try {
                  const err = await SetLauncherAPITarget(launcherApiUseCloud, launcherApiCustom.trim());
                  if (err) {
                    showAlert("Адрес API", err);
                    return;
                  }
                  const s = await GetLauncherAPITarget();
                  setLauncherApiEffective(s.effective_api_base || "");
                  GetQMServerAPIBase().then(setApiBase).catch(() => setApiBase(""));
                  GetMicrosoftAuthAvailable().then(setMicrosoftAuthAvailable).catch(() => {});
                  try {
                    const list = await GetRecentServers();
                    setServers(list);
                    setServersLoadError("");
                  } catch (e) {
                    setServers([]);
                    setServersLoadError(normalizeQMServerErrorMessage(String(e)));
                  }
                  try {
                    const items = await GetNews();
                    const list = Array.isArray(items) ? items : [];
                    setNews(
                      list.map((n: any, i: number) => ({
                        id: n?.id ?? i,
                        title: n?.title ?? "",
                        content: n?.content ?? "",
                        createdAt: n?.created_at ?? n?.createdAt ?? "",
                      }))
                    );
                  } catch {
                    setNews([]);
                  }
                  showAlert("Адрес API", "Сохранено. Данные обновлены.");
                } finally {
                  setLauncherApiSaving(false);
                }
              })();
            }}
          >
            {launcherApiSaving ? "Сохранение…" : "Сохранить"}
          </Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="space-y-3">
          <div className="flex items-center justify-between gap-4">
            <CardTitle className="text-base">CurseForge</CardTitle>
            <Switch
              checked={catalogCurseforgeEnabled}
              onCheckedChange={(on) => void persistCatalogStoreSettings(on, catalogModrinthEnabled)}
            />
          </div>
          <CardDescription>
            Ключ API можно сгенерировать в{' '}
            <a
              href="https://console.curseforge.com/"
              target="_blank"
              rel="noreferrer"
              className="text-primary underline"
            >
              CurseForge Developer Console
            </a>
            .
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="curseforge-api-key">API-ключ</Label>
            <Input
              id="curseforge-api-key"
              type="password"
              autoComplete="off"
              value={curseForgeKeyInput}
              onChange={(e) => setCurseForgeKeyInput(e.target.value)}
              placeholder={curseForgeKeySavedInFile ? "Ключ сохранён — введите новый, чтобы заменить" : "Вставьте ключ из консоли разработчика CurseForge"}
              className="font-mono text-sm"
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              disabled={curseForgeKeySaving}
              onClick={() => {
                setCurseForgeKeySaving(true);
                void (async () => {
                  try {
                    const err = await SetCurseForgeSettingsKey(
                      curseForgeKeyInput.trim(),
                      curseForgeUseMyKeyDefault,
                      false
                    );
                    if (err) {
                      showAlert("CurseForge API", err);
                      return;
                    }
                    setCurseForgeKeyInput("");
                    const s = await GetCurseForgeKeySettings();
                    setCurseForgeKeySavedInFile(!!s?.key_saved_in_file);
                    setCurseForgeUseMyKeyDefault(!!s?.use_my_key_default);
                    setCurseForgeEffectiveSource(typeof s?.effective_source === "string" ? s.effective_source : "none");
                    showAlert("CurseForge API", "Сохранено.");
                  } finally {
                    setCurseForgeKeySaving(false);
                  }
                })();
              }}
            >
              {curseForgeKeySaving ? "Сохранение…" : "Сохранить"}
            </Button>
            {curseForgeKeySavedInFile ? (
              <Button
                type="button"
                variant="outline"
                disabled={curseForgeKeySaving}
                onClick={() => {
                  setCurseForgeKeySaving(true);
                  void (async () => {
                    try {
                      const err = await SetCurseForgeSettingsKey("", false, true);
                      if (err) {
                        showAlert("CurseForge API", err);
                        return;
                      }
                      setCurseForgeKeyInput("");
                      setCurseForgeKeySavedInFile(false);
                      setCurseForgeUseMyKeyDefault(false);
                      const s = await GetCurseForgeKeySettings();
                      setCurseForgeEffectiveSource(typeof s?.effective_source === "string" ? s.effective_source : "none");
                      showAlert("CurseForge API", "Ключ из настроек удалён.");
                    } finally {
                      setCurseForgeKeySaving(false);
                    }
                  })();
                }}
              >
                Удалить ключ из настроек
              </Button>
            ) : null}
          </div>
          <div className="rounded-md border bg-muted/30 p-3 text-sm space-y-1">
            <p className="font-medium text-muted-foreground">Источник ключа для API</p>
            <p>{curseForgeEffectiveSourceLabel(curseForgeEffectiveSource)}</p>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="space-y-2">
          <div className="flex items-center justify-between gap-4">
            <CardTitle className="text-base">Modrinth</CardTitle>
            <Switch
              checked={catalogModrinthEnabled}
              onCheckedChange={(on) => void persistCatalogStoreSettings(catalogCurseforgeEnabled, on)}
            />
          </div>
          <CardDescription>Поиск и установка из каталога Modrinth (api.modrinth.com).</CardDescription>
        </CardHeader>
      </Card>
        </TabsContent>
      </Tabs>
    </div>
  );

  const renderNews = () => (
    <div className="space-y-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold">{t.news}</h2>
        <p className="text-sm text-muted-foreground">Актуальные новости с QMServer Cloud</p>
      </div>
      {news.length === 0 ? (
        <Card>
          <CardContent className="py-8">
            <p className="text-muted-foreground text-sm text-center">Новостей пока нет.</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4 max-h-[60vh] overflow-y-auto">
          {news.map((n) => (
            <Card key={n.id}>
              <CardHeader className="pb-2">
                <CardTitle className="text-base">{n.title}</CardTitle>
                <span className="text-xs text-muted-foreground">{formatNewsDate(n.createdAt)}</span>
              </CardHeader>
              <CardContent>
                <p className="text-muted-foreground text-sm whitespace-pre-wrap">{n.content}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );

  const renderServers = () => (
    <div className="space-y-6">
      <div className="space-y-4">
        <div className="flex justify-between items-center mb-6">
          <div>
            <p className="text-muted-foreground mt-1">{t.serversSelect}</p>
          </div>
        </div>

        <div className="bg-card border border-border rounded-lg overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t.serversTableName}
              </TableHead>
              <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t.serversTableVersion}
              </TableHead>
              <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t.serversTablePlayers}
              </TableHead>
              <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Рейтинг
              </TableHead>
              <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {t.serversConnect}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {servers.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-8 text-muted-foreground">
                  {serversLoadError ? (
                    <span className="text-destructive">{serversLoadError}</span>
                  ) : (
                    t.serversNotFound
                  )}
                </TableCell>
              </TableRow>
            ) : (
              servers.map((server) => (
                <TableRow
                  key={server.id}
                  className={
                    server.enabled === false || server.gameServerOnline === false
                      ? "opacity-80"
                      : undefined
                  }
                >
                  <TableCell className="px-4 py-4 whitespace-nowrap">
                    <div className="inline-flex items-center gap-2">
                      <button
                        type="button"
                        className="text-sm font-medium text-foreground hover:text-primary underline-offset-2 hover:underline"
                        onClick={() => setSelectedServerDetails(server)}
                      >
                        {server.name}
                      </button>
                      <McServerPingBadge
                        server={server}
                        onlineLabel={t.badgeOnline}
                        offlineLabel={t.badgeOffline}
                      />
                      {server.enabled === false ? (
                        <span className="text-xs px-1.5 py-0.5 rounded bg-destructive/15 text-destructive border border-destructive/30 select-none">
                          {t.serverDisabledShort}
                        </span>
                      ) : null}
                      {server.isPremium && (
                        <span className="text-xs px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-500 border border-yellow-500/30 select-none">
                          {t.premium}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm text-muted-foreground">
                    {server.version || "-"}
                  </TableCell>
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm text-muted-foreground">
                    {server.enabled === false || server.gameServerOnline === false
                      ? "—"
                      : `${server.players}/${server.maxPlayers}`}
                  </TableCell>
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm text-muted-foreground">
                    {/* Статический рейтинг-заглушка: 5/5, можно позже сделать динамическим */}
                    <span aria-label="Рейтинг сервера">5/5</span>
                  </TableCell>
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm">
                    <Button
                      size="sm"
                      onClick={() => connectToServer(server)}
                      disabled={server.enabled === false || !server.online}
                      variant="default"
                    >
                      <Play className="w-4 h-4 mr-2" />
                      {t.serversConnect}
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      </div>
    </div>
  );

  const openCreateInstanceDialog = () => {
    setCreateInstanceName("");
    setCreateInstanceLoader("vanilla");
    setCreateInstanceVersion("");
    setCreateInstanceLoaderVersion("");
    setCreateInstanceMcVersions([]);
    setCreateInstanceLoaderVersions([]);
    setShowCreateInstanceDialog(true);
  };

  const renderInstances = () => {
    const list = normalizeInstancesList(instances);
    return (
    <div className="min-w-0 space-y-4">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-foreground">{t.instances}</h2>
      </div>

      {list.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center gap-4 py-16 px-6 text-center">
            <Box className="h-14 w-14 text-muted-foreground opacity-70" aria-hidden />
            <div className="space-y-2 max-w-lg">
              <p className="text-base font-medium text-foreground">{t.instancesEmptyTitle}</p>
              <p className="text-sm text-muted-foreground leading-relaxed">{t.instancesEmptyHint}</p>
            </div>
            <Button size="lg" onClick={openCreateInstanceDialog}>
              <Plus className="w-4 h-4 mr-2" />
              {t.createInstance}
            </Button>
          </CardContent>
        </Card>
      ) : (
        <>
      <div className="flex min-w-0 justify-end items-center mb-2 gap-2">
        <Button variant="secondary" onClick={openCreateInstanceDialog}>
          <Plus className="w-4 h-4 mr-2" />
          {t.createInstance}
        </Button>
      </div>

      <div className="grid min-w-0 gap-6 md:grid-cols-2 lg:grid-cols-3">
        {list.map((instance: any, idx: number) => {
          if (instance == null || typeof instance !== "object") return null;
          // Поддерживаем оба варианта кейса полей (Go JSON / TypeScript моделей)
          const name =
            String(instance?.name ?? instance?.Name ?? "").trim() || `instance-${idx}`;
          const gameVersion = instance.GameVersion || instance.game_version || "-";
          const loader = instance.Loader || instance.mod_loader || "vanilla";
          const loaderVersion = instance.LoaderVersion || instance.mod_loader_version || "";
          const config = instance.Config || instance.config || {};
          const lastServer = config.LastServer || config.last_server || "";
          const isUsingQMServerCloud =
            config.IsUsingQMServerCloud ?? config.is_using_qmserver_cloud ?? false;
          const isPremium = config.IsPremium ?? config.is_premium ?? false;

          const loaderLabel = formatLoaderLabel(loader, loaderVersion);

          return (
            <Card
              key={`${name}-${idx}`}
              className="group flex cursor-pointer flex-col gap-0 overflow-hidden border border-border bg-card p-0 shadow-sm transition-[background-color,box-shadow,border-color] duration-300 ease-out hover:border-border hover:bg-accent hover:shadow-md"
              onClick={() => {
                setSelectedInstanceName(name);
                window.location.hash = '#instance';
              }}
            >
              <div className="aspect-16/10 w-full max-w-full shrink-0 overflow-hidden bg-muted">
                <img
                  src={INSTANCE_CARD_COVER_PLACEHOLDER}
                  alt=""
                  className="block h-full w-full max-w-full object-cover"
                  loading="lazy"
                  decoding="async"
                />
              </div>
              <CardHeader className="border-t border-border/60 transition-[border-color] duration-300 ease-out group-hover:border-border">
                <CardTitle className="flex items-center justify-between text-foreground transition-colors duration-300 ease-out group-hover:text-primary">
                  <span>{name}</span>
                  {isPremium && (
                    <span className="ml-2 text-xs px-1.5 py-0.5 rounded bg-yellow-500/20 text-yellow-500 border border-yellow-500/30">
                      {t.premium}
                    </span>
                  )}
                </CardTitle>
                <CardDescription className="text-muted-foreground transition-colors duration-300 ease-out">
                  Minecraft {gameVersion}
                </CardDescription>
              </CardHeader>
              <CardContent className="transition-colors duration-300 ease-out">
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between gap-4">
                    <span className="text-muted-foreground">Загрузчик:</span>
                    <span className="font-medium text-foreground">{loaderLabel}</span>
                  </div>
                  {lastServer && (
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground">Сервер:</span>
                      <span className="font-medium text-foreground text-right break-all">
                        {lastServer}
                        {isUsingQMServerCloud && ' (QMServer Cloud)'}
                      </span>
                    </div>
                  )}
                </div>
                <Button
                  className="mt-4 w-full transition-colors duration-200"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleLaunchInstance(name);
                  }}
                >
                  <Play className="mr-2 h-4 w-4" />
                  {t.launchInstance}
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>
        </>
      )}
    </div>
  );
  };

  useEffect(() => {
    if (selectedInstanceName) {
      GetInstanceDetails(selectedInstanceName).then(setInstanceDetails).catch(() => setInstanceDetails(null));
    } else {
      setInstanceDetails(null);
    }
  }, [selectedInstanceName]);

  useEffect(() => {
    if (activeTab === "instance-resources") {
      setInstanceResourcesSubTab("mods");
    }
  }, [activeTab, selectedInstanceName]);

  const renderInstancePage = () => {
    const instance = instances.find(
      (i: any) => (i.name || i.Name) === selectedInstanceName
    );

    if (!instance) {
      return (
        <div className="text-center py-12">
          <p className="text-muted-foreground">Инстанс не найден</p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => window.location.hash = '#instances'}
          >
            Вернуться к списку инстансов
          </Button>
        </div>
      );
    }

    // Поддерживаем оба варианта кейса полей (Go JSON / TypeScript моделей)
    const name = instance.name || instance.Name || "Без имени";
    const gameVersion = instance.GameVersion || instance.game_version || "-";
    const loader = instance.Loader || instance.mod_loader || "vanilla";
    const loaderVersion = instance.LoaderVersion || instance.mod_loader_version || "";
    const config = instance.Config || instance.config || {};
    const lastServer = config.LastServer || config.last_server || "";
    const isUsingQMServerCloud =
      config.IsUsingQMServerCloud ?? config.is_using_qmserver_cloud ?? false;
    const isPremium = config.IsPremium ?? config.is_premium ?? false;

    const loaderLabel = formatLoaderLabel(loader, loaderVersion);

    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => window.location.hash = '#instances'}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Назад
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">{name}</h1>
            <p className="text-muted-foreground">Minecraft {gameVersion}</p>
          </div>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <div className="md:col-span-2 space-y-6">
            <Card className="bg-card border border-border">
              <CardHeader>
                <CardTitle className="text-foreground">Информация об инстансе</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="flex justify-between gap-4">
                    <span className="text-muted-foreground">Версия Minecraft:</span>
                    <span className="font-medium text-foreground">{gameVersion}</span>
                  </div>
                  <div className="flex justify-between gap-4">
                    <span className="text-muted-foreground">Загрузчик:</span>
                    <span className="font-medium text-foreground">{loaderLabel}</span>
                  </div>
                  {isPremium && (
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground">Тип:</span>
                      <span className="font-medium text-yellow-500">Premium</span>
                    </div>
                  )}
                  {lastServer && (
                    <div className="flex justify-between gap-4">
                      <span className="text-muted-foreground">Последний сервер:</span>
                      <span className="font-medium text-foreground text-right break-all">
                        {lastServer}
                        {isUsingQMServerCloud && " (QMServer Cloud)"}
                      </span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

          </div>

          <div className="space-y-6">
            <Card className="bg-card border border-border">
              <CardHeader>
                <CardTitle className="text-foreground">Действия</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <Button
                  className="w-full"
                  onClick={() => handleLaunchInstance(name)}
                >
                  <Play className="w-4 h-4 mr-2" />
                  {t.launchInstance}
                </Button>
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => {
                    window.location.hash = `#instance-settings/${encodeURIComponent(name)}`;
                  }}
                >
                  <Settings className="w-4 h-4 mr-2" />
                  Настройки
                </Button>
                {isModdedInstanceLoader(loader) && (
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => {
                      window.location.hash = `#instance-resources/${encodeURIComponent(name)}`;
                    }}
                  >
                    <Package className="w-4 h-4 mr-2" />
                    Ресурсы
                  </Button>
                )}
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => {
                    const dir = instanceDetails?.dir || instanceDetails?.Dir;
                    if (dir) {
                      OpenPath(dir).then((err) => err && showAlert("Ошибка", err));
                    } else {
                      showAlert("Ошибка", "Путь к папке инстанса не найден");
                    }
                  }}
                >
                  <FolderOpen className="w-4 h-4 mr-2" />
                  Открыть папку
                </Button>
                <Button
                  variant="destructive"
                  className="w-full"
                  onClick={() => {
                    setInstanceNameToDelete(name);
                    setShowDeleteInstanceDialog(true);
                  }}
                >
                  <Trash2 className="w-4 h-4 mr-2" />
                  Удалить инстанс
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    );
  };

  const renderInstanceSettingsPage = () => {
    const instance = instances.find(
      (i: any) => (i.name || i.Name) === selectedInstanceName
    );

    if (!instance) {
      return (
        <div className="text-center py-12">
          <p className="text-muted-foreground">Инстанс не найден</p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => (window.location.hash = "#instances")}
          >
            Вернуться к списку инстансов
          </Button>
        </div>
      );
    }

    const name = instance.name || instance.Name || "Без имени";
    const gameVersion = instance.GameVersion || instance.game_version || "-";
    const config = instance.Config || instance.config || {};
    const minMem = config.MinMemory ?? config.min_memory ?? 4096;
    const maxMem = config.MaxMemory ?? config.max_memory ?? 4096;
    const minMB = Math.max(128, Math.min(32768, minMem));
    const maxMB = Math.max(512, Math.min(32768, Math.max(minMB, maxMem)));

    return (
      <div className="space-y-6 max-w-2xl">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              window.location.hash = "#instance";
            }}
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            Назад к инстансу
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Настройки инстанса</h1>
            <p className="text-muted-foreground">
              {name} · Minecraft {gameVersion}
            </p>
          </div>
        </div>

        <Card className="bg-card border border-border">
          <CardHeader>
            <CardTitle className="text-foreground">Память и запуск</CardTitle>
            <CardDescription>-Xms / -Xmx (МБ)</CardDescription>
          </CardHeader>
          <CardContent>
            <InstanceMemoryControls
              instanceName={name}
              minMemoryMB={minMB}
              maxMemoryMB={maxMB}
              showAlert={showAlert}
              onAfterSave={() =>
                GetInstances()
                  .then((raw) => setInstances(normalizeInstancesList(raw)))
                  .catch(() => setInstances([]))
              }
            />
          </CardContent>
        </Card>
      </div>
    );
  };

  const renderInstanceResourcesPage = () => {
    const instance = instances.find(
      (i: any) => (i.name || i.Name) === selectedInstanceName
    );

    if (!instance) {
      return (
        <div className="text-center py-12">
          <p className="text-muted-foreground">Инстанс не найден</p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => (window.location.hash = "#instances")}
          >
            Вернуться к списку инстансов
          </Button>
        </div>
      );
    }

    const name = instance.name || instance.Name || "Без имени";
    const gameVersion = instance.GameVersion || instance.game_version || "-";
    const loader = instance.Loader || instance.mod_loader || "vanilla";

    if (!isModdedInstanceLoader(loader)) {
      return (
        <div className="space-y-6 max-w-3xl">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="sm" onClick={() => (window.location.hash = "#instance")}>
              <ArrowLeft className="w-4 h-4 mr-2" />
              Назад
            </Button>
            <h1 className="text-2xl font-bold text-foreground">Ресурсы</h1>
          </div>
          <Card>
            <CardContent className="py-8">
              <p className="text-muted-foreground text-sm">
                Раздел «Ресурсы» доступен только для инстансов с загрузчиком модов (Fabric, Forge и т.д.), не для
                Vanilla.
              </p>
            </CardContent>
          </Card>
        </div>
      );
    }

    const modFiles = (instanceDetails?.mods ?? [])
      .filter((f: string) => f && isInstanceModJarFile(f))
      .sort((a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: "base" }));

    const refreshInstanceResourceDetails = () => {
      GetInstanceDetails(name)
        .then(setInstanceDetails)
        .catch(() => setInstanceDetails(null));
    };

    const resourceAlertTitle: Record<InstanceResourcesSubTab, string> = {
      modpacks: "Модпаки",
      mods: "Моды",
      datapacks: "Датапаки",
      resourcepacks: "Ресурспаки",
      shaderpacks: "Шейдерпаки",
    };

    const runInstanceResourceAction = async (
      category: InstanceResourcesSubTab,
      resourcePath: string,
      action: () => Promise<string>
    ) => {
      const key = instanceResourceBusyKey(category, resourcePath);
      setInstanceResourceBusy(key);
      try {
        const err = await action();
        if (err) showAlert(resourceAlertTitle[category], err);
        refreshInstanceResourceDetails();
      } finally {
        setInstanceResourceBusy(null);
      }
    };

    const renderResourceTable = (
      category: InstanceResourcesSubTab,
      paths: string[],
      parseRow: (path: string) => { title: string; version: string; enabled: boolean },
      deleteConfirmMessage: (path: string) => string,
      options?: { showStoreLinks?: boolean; remoteInstalls?: Record<string, launcher.RemoteInstallMeta> }
    ) => {
      const showStoreLinks = options?.showStoreLinks !== false;
      const remoteInstalls = options?.remoteInstalls;
      const storeCfEnabled = catalogCfForInstance;
      const storeMrEnabled = catalogMrForInstance;
      const anyStoreLink = storeCfEnabled || storeMrEnabled;

      const openStorePage = async (path: string, field: "curseforgeUrl" | "modrinthUrl") => {
        const rowBusy = instanceResourceBusyKey(category, path);
        setInstanceResourceBusy(rowBusy);
        try {
          const links = await ResolveInstanceResourceStoreLinks(name, category, path);
          const url = field === "curseforgeUrl" ? links.curseforgeUrl : links.modrinthUrl;
          if (!url?.trim()) {
            showAlert(
              resourceAlertTitle[category],
              field === "curseforgeUrl"
                ? "Не удалось подобрать ссылку на CurseForge."
                : "Не удалось подобрать ссылку на Modrinth."
            );
            return;
          }
          const err = await OpenBrowserURL(url.trim());
          if (err) showAlert(resourceAlertTitle[category], err);
        } finally {
          setInstanceResourceBusy(null);
        }
      };
      if (!paths.length) {
        return null;
      }
      return (
        <div className="min-w-0 overflow-x-auto rounded-md border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[200px]">Название</TableHead>
                <TableHead className="w-[120px] whitespace-nowrap">Версия</TableHead>
                <TableHead className="w-[150px] align-middle">Ссылки</TableHead>
                <TableHead className="text-right w-[140px] whitespace-nowrap">Действия</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paths.map((path: string) => {
                const { title, version, enabled } = parseRow(path);
                const busy = instanceResourceBusy === instanceResourceBusyKey(category, path);
                const RowIcon = getInstanceResourceRowIcon(category, path);
                const riMeta = remoteInstalls?.[remoteInstallRowKey(category, path)];
                const catalogIconUrl = riMeta?.iconUrl?.trim();
                return (
                  <TableRow key={path}>
                    <TableCell className="align-top">
                      <div className="flex gap-3 min-w-0 items-start">
                        <div
                          className={`shrink-0 w-9 h-9 rounded-md border border-border flex items-center justify-center ${
                            enabled ? "bg-muted/90" : "bg-muted/50 opacity-80"
                          }`}
                          aria-hidden
                        >
                          <InstanceResourceThumb iconUrl={catalogIconUrl} RowIcon={RowIcon} />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div
                            className={`font-medium flex items-center gap-1.5 flex-wrap ${enabled ? "text-foreground" : "text-muted-foreground"}`}
                          >
                            <span>{title}</span>
                            {(() => {
                              if (!remoteInstalls) return null;
                              const ri = remoteInstalls[remoteInstallRowKey(category, path)];
                              if (!ri) return null;
                              const src =
                                ri.source === "curseforge"
                                  ? "CurseForge"
                                  : ri.source === "modrinth"
                                    ? "Modrinth"
                                    : ri.source;
                              const tip = ri.title?.trim()
                                ? `${ri.title} · ${src}${ri.slug ? ` · ${ri.slug}` : ""}`
                                : `Установлено из каталога: ${src}${ri.slug ? ` (${ri.slug})` : ""}`;
                              return (
                                <span
                                  title={tip}
                                  className="inline-flex items-center rounded-full border border-emerald-600/40 dark:border-emerald-400/50 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-400"
                                  aria-label={tip}
                                >
                                  {src}
                                </span>
                              );
                            })()}
                          </div>
                          <div
                            className="text-xs text-muted-foreground font-mono truncate max-w-[min(420px,85vw)] mt-0.5"
                            title={path}
                          >
                            {path}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="align-top text-muted-foreground whitespace-nowrap">{version}</TableCell>
                    <TableCell className="align-middle">
                      {showStoreLinks ? (
                        !anyStoreLink ? (
                          <span className="text-xs text-muted-foreground">—</span>
                        ) : storeCfEnabled && storeMrEnabled ? (
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                type="button"
                                size="sm"
                                variant="outline"
                                className="h-7 text-xs px-2 gap-1 justify-center min-w-[8.5rem]"
                                disabled={busy}
                              >
                                Каталог
                                <ChevronDown className="h-3.5 w-3.5 opacity-70" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="start" className="min-w-[10rem]">
                              <DropdownMenuItem onSelect={() => void openStorePage(path, "curseforgeUrl")}>
                                CurseForge
                              </DropdownMenuItem>
                              <DropdownMenuItem onSelect={() => void openStorePage(path, "modrinthUrl")}>
                                Modrinth
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        ) : storeCfEnabled ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            className="h-7 text-xs px-2"
                            disabled={busy}
                            onClick={() => void openStorePage(path, "curseforgeUrl")}
                          >
                            CurseForge
                          </Button>
                        ) : (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            className="h-7 text-xs px-2"
                            disabled={busy}
                            onClick={() => void openStorePage(path, "modrinthUrl")}
                          >
                            Modrinth
                          </Button>
                        )
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="align-middle text-right">
                      <div className="inline-flex items-center justify-end gap-3">
                        <div className="flex items-center gap-2">
                          <span className="text-xs text-muted-foreground whitespace-nowrap max-sm:hidden">
                            {enabled ? "Вкл." : "Выкл."}
                          </span>
                          <Switch
                            checked={enabled}
                            disabled={busy}
                            onCheckedChange={(checked) => {
                              if (checked === enabled) return;
                              runInstanceResourceAction(category, path, () =>
                                SetInstanceResourceEnabled(name, category, path, checked)
                              );
                            }}
                          />
                        </div>
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          className="h-8 w-8 shrink-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                          disabled={busy}
                          title="Удалить"
                          aria-label="Удалить"
                          onClick={() => {
                            if (!window.confirm(deleteConfirmMessage(path))) return;
                            runInstanceResourceAction(category, path, () =>
                              DeleteInstanceResource(name, category, path)
                            );
                          }}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      );
    };

    const modpackPaths = [...(instanceDetails?.modpacks ?? [])]
      .filter(Boolean)
      .sort((a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: "base" }));
    const datapackPaths = [...(instanceDetails?.datapacks ?? [])]
      .filter(Boolean)
      .sort((a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: "base" }));
    const resourcepackPaths = [...(instanceDetails?.resourcepacks ?? [])]
      .filter(Boolean)
      .sort((a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: "base" }));
    const shaderpackPaths = [...(instanceDetails?.shaderpacks ?? [])]
      .filter(Boolean)
      .sort((a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: "base" }));

    const remoteInstallMap: Record<string, launcher.RemoteInstallMeta> =
      (instanceDetails?.remoteInstalls as Record<string, launcher.RemoteInstallMeta> | undefined) ?? {};
    const catalogCfForInstance = instanceDetails?.catalogCurseforgeEnabled !== false;
    const catalogMrForInstance = instanceDetails?.catalogModrinthEnabled !== false;

    const subTabs: { id: InstanceResourcesSubTab; label: string }[] = [
      { id: "modpacks", label: "Модпаки" },
      { id: "mods", label: "Моды" },
      { id: "datapacks", label: "Датапаки" },
      { id: "resourcepacks", label: "Ресурспаки" },
      { id: "shaderpacks", label: "Шейдерпаки" },
    ];

    return (
      <div className="space-y-6 max-w-4xl">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => (window.location.hash = "#instance")}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            Назад к инстансу
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Ресурсы</h1>
            <p className="text-muted-foreground">
              {name} · Minecraft {gameVersion}
            </p>
          </div>
        </div>

        <div className="flex flex-wrap gap-2 border-b border-border pb-3">
          {subTabs.map((t) => (
            <Button
              key={t.id}
              type="button"
              variant={instanceResourcesSubTab === t.id ? "secondary" : "ghost"}
              size="sm"
              className="rounded-md"
              onClick={() => setInstanceResourcesSubTab(t.id)}
            >
              {t.label}
            </Button>
          ))}
        </div>

        <Card className="bg-card border border-border">
          <CardHeader>
            <CardTitle className="text-foreground">
              {subTabs.find((x) => x.id === instanceResourcesSubTab)?.label}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!instanceDetails ? (
              <p className="text-muted-foreground text-sm">Загрузка...</p>
            ) : (
              <>
                <div className="mb-4 flex flex-wrap items-center gap-2">
                  <Button
                    type="button"
                    size="sm"
                    onClick={() => {
                      const hash = `resource-store/${encodeURIComponent(name)}/${instanceResourcesSubTab}`;
                      window.location.hash = `#${hash}`;
                    }}
                  >
                    <Plus className="w-4 h-4 mr-2" />
                    Добавить из каталога
                  </Button>
                </div>
                {instanceResourcesSubTab === "modpacks"
                  ? renderResourceTable(
                      "modpacks",
                      modpackPaths,
                      parseInstanceModpackMarkerRow,
                      (p) => `Удалить «${p}» из корня инстанса?`,
                      { showStoreLinks: false, remoteInstalls: remoteInstallMap }
                    )
                  : instanceResourcesSubTab === "mods"
                    ? renderResourceTable(
                        "mods",
                        modFiles,
                        parseInstanceModRow,
                        (p) => `Удалить файл «${p}» из папки mods?`,
                        { remoteInstalls: remoteInstallMap }
                      )
                    : instanceResourcesSubTab === "datapacks"
                      ? renderResourceTable(
                          "datapacks",
                          datapackPaths,
                          parseInstanceDatapackRow,
                          (p) => `Удалить датапак «${p}»?`,
                          { remoteInstalls: remoteInstallMap }
                        )
                      : instanceResourcesSubTab === "resourcepacks"
                        ? renderResourceTable(
                            "resourcepacks",
                            resourcepackPaths,
                            parseInstanceZipPackRow,
                            (p) => `Удалить «${p}» из resourcepacks?`,
                            { remoteInstalls: remoteInstallMap }
                          )
                        : renderResourceTable(
                            "shaderpacks",
                            shaderpackPaths,
                            parseInstanceZipPackRow,
                            (p) => `Удалить «${p}» из shaderpacks?`,
                            { remoteInstalls: remoteInstallMap }
                          )}
              </>
            )}
          </CardContent>
        </Card>
      </div>
    );
  };

  const handleLoginQMServerCloud = () => {
    OpenBrowserForQMServerCloud();
    setShowNoAccountDialog(false);
  };

  const handleLoginMicrosoft = () => {
    OpenBrowserForMicrosoft();
    setShowNoAccountDialog(false);
  };

  const handleLogout = async () => {
    setShowLogoutConfirmDialog(true);
  };

  const confirmLogout = async () => {
    setShowLogoutConfirmDialog(false);
    try {
      const result = await LogoutAccount();
      showAlert("Выход", result);
        // Refresh accounts list
        GetAccounts().then((accountsList) => {
          setAccounts(accountsList);
          // Show dialog if no accounts left
          if (accountsList.length === 0) {
            setShowNoAccountDialog(true);
          }
        }).catch(console.error);
        // Refresh current account
        GetCurrentAccount().then((account: any) => {
          applySidebarForCloudAccount(account, { requireDefault: true });
        }).catch(console.error);
    } catch (error) {
      showAlert("Ошибка", String(error));
    }
  };

  const handleCreateLocalAccountClick = () => {
    setShowAccountTypeDialog(false);
    setNewAccountName("");
    setNewAccountSkinModel("steve");
    setShowCreateAccountDialog(true);
  };

  const handleCreateLocalAccount = async () => {
    if (!newAccountName || !newAccountName.trim()) {
      return;
    }
    try {
      await CreateLocalAccount(newAccountName.trim(), newAccountSkinModel);
      setShowCreateAccountDialog(false);
      setNewAccountName("");
      // Refresh accounts list
      GetAccounts().then((accountsList) => {
        setAccounts(accountsList);
        // Close dialog if account was created
        if (accountsList.length > 0) {
          setShowNoAccountDialog(false);
        }
      }).catch(console.error);
      // Refresh current account
      GetCurrentAccount().then((account: any) => {
        applySidebarForCloudAccount(account, { requireDefault: true });
      }).catch(console.error);
    } catch (error) {
      showAlert("Ошибка", String(error));
    }
  };

  const handleCreateAccountCancel = () => {
    setShowCreateAccountDialog(false);
    setNewAccountName("");
    setNewAccountSkinModel("steve");
  };

  const handleAddAccountClick = () => {
    setShowAccountTypeDialog(true);
  };

  const handleCreateCloudGameAccountClick = () => {
    setShowAccountTypeDialog(false);
    setCloudGameAccountName("");
    setCloudGameAccountSkinModel("steve");
    setShowCloudGameAccountDialog(true);
  };

  const handleCreateCloudGameAccount = async () => {
    if (!cloudGameAccountName || !cloudGameAccountName.trim()) {
      showAlert("Ошибка", "Введите имя аккаунта");
      return;
    }
    try {
      const result = await CreateCloudGameAccount(cloudGameAccountName.trim(), cloudGameAccountSkinModel);
      if (result.startsWith("Error:") || result.startsWith("API error")) {
        showAlert("Ошибка", result);
      } else {
        setShowCloudGameAccountDialog(false);
        setCloudGameAccountName("");
        // Refresh accounts list
        GetAccounts().then((accountsList) => {
          setAccounts(accountsList);
        }).catch(console.error);
        showAlert("Успешно", result);
      }
    } catch (error) {
      showAlert("Ошибка", String(error));
    }
  };

  const handleCloudGameAccountCancel = () => {
    setShowCloudGameAccountDialog(false);
    setCloudGameAccountName("");
    setCloudGameAccountSkinModel("steve");
  };

  const handleDeleteAccount = (account: AccountInfo) => {
    if (account.type === "microsoft") {
      handleLogout();
      return;
    }
    setAccountToDelete(account);
    setShowDeleteAccountDialog(true);
  };

  const confirmDeleteAccount = async () => {
    if (!accountToDelete) return;
    const { username, type, email, gameAccountId } = accountToDelete;
    setShowDeleteAccountDialog(false);
    setAccountToDelete(null);
    try {
      if (type === "cloud") {
        const result = await LogoutCloudAccount(email || "");
        showAlert("Аккаунт отключён", result);
      } else if (type === "cloud_game") {
        if (gameAccountId) {
          const result = await DeleteCloudGameAccount(gameAccountId);
          showAlert("Аккаунт удалён", result);
        } else {
          showAlert("Ошибка", "ID аккаунта не найден");
        }
      } else {
        const result = await DeleteLocalAccount(username);
        showAlert("Аккаунт удалён", result);
      }
      GetAccounts().then((accountsList) => {
        setAccounts(accountsList);
        if (accountsList.length === 0) {
          setShowNoAccountDialog(true);
        }
      }).catch(console.error);
      GetCurrentAccount().then((account: any) => {
        applySidebarForCloudAccount(account, { requireDefault: true });
      }).catch(console.error);
    } catch (error) {
      showAlert("Ошибка", String(error));
    }
  };

  const confirmDeleteInstance = async () => {
    const name = instanceNameToDelete;
    if (!name) return;
    setShowDeleteInstanceDialog(false);
    setInstanceNameToDelete(null);
    const err = await DeleteInstance(name);
    if (err) {
      showAlert("Ошибка", err);
      return;
    }
    GetInstances()
      .then((raw) => setInstances(normalizeInstancesList(raw)))
      .catch(() => setInstances([]));
    setInstanceDetails(null);
    setSelectedInstanceName("");
    window.location.hash = "#instances";
  };

  const handleSetDefaultAccount = async (username: string) => {
    try {
      await SetDefaultAccount(username);
      // Refresh accounts list
      GetAccounts().then((accountsList) => {
        setAccounts(accountsList);
      }).catch(console.error);
      // Refresh current account
      GetCurrentAccount().then((account: any) => {
        applySidebarForCloudAccount(account, { requireDefault: true });
      }).catch(console.error);
    } catch (error) {
      showAlert("Ошибка", String(error));
    }
  };

  const prepareGameAccountForLaunch = async (
    username: string,
    persistLocalAsDefault: boolean
  ) => {
    const account = accounts.find((a) => a.username === username);
    try {
      if (persistLocalAsDefault && account?.type === "local") {
        await SetDefaultAccount(username);
        GetAccounts().then(setAccounts).catch(console.error);
      }
      if (account?.type === "microsoft") {
        await LoginAccount(false);
      }
    } catch (err) {
      console.error("prepareGameAccountForLaunch:", err);
    }
  };

  const doLaunch = async (
    disabledMods: string[] = [],
    launchParams?: {
      instanceName: string;
      serverAddress: string;
      serverID: number;
      syncConfigFromServer: boolean;
      selectedAccountName: string;
    },
    enabledResourcepacksOrder?: string[]
  ) => {
    const params = launchParams || pendingLaunch;
    if (!params) return;
    const lp = params as {
      instanceName: string;
      serverAddress: string;
      serverID: number;
      serverName?: string;
      syncConfigFromServer: boolean;
      selectedAccountName: string;
    };
    const instanceName = lp.instanceName ?? "";
    const serverAddress = lp.serverAddress ?? "";
    const serverID = lp.serverID ?? 0;
    const sync = lp.syncConfigFromServer ?? syncConfigFromServer;
    const account = lp.selectedAccountName ?? "";
    setShowLaunchProgressDialog(true);
    setLaunchProgress({ message: "Запуск инстанса..." });
    try {
      const disabledModsJSON = JSON.stringify(disabledMods);
      const enabledRpOrderJSON = enabledResourcepacksOrder && enabledResourcepacksOrder.length > 0
        ? JSON.stringify(enabledResourcepacksOrder)
        : "";
      const result = await LaunchInstanceWithAccount(
        instanceName || "",
        serverAddress || "",
        serverID || 0,
        sync,
        account || "",
        disabledModsJSON,
        enabledRpOrderJSON,
        lp.serverName ?? ""
      );
      if (result && result.toLowerCase().includes("error")) {
        setLaunchProgress({ message: `Ошибка: ${result}`, type: "error" });
        setTimeout(() => setShowLaunchProgressDialog(false), 3000);
      } else {
        setLaunchProgress({ message: "Minecraft запущен успешно!", type: "success" });
        setTimeout(() => setShowLaunchProgressDialog(false), 2000);
      }
    } catch (error) {
      setLaunchProgress({ message: `Ошибка запуска: ${error}`, type: "error" });
      setTimeout(() => setShowLaunchProgressDialog(false), 3000);
    }
    GetInstances()
      .then((raw) => setInstances(normalizeInstancesList(raw)))
      .catch(() => setInstances([]));
  };

  const launchWithSelectedGameAccount = async (
    target: {
      instanceName?: string;
      serverAddress?: string;
      serverID?: number;
      serverName?: string;
    },
    gameAccountUsername: string,
    syncFromServer: boolean
  ) => {
    await prepareGameAccountForLaunch(gameAccountUsername, false);
    const serverID = target.serverID || 0;
    const launchParams = {
      instanceName: target.instanceName || "",
      serverAddress: target.serverAddress || "",
      serverID,
      serverName: target.serverName || "",
      syncConfigFromServer: syncFromServer,
      selectedAccountName: gameAccountUsername || "",
    };
    if (serverID > 0 && apiBase) {
      setPendingLaunch(launchParams);
      setShowModSelectionDialog(true);
    } else {
      doLaunch([], launchParams);
    }
  };

  const handleModSelectionConfirm = async (
    disabledMods: string[],
    enabledResourcepacksOrder?: string[],
    launchMeta?: LaunchConfirmMeta
  ) => {
    const pl = pendingLaunch;
    setShowModSelectionDialog(false);
    if (!pl) return;
    const username = launchMeta?.username ?? pl.selectedAccountName;
    const sync = launchMeta?.syncFromServer ?? pl.syncConfigFromServer;
    const saveDef = launchMeta?.saveAsDefault ?? false;
    await prepareGameAccountForLaunch(username, saveDef);
    const launchParams = {
      ...pl,
      selectedAccountName: username,
      syncConfigFromServer: sync,
    };
    setPendingLaunch(null);
    doLaunch(disabledMods, launchParams, enabledResourcepacksOrder);
  };

  const handleModSelectionCancel = () => {
    setShowModSelectionDialog(false);
    setPendingLaunch(null);
    setSyncConfigFromServer(false);
    setShowLaunchProgressDialog(false);
  };

  const renderAccounts = () => {
    const cloudAccounts = accounts.filter((a) => a.type === "cloud");
    const microsoftAccounts = accounts.filter((a) => a.type === "microsoft");
    return (
      <div className="space-y-4">
        {/* Информация о подключённых аккаунтах (QMServer Cloud, Microsoft) */}
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <div className="p-4 border-b border-border">
            <h3 className="font-medium text-sm text-muted-foreground">{t.authConnectedAccounts}</h3>
          </div>
          <div className="p-4 space-y-3">
            {/* QMServer Cloud */}
            {cloudAccounts.length > 0 ? cloudAccounts.map((acc, i) => (
              <div
                key={`cloud-${i}`}
                className="flex items-center justify-between rounded-md bg-muted/50 px-4 py-3 cursor-pointer hover:bg-muted/70 transition-colors"
                onClick={() => {
                  setAccountSummaryTarget({ type: "cloud", username: acc.username, email: acc.email });
                  setShowAccountSummaryDialog(true);
                }}
              >
                <div className="flex-1">
                  <p className="font-medium text-sm">{t.authCloud}</p>
                  <p className="text-xs text-muted-foreground">{acc.email || acc.username}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteAccount(acc);
                    }}
                  >
                    <LogOut className="w-4 h-4 mr-1" />
                  </Button>
                </div>
              </div>
            )) : (
              <div className="flex items-center justify-between rounded-md bg-muted/50 px-4 py-3">
                <div>
                  <p className="font-medium text-sm">{t.authCloud}</p>
                  <p className="text-xs text-muted-foreground">QMServer Cloud</p>
                </div>
                <Button size="sm" variant="default" onClick={handleLoginQMServerCloud}>
                  <LogIn className="w-4 h-4 mr-2" />
                  Войти
                </Button>
              </div>
            )}
            {/* Microsoft / Mojang — только если настроен QMLAUNCHER_MSA_CLIENT_ID */}
            {microsoftAuthAvailable && (
              <>
                {microsoftAccounts.length > 0 ? microsoftAccounts.map((acc, i) => (
                  <div
                    key={`ms-${i}`}
                    className="flex items-center justify-between rounded-md bg-muted/50 px-4 py-3 cursor-pointer hover:bg-muted/70 transition-colors"
                    onClick={() => {
                      setAccountSummaryTarget({ type: "microsoft", username: acc.username });
                      setShowAccountSummaryDialog(true);
                    }}
                  >
                    <div className="flex-1 min-w-0">
                      <p className="font-medium text-sm">{t.authMicrosoft}</p>
                      <p className="text-xs text-muted-foreground truncate">{acc.username}</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {acc.status === "active" ? null : (
                        <span
                          className="text-xs text-amber-600 dark:text-amber-400 cursor-help underline decoration-dotted underline-offset-2"
                          title={t.authStatusExpiredTooltip}
                        >
                          {t.authStatusExpired}
                        </span>
                      )}
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleLogout();
                        }}
                      >
                        <LogOut className="w-4 h-4 mr-1" />
                        {t.authLogout}
                      </Button>
                    </div>
                  </div>
                )) : (
                  <div className="flex items-center justify-between rounded-md bg-muted/50 px-4 py-3">
                    <div>
                      <p className="font-medium text-sm">{t.authMicrosoft}</p>
                      <p className="text-xs text-muted-foreground">Microsoft (Mojang)</p>
                    </div>
                    <Button size="sm" variant="default" onClick={handleLoginMicrosoft}>
                      <LogIn className="w-4 h-4 mr-2" />
                      Войти
                    </Button>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    );
  };

  const renderGameAccounts = () => {
    // Игровые профили для запуска: local, cloud_game и microsoft (лицензия), без строки QMServer Cloud (email).
    const gameAccounts = accounts.filter(isLaunchableGameAccount);

    return (
      <div className="space-y-4">
        <div className="bg-card border border-border rounded-lg overflow-hidden">
          <div className="p-4 border-b border-border flex justify-end items-center">
            <div className="flex gap-2">
              {!accounts.some((a) => a.type === "cloud") && (
                <Button variant="secondary" size="sm" onClick={handleLoginQMServerCloud}>
                  <Plus className="w-4 h-4 mr-2" />
                  {t.authAddCloud}
                </Button>
              )}
              <Button variant="secondary" size="sm" onClick={handleAddAccountClick}>
                <Plus className="w-4 h-4 mr-2" />
                {t.authCreate}
              </Button>
            </div>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t.authTableUsername}
                </TableHead>
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t.authTableType}
                </TableHead>
                <TableHead className="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  {t.authTableActions}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {gameAccounts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-center py-8 text-muted-foreground">
                    {t.serversNotFound.replace(/servers/gi, "accounts")}
                  </TableCell>
                </TableRow>
              ) : (
                gameAccounts.map((account, index) => (
                <TableRow
                  key={`${account.type}-${account.username}-${index}`}
                  className="cursor-pointer hover:bg-accent/40"
                  onClick={() => {
                    if (account.type === "local" && account.username) {
                      handleSetDefaultAccount(account.username);
                    } else if (account.type === "cloud_game" && account.gameAccountId) {
                      setInventoryAccount({ username: account.username, gameAccountId: account.gameAccountId });
                      setInventoryData(null);
                      setShowInventoryDialog(true);
                      GetGameAccountInventory(account.gameAccountId).then((res: any) => {
                        if (res.error) setInventoryData({ inventories: [], error: res.error });
                        else setInventoryData({ inventories: res.inventories || [] });
                      });
                    }
                  }}
                >
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm text-foreground">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 shrink-0 overflow-hidden rounded-md bg-muted/50">
                        {isSkinTextureUrl(getAccountSkinUrl(account)) ? (
                          <div
                            className="size-full"
                            style={skinTextureHeadStyle(apiBase ? proxiedUrl(getAccountSkinUrl(account), apiBase) : getAccountSkinUrl(account), 32)}
                          />
                        ) : (
                          <img
                            src={apiBase ? proxiedUrl(getAccountSkinUrl(account), apiBase) : getAccountSkinUrl(account)}
                            alt=""
                            className="w-full h-full object-contain"
                            style={{ imageRendering: "pixelated" }}
                            onError={(e) => {
                              const model = account.skinModel === "alex" ? "Alex" : "Steve";
                              e.currentTarget.src = apiBase ? proxiedUrl(`https://mc-heads.net/avatar/MHF_${model}/32`, apiBase) : `https://mc-heads.net/avatar/MHF_${model}/32`;
                              e.currentTarget.onerror = null;
                            }}
                          />
                        )}
                      </div>
                      <span>{account.username || "-"}</span>
                      {account.isDefault && (
                        <span className="text-xs px-1.5 py-0.5 rounded bg-primary/20 text-primary border border-primary/30">
                          {t.statusDefault}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm text-foreground">
                    {account.type === "microsoft" ? t.authMicrosoft : account.type === "cloud" ? t.authCloud : account.type === "cloud_game" ? "QMServer Cloud" : t.authLocal}
                  </TableCell>
                  <TableCell className="px-4 py-4 whitespace-nowrap text-sm">
                    <div className="flex items-center gap-2">
                      {account.type === "microsoft" && account.status !== "none" && (
                        <>
                          {accounts.some((a) => a.type === "cloud") &&
                            (isMicrosoftLinkedToCloudGame(accounts) ? (
                              <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground max-w-[220px]">
                                <Check className="w-4 h-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden />
                                {t.authSyncMicrosoftCloudLinked}
                              </span>
                            ) : (
                              <Button
                                size="sm"
                                variant="secondary"
                                onClick={async (e) => {
                                  e.stopPropagation();
                                  try {
                                    const result = await SyncMicrosoftAccountToCloud();
                                    showAlert(t.authSyncMicrosoftCloudResultTitle, result);
                                    void GetAccounts().then(setAccounts);
                                  } catch (err) {
                                    showAlert("Error", String(err));
                                  }
                                }}
                              >
                                <CloudUpload className="w-4 h-4 mr-2" />
                                {t.authSyncMicrosoftCloudBtn}
                              </Button>
                            ))}
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleLogout();
                            }}
                          >
                            <LogOut className="w-4 h-4 mr-2" />
                            {t.authLogout}
                          </Button>
                        </>
                      )}
                      {account.type === "local" && (
                        <>
                          {accounts.some(a => a.type === "cloud") && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={async (e) => {
                                e.stopPropagation();
                                setShowSyncSkinDialog(account);
                                setSyncSkinProvider("");
                                setSyncSkinUrl("");
                                try {
                                  const [prov, ely] = await Promise.all([GetSkinProviderConfig().catch(() => ({})), GetCloudElyLinked().catch(() => false)]);
                                  setSkinProviders(prov && Object.keys(prov).length ? prov : { ely_by: true });
                                  setCloudElyLinked(ely);
                                } catch {
                                  setSkinProviders({ ely_by: true });
                                  setCloudElyLinked(false);
                                }
                              }}
                            >
                              <CloudUpload className="w-4 h-4 mr-2" />
                              Синхр.
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteAccount(account);
                            }}
                          >
                            <Trash2 className="w-4 h-4 mr-2" />
                            {t.authDelete}
                          </Button>
                        </>
                      )}
                      {account.type === "cloud" && (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={async (e) => {
                              e.stopPropagation();
                              try {
                                const [result, prov, ely] = await Promise.all([
                                  GetCloudGameAccounts(),
                                  GetSkinProviderConfig().catch(() => ({})),
                                  GetCloudElyLinked().catch(() => false),
                                ]);
                                if (typeof result === "string") {
                                  showAlert("Ошибка", result);
                                  return;
                                }
                                setSkinProviders(prov && Object.keys(prov).length ? prov : { ely_by: true });
                                setCloudElyLinked(ely);
                                const mapped = result.map((r: any) => ({ id: r.id, username: r.username, uuid: r.uuid, skinModel: r.skinModel || "steve", skinUrl: r.skinUrl || "" }));
                                setCloudGameAccounts(mapped);
                                const first = mapped[0];
                                if (first) {
                                  setEditCloudAccountId(first.id);
                                  setEditCloudSkinUrl(first.skinUrl || "");
                                  setEditCloudSkinModel((first.skinModel as "steve" | "alex") || "steve");
                                  const detected = detectProviderFromUrl(first.skinUrl);
                                  const opts = ["ely_by"].filter((k) => (k === "ely_by" ? ely : true) && ((prov as Record<string, boolean>)?.[k] ?? true));
                                  setEditCloudSkinProvider(opts.includes(detected) ? detected : "custom");
                                } else {
                                  setEditCloudAccountId(null);
                                  setEditCloudSkinProvider("");
                                }
                                setShowEditCloudSkinDialog(true);
                              } catch (err) {
                                showAlert("Ошибка", String(err));
                              }
                            }}
                          >
                            Скин
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteAccount(account);
                            }}
                          >
                            <LogOut className="w-4 h-4 mr-2" />
                            Отключить
                          </Button>
                        </>
                      )}
                      {account.type === "cloud_game" && (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={async (e) => {
                              e.stopPropagation();
                              try {
                                const [result, prov, ely] = await Promise.all([
                                  GetCloudGameAccounts(),
                                  GetSkinProviderConfig().catch(() => ({})),
                                  GetCloudElyLinked().catch(() => false),
                                ]);
                                if (typeof result === "string") {
                                  showAlert("Ошибка", result);
                                  return;
                                }
                                setSkinProviders(prov && Object.keys(prov).length ? prov : { ely_by: true });
                                setCloudElyLinked(ely);
                                const mapped = result.map((r: any) => ({ id: r.id, username: r.username, uuid: r.uuid, skinModel: r.skinModel || "steve", skinUrl: r.skinUrl || "" }));
                                setCloudGameAccounts(mapped);
                                // Find the current account by username
                                const current = mapped.find((r: any) => r.username === account.username);
                                if (current) {
                                  setEditCloudAccountId(current.id);
                                  setEditCloudSkinUrl(current.skinUrl || "");
                                  setEditCloudSkinModel((current.skinModel as "steve" | "alex") || "steve");
                                  const detected = detectProviderFromUrl(current.skinUrl);
                                  const opts = ["ely_by"].filter((k) => (k === "ely_by" ? ely : true) && ((prov as Record<string, boolean>)?.[k] ?? true));
                                  setEditCloudSkinProvider(opts.includes(detected) ? detected : "custom");
                                } else {
                                  setEditCloudAccountId(null);
                                  setEditCloudSkinProvider("");
                                }
                                setShowEditCloudSkinDialog(true);
                              } catch (err) {
                                showAlert("Ошибка", String(err));
                              }
                            }}
                          >
                            Скин
                          </Button>
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleDeleteAccount(account);
                            }}
                          >
                            <Trash2 className="w-4 h-4 mr-2" />
                            {t.authDelete}
                          </Button>
                        </>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
    );
  };

  // Handle navigation
  useEffect(() => {
    const handleHashChange = () => {
      const { tab, instanceNameFromRoute, resourceStoreCategory: rsCat } = parseLocationHash(
        window.location.hash
      );
      setActiveTab(tab);
      if (instanceNameFromRoute !== undefined) {
        setSelectedInstanceName(instanceNameFromRoute);
      } else if (!window.location.hash.replace(/^#/, "").trim()) {
        setActiveTab("news");
      }
      if (rsCat !== undefined) {
        setResourceStoreCategory(rsCat);
      }
    };

    // Check initial hash
    handleHashChange();

    // Listen for hash changes
    window.addEventListener('hashchange', handleHashChange);
    return () => window.removeEventListener('hashchange', handleHashChange);
  }, []);

  const handleNoAccountAdd = () => {
    setShowNoAccountDialog(false);
    setShowAccountTypeDialog(true);
  };

  const handleNoAccountCancel = () => {
    setShowNoAccountDialog(false);
  };

  return (
    <ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
      {activeTab === "resource-store" ? (
        <ResourceStoreBrowser
          instanceName={selectedInstanceName}
          category={resourceStoreCategory}
          onInstallSuccess={() => {
            if (selectedInstanceName) {
              GetInstanceDetails(selectedInstanceName).then(setInstanceDetails).catch(() => {});
            }
          }}
          onClose={() => {
            if (selectedInstanceName) {
              GetInstanceDetails(selectedInstanceName).then(setInstanceDetails).catch(() => {});
            }
            const openedAsPopout = !!window.opener;
            try {
              window.close();
            } catch {
              /* no-op */
            }
            if (!openedAsPopout) {
              if (selectedInstanceName) {
                window.location.hash = `#instance-resources/${encodeURIComponent(selectedInstanceName)}`;
              } else {
                window.location.hash = "#instances";
              }
            }
          }}
        />
      ) : (
      <SidebarProvider
        style={
          {
            "--sidebar-width": "calc(var(--spacing) * 56)",
            "--header-height": "calc(var(--spacing) * 12)",
          } as CSSProperties
        }
      >
        <AppSidebar
          variant="inset"
          hasAccounts={accounts.length > 0}
          onLogout={handleLogout}
          onAbout={() => setShowAboutDialog(true)}
          currentAccount={{
            ...currentAccount,
            avatar: (() => {
              if (currentAccountType === "cloud") {
                return cloudProfileAvatar || undefined;
              }
              if (currentAccountType === "microsoft") {
                const acc = accounts.find((a) => a.username === currentAccount?.name);
                const url = acc ? getAccountSkinUrl(acc) : undefined;
                return url && apiBase ? proxiedUrl(url, apiBase) : url;
              }
              return undefined;
            })(),
            avatarIsSkinTexture: (() => {
              if (currentAccountType === "cloud") return false;
              if (currentAccountType === "microsoft") {
                const acc = accounts.find((a) => a.username === currentAccount?.name);
                return acc ? isSkinTextureUrl(getAccountSkinUrl(acc)) : false;
              }
              return false;
            })(),
            isCloudPremium: currentAccountType === "cloud" && cloudIsPremium,
          }}
          accountClickable={true}
          disabled={showLaunchProgressDialog}
        />
        <SidebarInset>
          <SiteHeader
            versionLabel={launcherVersion}
            title={
              activeTab === 'news'
                ? t.news
                : activeTab === 'servers'
                  ? t.servers
                  : activeTab === 'instances'
                    ? t.instances
                    : activeTab === 'instance'
                      ? selectedInstanceName || t.instances
                      : activeTab === 'instance-settings'
                        ? `Настройки: ${selectedInstanceName || "…"}`
                        : activeTab === 'instance-resources'
                          ? `Ресурсы: ${selectedInstanceName || "…"}`
                          : activeTab === 'accounts'
                            ? t.account
                            : activeTab === 'settings'
                              ? t.settings
                              : t.gameAccounts
            }
          />
          <div className="flex flex-1 flex-col">
            <div className="@container/main flex flex-1 flex-col gap-2">
              <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
                <div className="px-4 lg:px-6">
                  {activeTab === 'news' && renderNews()}
                  {activeTab === 'servers' && renderServers()}
                  {activeTab === 'instances' && renderInstances()}
                  {activeTab === 'instance' && renderInstancePage()}
                  {activeTab === 'instance-settings' && renderInstanceSettingsPage()}
                  {activeTab === 'instance-resources' && renderInstanceResourcesPage()}
                  {activeTab === 'accounts' && renderAccounts()}
                  {activeTab === 'game-accounts' && renderGameAccounts()}
                  {activeTab === 'settings' && renderSettings()}
                </div>
              </div>
            </div>
          </div>
        </SidebarInset>
      </SidebarProvider>
      )}

      {selectedServerDetails && (
        <Dialog
          open={!!selectedServerDetails}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedServerDetails(null);
            }
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Информация о сервере</DialogTitle>
              <DialogDescription>
                Подробная информация о выбранном сервере Minecraft.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2 py-2 text-sm">
              <div className="flex justify-between gap-4 items-center">
                <span className="text-muted-foreground">Название:</span>
                <span className="font-medium text-foreground inline-flex items-center gap-2 flex-wrap justify-end">
                  {selectedServerDetails.name}
                  <McServerPingBadge
                    server={selectedServerDetails}
                    onlineLabel={t.badgeOnline}
                    offlineLabel={t.badgeOffline}
                  />
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">IP:</span>
                <span className="font-medium text-foreground">
                  {selectedServerDetails.address}:{selectedServerDetails.port}
                </span>
              </div>
              <div className="flex justify-between gap-4 items-center">
                <span className="text-muted-foreground">Статус:</span>
                <span className="font-medium flex flex-wrap justify-end items-center">
                  {selectedServerDetails.enabled === false ? (
                    <span className="text-destructive">{t.serverDisabledShort}</span>
                  ) : (
                    <McServerPingBadge
                      server={selectedServerDetails}
                      onlineLabel={t.badgeOnline}
                      offlineLabel={t.badgeOffline}
                    />
                  )}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Версия игры:</span>
                <span className="font-medium text-foreground">{selectedServerDetails.version || "-"}</span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Загрузчик:</span>
                <span className="font-medium text-foreground">
                  {selectedServerDetails.modLoader
                    ? formatLoaderDisplayName(selectedServerDetails.modLoader)
                    : "-"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Версия загрузчика:</span>
                <span className="font-medium text-foreground">
                  {selectedServerDetails.modLoaderVersion || "-"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Игроки:</span>
                <span className="font-medium text-foreground">
                  {selectedServerDetails.enabled === false ||
                  selectedServerDetails.gameServerOnline === false
                    ? "—"
                    : `${selectedServerDetails.players}/${selectedServerDetails.maxPlayers}`}
                </span>
              </div>
              {(selectedServerDetails.serverID ?? 0) > 0 && apiBase && (
                <div className="pt-3 border-t">
                  <ServerModList
                    serverID={selectedServerDetails.serverID ?? 0}
                    apiBase={apiBase}
                    onModClick={(mod) => setSelectedModDetail({ path: mod.path, name: mod.name, meta: mod.meta })}
                  />
                </div>
              )}
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setSelectedServerDetails(null)}
              >
                Закрыть
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      <ModDetailDialog
        mod={selectedModDetail}
        open={!!selectedModDetail}
        onOpenChange={(open) => !open && setSelectedModDetail(null)}
      />

      {selectedInstanceDetails && (
        <Dialog
          open={!!selectedInstanceDetails}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedInstanceDetails(null);
            }
          }}
        >
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Информация об инстансе</DialogTitle>
              <DialogDescription>
                Сводная информация о выбранном инстансе Minecraft.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2 py-2 text-sm">
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Название:</span>
                <span className="font-medium text-foreground">
                  {selectedInstanceDetails.name || "Без имени"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">UUID:</span>
                <span className="font-mono text-xs text-foreground truncate max-w-[220px]">
                  {selectedInstanceDetails.uuid || "-"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Версия Minecraft:</span>
                <span className="font-medium text-foreground">
                  {selectedInstanceDetails.gameVersion || "-"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Загрузчик:</span>
                <span className="font-medium text-foreground">
                  {selectedInstanceDetails.loader
                    ? formatLoaderLabel(
                        selectedInstanceDetails.loader,
                        selectedInstanceDetails.loaderVersion
                      )
                    : "Vanilla"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Папка инстанса:</span>
                <span className="font-mono text-xs text-foreground truncate max-w-[220px]">
                  {selectedInstanceDetails.dir || "-"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Последний сервер:</span>
                <span className="font-medium text-foreground break-all text-right">
                  {selectedInstanceDetails.lastServer || "-"}
                  {selectedInstanceDetails.isUsingQMServerCloud && " (QMServer Cloud)"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Аккаунт:</span>
                <span className="font-medium text-foreground">
                  {selectedInstanceDetails.lastUser || "-"}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Mods:</span>
                <span className="font-medium text-foreground">
                  {Array.isArray(selectedInstanceDetails.mods)
                    ? selectedInstanceDetails.mods.length
                    : 0}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Shaderpacks:</span>
                <span className="font-medium text-foreground">
                  {Array.isArray(selectedInstanceDetails.shaderpacks)
                    ? selectedInstanceDetails.shaderpacks.length
                    : 0}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Resourcepacks:</span>
                <span className="font-medium text-foreground">
                  {Array.isArray(selectedInstanceDetails.resourcepacks)
                    ? selectedInstanceDetails.resourcepacks.length
                    : 0}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Schematics:</span>
                <span className="font-medium text-foreground">
                  {Array.isArray(selectedInstanceDetails.schematics)
                    ? selectedInstanceDetails.schematics.length
                    : 0}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Config файлов:</span>
                <span className="font-medium text-foreground">
                  {Array.isArray(selectedInstanceDetails.configFiles)
                    ? selectedInstanceDetails.configFiles.length
                    : 0}
                </span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">KubeJS файлов:</span>
                <span className="font-medium text-foreground">
                  {Array.isArray(selectedInstanceDetails.kubejsFiles)
                    ? selectedInstanceDetails.kubejsFiles.length
                    : 0}
                </span>
              </div>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setSelectedInstanceDetails(null)}
              >
                Закрыть
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      <Dialog open={showNoAccountDialog} onOpenChange={setShowNoAccountDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t.noAccountTitle}</DialogTitle>
            <DialogDescription>
              {t.noAccountDescription}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={handleNoAccountCancel}>
              {t.noAccountCancel}
            </Button>
            <Button onClick={handleNoAccountAdd}>
              {t.noAccountAdd}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showCreateAccountDialog} onOpenChange={setShowCreateAccountDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t.createAccountTitle}</DialogTitle>
            <DialogDescription>
              {t.createAccountDescription}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="account-name">{t.authTableUsername}</Label>
              <Input
                id="account-name"
                value={newAccountName}
                onChange={(e) => setNewAccountName(e.target.value)}
                placeholder={t.authTableUsername}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleCreateLocalAccount();
                  }
                }}
                autoFocus
              />
            </div>
            <div className="grid gap-2">
              <Label>{t.createAccountSkinModel}</Label>
              <div className="flex gap-4">
                <label
                  className={`flex flex-col items-center gap-2 p-3 rounded-lg border-2 cursor-pointer transition-colors ${
                    newAccountSkinModel === "steve"
                      ? "border-primary bg-primary/5"
                      : "border-muted hover:border-muted-foreground/50"
                  }`}
                >
                  <input
                    type="radio"
                    name="skinModel"
                    value="steve"
                    checked={newAccountSkinModel === "steve"}
                    onChange={() => setNewAccountSkinModel("steve")}
                    className="sr-only"
                  />
                  <img
                    src="https://mc-heads.net/avatar/MHF_Steve/64"
                    alt="Steve"
                    className="w-16 h-16 object-contain pixelated"
                    style={{ imageRendering: "pixelated" }}
                  />
                  <span className="text-sm font-medium">{t.createAccountSkinSteve}</span>
                </label>
                <label
                  className={`flex flex-col items-center gap-2 p-3 rounded-lg border-2 cursor-pointer transition-colors ${
                    newAccountSkinModel === "alex"
                      ? "border-primary bg-primary/5"
                      : "border-muted hover:border-muted-foreground/50"
                  }`}
                >
                  <input
                    type="radio"
                    name="skinModel"
                    value="alex"
                    checked={newAccountSkinModel === "alex"}
                    onChange={() => setNewAccountSkinModel("alex")}
                    className="sr-only"
                  />
                  <img
                    src="https://mc-heads.net/avatar/MHF_Alex/64"
                    alt="Alex"
                    className="w-16 h-16 object-contain pixelated"
                    style={{ imageRendering: "pixelated" }}
                  />
                  <span className="text-sm font-medium">{t.createAccountSkinAlex}</span>
                </label>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCreateAccountCancel}>
              {t.noAccountCancel}
            </Button>
            <Button onClick={handleCreateLocalAccount} disabled={!newAccountName || !newAccountName.trim()}>
              {t.createAccountCreate}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showAccountTypeDialog} onOpenChange={(open) => {
        if (!open) setShowAccountTypeDialog(false);
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {accounts.length === 0
                ? "Добавьте первый аккаунт"
                : "Выберите тип аккаунта"}
            </DialogTitle>
            <DialogDescription>
              {accounts.length === 0
                ? "Выберите способ: локальный ник, вход через QMServer Cloud или Microsoft (Mojang)."
                : "Выберите тип создаваемого игрового аккаунта"}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <button
              type="button"
              className="flex items-center gap-4 p-4 rounded-lg border-2 border-border hover:border-primary transition-colors text-left"
              onClick={handleCreateLocalAccountClick}
            >
              <div className="w-12 h-12 shrink-0 bg-primary/10 rounded-lg flex items-center justify-center">
                <User className="w-6 h-6 text-primary" />
              </div>
              <div>
                <h3 className="font-medium text-foreground">Локальный аккаунт</h3>
                <p className="text-sm text-muted-foreground">Офлайн-ник для локальной игры без входа на сервер</p>
              </div>
            </button>
            {accounts.length === 0 ? (
              <button
                type="button"
                className="flex items-center gap-4 p-4 rounded-lg border-2 border-border hover:border-primary transition-colors text-left"
                onClick={() => {
                  setShowAccountTypeDialog(false);
                  handleLoginQMServerCloud();
                }}
              >
                <div className="w-12 h-12 shrink-0 bg-primary/10 rounded-lg flex items-center justify-center">
                  <Cloud className="w-6 h-6 text-primary" />
                </div>
                <div>
                  <h3 className="font-medium text-foreground">Аккаунт QMServer Cloud</h3>
                  <p className="text-sm text-muted-foreground">Войти через браузер (учётная запись облака проекта)</p>
                </div>
              </button>
            ) : null}
            {accounts.length === 0 && microsoftAuthAvailable ? (
              <button
                type="button"
                className="flex items-center gap-4 p-4 rounded-lg border-2 border-border hover:border-primary transition-colors text-left"
                onClick={() => {
                  setShowAccountTypeDialog(false);
                  handleLoginMicrosoft();
                }}
              >
                <div className="w-12 h-12 shrink-0 bg-primary/10 rounded-lg flex items-center justify-center">
                  <LogIn className="w-6 h-6 text-primary" />
                </div>
                <div>
                  <h3 className="font-medium text-foreground">Microsoft (Mojang)</h3>
                  <p className="text-sm text-muted-foreground">Официальный аккаунт Minecraft для лицензионной игры</p>
                </div>
              </button>
            ) : null}
            {accounts.length > 0 && accounts.some((a) => a.type === "cloud") ? (
              <button
                type="button"
                className="flex items-center gap-4 p-4 rounded-lg border-2 border-border hover:border-primary transition-colors text-left"
                onClick={handleCreateCloudGameAccountClick}
              >
                <div className="w-12 h-12 shrink-0 bg-primary/10 rounded-lg flex items-center justify-center">
                  <Cloud className="w-6 h-6 text-primary" />
                </div>
                <div>
                  <h3 className="font-medium text-foreground">Новый игровой аккаунт QMServer Cloud</h3>
                  <p className="text-sm text-muted-foreground">Дополнительный игровой профиль в облаке</p>
                </div>
              </button>
            ) : null}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAccountTypeDialog(false)}>
              Отмена
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showCloudGameAccountDialog} onOpenChange={setShowCloudGameAccountDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Создать QMServer Cloud аккаунт</DialogTitle>
            <DialogDescription>
              Создайте новый игровой аккаунт в QMServer Cloud
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="cloud-account-name">{t.authTableUsername}</Label>
              <Input
                id="cloud-account-name"
                value={cloudGameAccountName}
                onChange={(e) => setCloudGameAccountName(e.target.value)}
                placeholder={t.authTableUsername}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    handleCreateCloudGameAccount();
                  }
                }}
                autoFocus
              />
            </div>
            <div className="grid gap-2">
              <Label>{t.createAccountSkinModel}</Label>
              <div className="flex gap-4">
                <label
                  className={`flex flex-col items-center gap-2 p-3 rounded-lg border-2 cursor-pointer transition-colors ${
                    cloudGameAccountSkinModel === "steve"
                      ? "border-primary bg-primary/5"
                      : "border-muted hover:border-muted-foreground/50"
                  }`}
                >
                  <input
                    type="radio"
                    name="cloudSkinModel"
                    value="steve"
                    checked={cloudGameAccountSkinModel === "steve"}
                    onChange={() => setCloudGameAccountSkinModel("steve")}
                    className="sr-only"
                  />
                  <img
                    src="https://mc-heads.net/avatar/MHF_Steve/64"
                    alt="Steve"
                    className="w-16 h-16 object-contain pixelated"
                    style={{ imageRendering: "pixelated" }}
                  />
                  <span className="text-sm font-medium">{t.createAccountSkinSteve}</span>
                </label>
                <label
                  className={`flex flex-col items-center gap-2 p-3 rounded-lg border-2 cursor-pointer transition-colors ${
                    cloudGameAccountSkinModel === "alex"
                      ? "border-primary bg-primary/5"
                      : "border-muted hover:border-muted-foreground/50"
                  }`}
                >
                  <input
                    type="radio"
                    name="cloudSkinModel"
                    value="alex"
                    checked={cloudGameAccountSkinModel === "alex"}
                    onChange={() => setCloudGameAccountSkinModel("alex")}
                    className="sr-only"
                  />
                  <img
                    src="https://mc-heads.net/avatar/MHF_Alex/64"
                    alt="Alex"
                    className="w-16 h-16 object-contain pixelated"
                    style={{ imageRendering: "pixelated" }}
                  />
                  <span className="text-sm font-medium">{t.createAccountSkinAlex}</span>
                </label>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCloudGameAccountCancel}>
              {t.noAccountCancel}
            </Button>
            <Button onClick={handleCreateCloudGameAccount} disabled={!cloudGameAccountName || !cloudGameAccountName.trim()}>
              {t.createAccountCreate}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ModSelectionDialog
        open={showModSelectionDialog}
        onOpenChange={(open) => {
          if (!open) handleModSelectionCancel();
        }}
        serverID={pendingLaunch?.serverID ?? 0}
        serverName={pendingLaunch?.serverName ?? ""}
        apiBase={apiBase}
        onConfirm={handleModSelectionConfirm}
        onCancel={handleModSelectionCancel}
        launchAccounts={launchableAccounts.map((a) => ({
          username: a.username,
          type: a.type,
          isDefault: a.isDefault,
        }))}
        showLaunchAccountSection={launchableAccounts.length > 1}
        initialAccountUsername={pendingLaunch?.selectedAccountName ?? ""}
        initialSyncFromServer={pendingLaunch?.syncConfigFromServer ?? false}
        showSyncFromServer={(pendingLaunch?.serverID ?? 0) > 0}
      />

      <Dialog open={showLaunchProgressDialog} onOpenChange={(open) => {
        // Prevent closing dialog during launch process
        if (!open && launchProgress.type !== "success" && launchProgress.type !== "error") {
          return;
        }
        setShowLaunchProgressDialog(open);
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Установка и запуск Minecraft</DialogTitle>
            <DialogDescription>
              Пожалуйста, подождите...
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between text-sm">
                  <span
                    className={
                      launchProgress.type === "success"
                        ? "text-green-500"
                        : launchProgress.type === "error"
                          ? "text-red-500"
                          : "text-foreground"
                    }
                  >
                    {launchProgress.type === "success" ? "✓ " : launchProgress.type === "error" ? "✗ " : ""}
                    {launchProgress.message || "Подготовка..."}
                  </span>
                  {launchProgress.progress !== undefined &&
                    launchProgress.type !== "success" &&
                    launchProgress.type !== "error" && (
                      <span className="text-muted-foreground shrink-0 ml-2">{Math.round(launchProgress.progress)}%</span>
                    )}
                </div>
                {launchProgress.currentFile &&
                  launchProgress.type !== "success" &&
                  launchProgress.type !== "error" && (
                    <p className="text-muted-foreground text-xs truncate" title={launchProgress.currentFile}>
                      {launchProgress.currentFile}
                    </p>
                  )}
              </div>
              {(launchProgress.progress !== undefined || launchProgress.type === "sync-progress") &&
                launchProgress.type !== "success" &&
                launchProgress.type !== "error" && (
                  <Progress value={launchProgress.progress ?? 0} className="h-2" />
                )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={showDeleteInstanceDialog} onOpenChange={(open) => {
        if (!open) {
          setShowDeleteInstanceDialog(false);
          setInstanceNameToDelete(null);
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Удалить инстанс</DialogTitle>
            <DialogDescription>
              {instanceNameToDelete
                ? `Удалить инстанс «${instanceNameToDelete}» со всеми файлами? Это действие нельзя отменить.`
                : ""}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowDeleteInstanceDialog(false); setInstanceNameToDelete(null); }}>
              Отмена
            </Button>
            <Button variant="destructive" onClick={confirmDeleteInstance}>
              Удалить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showDeleteAccountDialog} onOpenChange={(open) => {
        if (!open) {
          setShowDeleteAccountDialog(false);
          setAccountToDelete(null);
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Удаление аккаунта</DialogTitle>
            <DialogDescription>
              {accountToDelete ? (accountToDelete.type === "cloud"
                ? `Отключить аккаунт "${accountToDelete.username}" от QMLauncher?`
                : `${t.authDelete} "${accountToDelete.username}"? Данные аккаунта (config, saves) будут удалены.`) : ""}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setShowDeleteAccountDialog(false); setAccountToDelete(null); }}>
              Отмена
            </Button>
            <Button variant="destructive" onClick={confirmDeleteAccount}>
              Удалить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showLogoutConfirmDialog} onOpenChange={(open) => {
        if (!open) setShowLogoutConfirmDialog(false);
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Выход из аккаунта</DialogTitle>
            <DialogDescription>
              {t.authLogout}?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowLogoutConfirmDialog(false)}>
              Отмена
            </Button>
            <Button variant="destructive" onClick={confirmLogout}>
              Выйти
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showAlertDialog} onOpenChange={(open) => {
        if (!open) setShowAlertDialog(false);
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{alertContent?.title ?? ""}</DialogTitle>
            <DialogDescription>
              {alertContent?.message ?? ""}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={() => setShowAlertDialog(false)}>
              OK
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={showAboutDialog}
        onOpenChange={(open) => {
          if (!open && !aboutCheckLoading) setShowAboutDialog(false);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{aboutAppTitle}</DialogTitle>
            <DialogDescription asChild>
              <div className="space-y-3 text-sm text-muted-foreground">
                <p>
                  QMLauncher — клиент для подключения к серверам QMServer, управления сборками Minecraft и
                  аккаунтами.
                </p>
                {aboutLoading ? (
                  <p>{t.loading}</p>
                ) : aboutInfo ? (
                  <ul className="list-none space-y-1 text-foreground">
                    <li>
                      <span className="text-muted-foreground">Версия: </span>
                      {aboutInfo.version || "—"}
                    </li>
                    <li>
                      <span className="text-muted-foreground">ОС: </span>
                      {aboutInfo.os || "—"}
                    </li>
                    <li>
                      <span className="text-muted-foreground">Архитектура: </span>
                      {aboutInfo.arch || "—"}
                    </li>
                  </ul>
                ) : null}
              </div>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setShowAboutDialog(false)}
              disabled={aboutCheckLoading}
            >
              Закрыть
            </Button>
            <Button
              onClick={async () => {
                setAboutCheckLoading(true);
                try {
                  const ok = await CheckLauncherUpdateAvailable();
                  if (ok) {
                    setShowAboutDialog(false);
                    setShowUpdateDialog(true);
                  } else {
                    showAlert("Обновления", "У вас установлена последняя версия.");
                  }
                } catch (e) {
                  showAlert("Обновления", String(e));
                } finally {
                  setAboutCheckLoading(false);
                }
              }}
              disabled={aboutCheckLoading}
            >
              {aboutCheckLoading ? t.loading : checkUpdatesLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showUpdateDialog} onOpenChange={(open) => {
        if (!open && !updateApplying) setShowUpdateDialog(false);
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Обновление QMLauncher</DialogTitle>
            <DialogDescription>
              Найдено новое обновление. Необходимо перезапустить лаунчер для установки.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setShowUpdateDialog(false)}
              disabled={updateApplying}
            >
              Отмена
            </Button>
            <Button
              onClick={async () => {
                setUpdateApplying(true);
                try {
                  const err = await ApplyLauncherUpdate();
                  if (err) {
                    showAlert("Ошибка обновления", `Не удалось установить обновление: ${err}`);
                  }
                  setUpdateApplying(false);
                  setShowUpdateDialog(false);
                } catch (e) {
                  showAlert("Ошибка обновления", String(e));
                  setUpdateApplying(false);
                  setShowUpdateDialog(false);
                }
              }}
              disabled={updateApplying}
            >
              {updateApplying ? "Установка…" : "Обновить"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!showSyncSkinDialog} onOpenChange={(open) => !open && setShowSyncSkinDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Синхронизация в QMServer Cloud</DialogTitle>
            <DialogDescription>
              Выберите поставщика скина для аккаунта &quot;{showSyncSkinDialog?.username}&quot;.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label>Поставщик скина</Label>
              <NativeSelect
                value={syncSkinProvider}
                onChange={(e) => {
                  const v = e.target.value;
                  setSyncSkinProvider(v);
                  if (v && PROVIDER_URLS[v]) {
                    setSyncSkinUrl(PROVIDER_URLS[v](showSyncSkinDialog?.username || ""));
                  } else if (v === "custom") {
                    setSyncSkinUrl("");
                  }
                }}
              >
                <NativeSelectOption value="">— Выберите поставщика —</NativeSelectOption>
                {skinProviders.ely_by && <NativeSelectOption value="ely_by">{PROVIDER_LABELS.ely_by}</NativeSelectOption>}
                <NativeSelectOption value="custom">Свой URL</NativeSelectOption>
              </NativeSelect>
            </div>
            <div className="grid gap-2">
              <Label>URL скина</Label>
              <Input
                value={syncSkinUrl}
                onChange={(e) => setSyncSkinUrl(e.target.value)}
                placeholder="https://skinsystem.ely.by/skins/Player.png"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowSyncSkinDialog(null)}>
              Отмена
            </Button>
            <Button
              onClick={async () => {
                if (!showSyncSkinDialog) return;
                try {
                  const result = await SyncLocalAccountToCloud(showSyncSkinDialog.username, syncSkinUrl.trim());
                  showAlert("Синхронизация", result);
                  setShowSyncSkinDialog(null);
                  GetAccounts().then(setAccounts);
                  GetCurrentAccount().then((acc: any) => acc?.name && setCurrentAccount({ name: acc.name, email: acc.email || "" }));
                } catch (err) {
                  showAlert("Ошибка", String(err));
                }
              }}
            >
              Синхронизировать
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={showEditCloudSkinDialog} onOpenChange={(open) => {
        if (!open) {
          setShowEditCloudSkinDialog(false);
          setCloudGameAccounts([]);
          setEditCloudAccountId(null);
        }
      }}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Редактировать скин (QMServer Cloud)</DialogTitle>
            <DialogDescription>
              {cloudGameAccounts.length > 1 ? "Выберите аккаунт и настройте скин" : "Настройте скин для игрового аккаунта"}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col sm:flex-row gap-6 py-4">
            <div className="flex-1 grid gap-4 min-w-0">
              {cloudGameAccounts.length > 1 && (
                <div className="grid gap-2">
                  <Label>Игровой аккаунт</Label>
                  <NativeSelect
                    value={editCloudAccountId ?? ""}
                    onChange={(e) => {
                      const id = parseInt(e.target.value, 10);
                      setEditCloudAccountId(id);
                      const acc = cloudGameAccounts.find((a) => a.id === id);
                      if (acc) {
                        setEditCloudSkinUrl(acc.skinUrl || "");
                        setEditCloudSkinModel((acc.skinModel as "steve" | "alex") || "steve");
                        const detected = detectProviderFromUrl(acc.skinUrl);
                        const opts = ["ely_by"].filter((k) => skinProviders[k] ?? true);
                        setEditCloudSkinProvider(opts.includes(detected) ? detected : "custom");
                      }
                    }}
                  >
                    <NativeSelectOption value="">—</NativeSelectOption>
                    {cloudGameAccounts.map((a) => (
                      <NativeSelectOption key={a.id} value={String(a.id)}>
                        {a.username}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                </div>
              )}
              {editCloudAccountId != null && (
                <>
                  <div className="grid gap-2">
                    <Label>Модель скина</Label>
                    <div className="flex gap-4">
                      <label className={`flex flex-col items-center gap-2 p-3 rounded-lg border-2 cursor-pointer transition-colors ${editCloudSkinModel === "steve" ? "border-primary bg-primary/5" : "border-muted"}`}>
                        <input type="radio" name="editCloudSkinModel" value="steve" checked={editCloudSkinModel === "steve"} onChange={() => setEditCloudSkinModel("steve")} className="sr-only" />
                        <img src="https://mc-heads.net/avatar/MHF_Steve/64" alt="Steve" className="w-16 h-16 object-contain" style={{ imageRendering: "pixelated" }} />
                        <span className="text-sm">Steve</span>
                      </label>
                      <label className={`flex flex-col items-center gap-2 p-3 rounded-lg border-2 cursor-pointer transition-colors ${editCloudSkinModel === "alex" ? "border-primary bg-primary/5" : "border-muted"}`}>
                        <input type="radio" name="editCloudSkinModel" value="alex" checked={editCloudSkinModel === "alex"} onChange={() => setEditCloudSkinModel("alex")} className="sr-only" />
                        <img src="https://mc-heads.net/avatar/MHF_Alex/64" alt="Alex" className="w-16 h-16 object-contain" style={{ imageRendering: "pixelated" }} />
                        <span className="text-sm">Alex</span>
                      </label>
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <Label>Поставщик скина</Label>
                    <NativeSelect
                      value={editCloudSkinProvider}
                      onChange={(e) => {
                        const v = e.target.value;
                        setEditCloudSkinProvider(v);
                        const acc = cloudGameAccounts.find((a) => a.id === editCloudAccountId);
                        if (acc && v && PROVIDER_URLS[v]) {
                          setEditCloudSkinUrl(PROVIDER_URLS[v](acc.username, acc.uuid));
                        } else if (v === "custom" && acc) {
                          setEditCloudSkinUrl(acc.skinUrl || "");
                        }
                      }}
                    >
                      <NativeSelectOption value="">— Выберите поставщика —</NativeSelectOption>
                      {skinProviders.ely_by && <NativeSelectOption value="ely_by">{PROVIDER_LABELS.ely_by}</NativeSelectOption>}
                      <NativeSelectOption value="custom">Свой URL</NativeSelectOption>
                    </NativeSelect>
                  </div>
                  <div className="grid gap-2">
                    <Label>URL скина</Label>
                    <Input
                      value={editCloudSkinUrl}
                      onChange={(e) => setEditCloudSkinUrl(e.target.value)}
                      placeholder="https://..."
                    />
                  </div>
                </>
              )}
            </div>
            {editCloudAccountId != null && (() => {
              const acc = cloudGameAccounts.find((a) => a.id === editCloudAccountId);
              const previewSkinUrl =
                acc != null
                  ? getCloudSkinEditPreviewUrl(acc, editCloudSkinUrl ?? "", editCloudSkinProvider, editCloudSkinModel)
                  : "";
              const skinUrlForPreview = previewSkinUrl && apiBase ? proxiedUrl(previewSkinUrl, apiBase) : previewSkinUrl;
              return (
                <div className="flex flex-col items-center shrink-0">
                  <p className="text-sm text-muted-foreground mb-2">Превью (перетащите для поворота)</p>
                  <Suspense
                    fallback={
                      <div
                        className="rounded-lg border border-border bg-muted/30 flex items-center justify-center text-xs text-muted-foreground"
                        style={{ width: 200, height: 280 }}
                      >
                        Загрузка 3D…
                      </div>
                    }
                  >
                    <SkinPreview3d
                      skinUrl={skinUrlForPreview}
                      skinModel={editCloudSkinModel}
                      width={200}
                      height={280}
                    />
                  </Suspense>
                </div>
              );
            })()}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditCloudSkinDialog(false)}>
              Отмена
            </Button>
            <Button
              disabled={editCloudAccountId == null}
              onClick={async () => {
                if (editCloudAccountId == null) return;
                try {
                  const result = await UpdateCloudGameAccount(editCloudAccountId, editCloudSkinUrl.trim(), editCloudSkinModel);
                  showAlert("Скин", result);
                  setShowEditCloudSkinDialog(false);
                  GetAccounts().then(setAccounts);
                } catch (err) {
                  showAlert("Ошибка", String(err));
                }
              }}
            >
              Сохранить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Сводка по подключённому аккаунту */}
      <Dialog open={showAccountSummaryDialog} onOpenChange={(open) => { setShowAccountSummaryDialog(open); if (!open) setAccountSummaryTarget(null); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Информация об аккаунте</DialogTitle>
          </DialogHeader>
          {accountSummaryTarget && (
            <div className="space-y-3 text-sm">
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Тип:</span>
                <span className="font-medium">
                  {accountSummaryTarget.type === "cloud" ? t.authCloud : t.authMicrosoft}
                </span>
              </div>
              {accountSummaryTarget.username && (
                <div className="flex justify-between gap-4">
                  <span className="text-muted-foreground">Имя:</span>
                  <span className="font-medium">{accountSummaryTarget.username}</span>
                </div>
              )}
              {accountSummaryTarget.email && (
                <div className="flex justify-between gap-4">
                  <span className="text-muted-foreground">Email:</span>
                  <span className="font-medium break-all text-right">{accountSummaryTarget.email}</span>
                </div>
              )}
            </div>
          )}
          <DialogFooter>
            <Button onClick={() => { setShowAccountSummaryDialog(false); setAccountSummaryTarget(null); }}>Закрыть</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Инвентарь игрового аккаунта (cloud_game) — как в QMWeb */}
      <Dialog open={showInventoryDialog} onOpenChange={(open) => { setShowInventoryDialog(open); if (!open) { setInventoryAccount(null); setInventoryData(null); } }}>
        <DialogContent className="max-w-[600px] max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Инвентарь — {inventoryAccount?.username || ""}</DialogTitle>
            <DialogDescription>Содержимое инвентаря с серверов (QXSync; на QMServer нужны модуль Minecraft и интеграция QMWeb)</DialogDescription>
          </DialogHeader>
          {inventoryData === null ? (
            <p className="text-muted-foreground py-4">Загрузка...</p>
          ) : inventoryData.error ? (
            <p className="text-destructive py-4">{inventoryData.error}</p>
          ) : !inventoryData.inventories?.length ? (
            <div className="space-y-2 text-sm text-muted-foreground py-4">
              <p>Нет данных. После выхода <strong className="text-foreground">QXSync</strong> (скоро) инвентарь с сервера будет синхронизироваться так:</p>
              <ol className="list-decimal list-inside space-y-1 pl-1">
                <li>На QMServer включены модуль Minecraft и интеграция QMWeb</li>
                <li>На сервере установлен мод/плагин <strong className="text-foreground">QXSync</strong></li>
                <li>В QMAdmin настроен API-ключ синхронизации инвентаря</li>
                <li>Событие выхода с сервера или таймер — клиент отправляет снимок в QMServer</li>
              </ol>
            </div>
          ) : (
            <div className="flex flex-col gap-6 py-2">
              {inventoryData.inventories.map((inv) => (
                <div key={inv.server_id} className="space-y-3">
                  <p className="text-sm text-primary font-medium">
                    Сервер: {inv.server_name || inv.server_id}
                    {inv.server_version && <span className="ml-2 text-muted-foreground">({inv.server_version})</span>}
                    {inv.timestamp ? <span className="ml-2 text-muted-foreground font-normal">{new Date(inv.timestamp).toLocaleString()}</span> : null}
                  </p>
                  <InventoryGrid main={inv.main} armor={inv.armor} offhand={inv.offhand} gameVersion={inv.server_version} apiBase={apiBase} />
                </div>
              ))}
            </div>
          )}
          <DialogFooter>
            <Button onClick={() => { setShowInventoryDialog(false); setInventoryAccount(null); setInventoryData(null); }}>Закрыть</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Создание нового инстанса */}
      <Dialog open={showCreateInstanceDialog} onOpenChange={(open) => { setShowCreateInstanceDialog(open); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t.createInstance}</DialogTitle>
            <DialogDescription>Укажите параметры нового инстанса Minecraft</DialogDescription>
          </DialogHeader>
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              if (!createInstanceName.trim()) return;
              const mcResolved = createInstanceMcVersions.includes(createInstanceVersion)
                ? createInstanceVersion
                : (createInstanceMcVersions[0] ?? "").trim();
              if (!mcResolved) return;
              const loaderResolved =
                createInstanceLoader === "vanilla"
                  ? ""
                  : (createInstanceLoaderVersions.includes(createInstanceLoaderVersion)
                      ? createInstanceLoaderVersion
                      : (createInstanceLoaderVersions[0] ?? "").trim());
              if (createInstanceLoader !== "vanilla" && !loaderResolved) return;
              setCreateInstanceSubmitting(true);
              try {
                const err = await CreateInstance(
                  createInstanceName.trim(),
                  mcResolved,
                  createInstanceLoader,
                  loaderResolved
                );
                if (err) {
                  showAlert("Ошибка", err);
                } else {
                  setShowCreateInstanceDialog(false);
                  GetInstances()
                    .then((raw) => setInstances(normalizeInstancesList(raw)))
                    .catch(() => setInstances([]));
                  setSelectedInstanceName(createInstanceName.trim());
                  window.location.hash = "#instance";
                }
              } finally {
                setCreateInstanceSubmitting(false);
              }
            }}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label htmlFor="instance-name">Имя инстанса</Label>
              <Input
                id="instance-name"
                value={createInstanceName}
                onChange={(e) => setCreateInstanceName(e.target.value)}
                placeholder="MyInstance"
                required
                minLength={1}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="instance-version">Версия Minecraft</Label>
              <NativeSelect
                id="instance-version"
                value={
                  createInstanceMcLoading
                    ? ""
                    : createInstanceMcVersions.includes(createInstanceVersion)
                      ? createInstanceVersion
                      : (createInstanceMcVersions[0] ?? "")
                }
                onChange={(e) => setCreateInstanceVersion(e.target.value)}
                disabled={createInstanceMcLoading || createInstanceMcVersions.length === 0}
              >
                {createInstanceMcLoading ? (
                  <NativeSelectOption value="">{t.loading}</NativeSelectOption>
                ) : createInstanceMcVersions.length === 0 ? (
                  <NativeSelectOption value="">—</NativeSelectOption>
                ) : (
                  createInstanceMcVersions.map((v) => (
                    <NativeSelectOption key={v} value={v}>
                      {v}
                    </NativeSelectOption>
                  ))
                )}
              </NativeSelect>
              {!createInstanceMcLoading && createInstanceMcVersions.length === 0 ? (
                <p className="text-sm text-destructive">Не удалось загрузить список версий Minecraft</p>
              ) : null}
            </div>
            <div className="space-y-2">
              <Label htmlFor="instance-loader">Загрузчик модов</Label>
              <NativeSelect
                id="instance-loader"
                value={createInstanceLoader}
                onChange={(e) => setCreateInstanceLoader(e.target.value)}
              >
                <NativeSelectOption value="vanilla">Vanilla</NativeSelectOption>
                <NativeSelectOption value="fabric">Fabric</NativeSelectOption>
                <NativeSelectOption value="quilt">Quilt</NativeSelectOption>
                <NativeSelectOption value="forge">Forge</NativeSelectOption>
                <NativeSelectOption value="neoforge">NeoForge</NativeSelectOption>
              </NativeSelect>
            </div>
            {createInstanceLoader !== "vanilla" && (
              <div className="space-y-2">
                <Label htmlFor="instance-loader-version">Версия загрузчика</Label>
                <NativeSelect
                  id="instance-loader-version"
                  value={
                    createInstanceLoaderVersionsLoading || !createInstanceLoaderVersions.length
                      ? ""
                      : createInstanceLoaderVersions.includes(createInstanceLoaderVersion)
                        ? createInstanceLoaderVersion
                        : (createInstanceLoaderVersions[0] ?? "")
                  }
                  onChange={(e) => setCreateInstanceLoaderVersion(e.target.value)}
                  disabled={
                    createInstanceLoaderVersionsLoading ||
                    !createInstanceVersion.trim() ||
                    createInstanceLoaderVersions.length === 0
                  }
                >
                  {createInstanceLoaderVersionsLoading ? (
                    <NativeSelectOption value="">{t.loading}</NativeSelectOption>
                  ) : createInstanceLoaderVersions.length === 0 ? (
                    <NativeSelectOption value="">
                      {createInstanceVersion.trim() ? "Нет доступных версий" : "Сначала выберите Minecraft"}
                    </NativeSelectOption>
                  ) : (
                    createInstanceLoaderVersions.map((v) => (
                      <NativeSelectOption key={v} value={v}>
                        {v}
                      </NativeSelectOption>
                    ))
                  )}
                </NativeSelect>
              </div>
            )}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowCreateInstanceDialog(false)}>
                Отмена
              </Button>
              <Button
                type="submit"
                disabled={
                  createInstanceSubmitting ||
                  !createInstanceName.trim() ||
                  createInstanceMcLoading ||
                  createInstanceMcVersions.length === 0 ||
                  (createInstanceLoader !== "vanilla" &&
                    (createInstanceLoaderVersionsLoading || createInstanceLoaderVersions.length === 0))
                }
              >
                {createInstanceSubmitting ? "Создание..." : "Создать"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <Toaster richColors closeButton position="top-center" />
    </ThemeProvider>
  );
}

/** Простое отображение инвентаря (слоты + иконки предметов) как в QMWeb */
function InventoryGrid({
  main,
  armor,
  offhand,
  gameVersion = "1.21.1",
  apiBase,
}: {
  main: Array<{ slot: number; item: string; count: number }>;
  armor: Array<{ slot: number; item: string; count: number }>;
  offhand?: { slot: number; item: string; count: number };
  gameVersion?: string;
  apiBase?: string;
}) {
  const version = gameVersion?.match(/^[\d.]+/)?.[0] || "1.21.1";
  const baseUrl = `https://assets.mcasset.cloud/${version}/assets/minecraft/textures`;
  const getSlotAt = (arr: Array<{ slot: number; item: string; count: number }>, idx: number) =>
    arr?.find((s) => s.slot === idx);
  const slotPx = 36;
  const itemToTexture = (item: string) => {
    const id = (item || "").replace(/^minecraft:/i, "").trim() || "air";
    if (id === "air") return "";
    const name = id.replace(/_/g, "_");
    const isBlock = /_block|_ore|_log|_planks|_glass|_wool|_bricks/i.test(name);
    const dir = isBlock ? "block" : "item";
    const url = `${baseUrl}/${dir}/${name}.png`;
    return apiBase ? `${apiBase}/skins/proxy?url=${encodeURIComponent(url)}` : url;
  };
  const Slot = ({ s }: { s?: { slot: number; item: string; count: number } }) => (
    <div
      className="relative flex items-center justify-center rounded border-2 border-amber-900/80 bg-[#8b8b8b]/90 overflow-hidden"
      style={{ width: slotPx, height: slotPx, minWidth: slotPx, minHeight: slotPx }}
    >
      {s?.item ? (
        <>
          <img
            src={itemToTexture(s.item)}
            alt=""
            className="w-[85%] h-[85%] object-contain"
            style={{ imageRendering: "pixelated" }}
          />
          {s.count > 1 && (
            <span className="absolute bottom-0 right-1 text-[10px] font-bold text-white leading-none" style={{ textShadow: "1px 1px 0 #333" }}>
              {s.count}
            </span>
          )}
        </>
      ) : null}
    </div>
  );
  const armorOrder = [100, 101, 102, 103, 39, 38, 37, 36]; // helmet, chest, legs, boots
  const armorSlots = armorOrder.map((i) => getSlotAt(armor || [], i)).filter(Boolean);
  return (
    <div className="inline-block rounded-lg p-3 bg-[#2d2d2d] border border-[#555]">
      <div className="flex gap-2 mb-2">
        {armorSlots.slice(0, 4).map((a, i) => (
          <Slot key={i} s={a} />
        ))}
        {offhand && <Slot s={offhand} />}
      </div>
      <div className="grid grid-cols-9 gap-0.5">
        {[0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35].map((i) => (
          <div key={i} className="relative">
            <Slot s={getSlotAt(main || [], i)} />
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;
