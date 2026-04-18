import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface ModMetadata {
  optional?: boolean;
  launcher_disabled?: boolean;
  display_name?: string;
  icon_url?: string;
  description?: string;
  curseforge_url?: string;
  modrinth_url?: string;
}

interface ModConfig {
  mod_paths?: string[];
  mods_metadata?: Record<string, ModMetadata>;
}

interface ModInfo {
  path: string;
  name: string;
  meta?: ModMetadata;
}

interface ServerModListProps {
  serverID: number;
  apiBase: string;
  onModClick?: (mod: ModInfo) => void;
}

export function ServerModList({ serverID, apiBase, onModClick }: ServerModListProps) {
  const [config, setConfig] = useState<ModConfig | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (serverID <= 0 || !apiBase) {
      setConfig(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    const url = `${apiBase.replace(/\/$/, "")}/servers/${serverID}/mod-config`;
    fetch(url)
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error("Failed"))))
      .then((data: ModConfig) => setConfig(data))
      .catch(() => setConfig({ mod_paths: [], mods_metadata: {} }))
      .finally(() => setLoading(false));
  }, [serverID, apiBase]);

  const mods: ModInfo[] = (config?.mod_paths ?? []).map((path) => {
    const meta = config?.mods_metadata?.[path];
    const name = meta?.display_name || path.split("/").pop() || path;
    return { path, name, meta };
  });

  if (loading) {
    return <p className="text-sm text-muted-foreground">Загрузка списка модов...</p>;
  }
  if (mods.length === 0) {
    return <p className="text-sm text-muted-foreground">Моды не найдены</p>;
  }

  return (
    <div className="space-y-1">
      <p className="text-xs font-medium text-muted-foreground mb-2">Моды сервера ({mods.length})</p>
      <div className="max-h-40 overflow-y-auto scrollbar-hide space-y-1">
        {mods.map((mod) => (
          <button
            key={mod.path}
            type="button"
            className="flex items-center gap-2 w-full text-left px-2 py-1.5 rounded hover:bg-muted text-sm"
            onClick={() => onModClick?.(mod)}
          >
            {mod.meta?.icon_url ? (
              <img src={mod.meta.icon_url} alt="" className="size-5 shrink-0 rounded object-cover" />
            ) : (
              <div className="size-5 shrink-0 rounded bg-muted" />
            )}
            <span className="truncate">{mod.name}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

interface ModDetailDialogProps {
  mod: ModInfo | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ModDetailDialog({ mod, open, onOpenChange }: ModDetailDialogProps) {
  if (!mod) return null;
  const { name, meta } = mod;
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {meta?.icon_url ? (
              <img src={meta.icon_url} alt="" className="size-8 rounded object-cover" />
            ) : (
              <div className="size-8 rounded bg-muted" />
            )}
            {name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-3 text-sm">
          {meta?.description && (
            <p className="text-muted-foreground whitespace-pre-wrap">{meta.description}</p>
          )}
          <div className="flex flex-wrap gap-2">
            {meta?.curseforge_url && (
              <a
                href={meta.curseforge_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                CurseForge
              </a>
            )}
            {meta?.modrinth_url && (
              <a
                href={meta.modrinth_url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                Modrinth
              </a>
            )}
          </div>
          {!meta?.description && !meta?.curseforge_url && !meta?.modrinth_url && (
            <p className="text-muted-foreground">Нет дополнительной информации</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
