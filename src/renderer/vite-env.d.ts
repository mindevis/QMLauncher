/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __QM_LAUNCHER_VERSION__: string | undefined

interface Window {
  electronAPI: {
    getAppVersion: () => Promise<string>
    getPlatform: () => Promise<string>
    getHwid: () => Promise<string>
    getLauncherConfig: () => Promise<any>
    saveLauncherConfig: (config: any) => Promise<any>
    launchMinecraft: (launchArgs: any) => Promise<any>
    stopMinecraft: () => Promise<any>
    getLauncherDbConfig: (serverId: number) => Promise<any>
    getServerMods: (serverId: number, apiBaseUrl: string) => Promise<any>
    downloadMod: (downloadUrl: string, savePath: string) => Promise<any>
    checkAndUpdateMods: (serverId: number, apiBaseUrl?: string) => Promise<any>
    onMinecraftExited: (callback: (code: number | null) => void) => void
    onMinecraftError: (callback: (error: string) => void) => void
    // Window controls
    windowMinimize: () => Promise<void>
    windowMaximize: () => Promise<void>
    windowClose: () => Promise<void>
    windowIsMaximized: () => Promise<boolean>
    // Theme management
    getCustomThemes?: () => Promise<any[]>
    saveCustomTheme?: (theme: any) => Promise<any>
    removeCustomTheme?: (themeId: string) => Promise<any>
    // Embedded servers
    getEmbeddedServers?: () => Promise<any[]>
    // Auth token management
    saveAuthToken?: (token: string) => Promise<any>
    getAuthToken?: () => Promise<string | null>
    clearAuthToken?: () => Promise<any>
    // Client installation check
    checkClientInstalled?: (serverId: number) => Promise<{ success: boolean; installed: boolean; hasClient?: boolean; hasMods?: boolean; hasModsConfig?: boolean; minecraftVersion?: string | null; modsDir?: string; error?: string }>
    // Minecraft client installation
    installMinecraftClient?: (version: string) => Promise<{ success: boolean; alreadyInstalled?: boolean; message?: string; clientJar?: string; librariesDownloaded?: number; error?: string }>
    // Installation progress listener
    onInstallationProgress?: (callback: (progress: { stage: string; progress: number }) => void) => void
  }
}

