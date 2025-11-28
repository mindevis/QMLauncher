export interface Server {
  id: number
  name: string
  description: string
  server_name?: string
  server_address?: string
  server_port: number
  preview_image_url?: string
  minecraft_version: string
  loader_enabled: boolean
  loader_type: string
  loader_version?: string
  server_uuid?: string | null
  server_status?: "online" | "offline" | "unknown"
}

export interface LauncherConfig {
  serverId?: number
  serverName?: string
  serverAddress?: string
  serverPort?: number
  previewImageUrl?: string
  minecraftVersion?: string
  loaderEnabled?: boolean
  loaderType?: string
  loaderVersion?: string
  apiBaseUrl?: string
}

export interface Profile {
  id: string
  name: string
  serverId?: number
  minecraftVersion: string
  loaderEnabled: boolean
  loaderType: string
  loaderVersion?: string
  username: string
  memory: number
  javaPath?: string
  jvmArgs?: string[]
  gameArgs?: string[]
}

export interface LauncherSettings {
  javaPath: string
  minMemory: number
  maxMemory: number
  windowWidth: number
  windowHeight: number
  selectedProfile: string | null
  profiles: Record<string, Profile>
  apiBaseUrl?: string
}

export interface MinecraftVersion {
  id: string
  type: string
  releaseTime: string
}

declare global {
  interface Window {
    electronAPI: {
      getAppVersion: () => Promise<string>
      getPlatform: () => Promise<string>
      getHwid: () => Promise<string | null>
      getLauncherConfig: () => Promise<LauncherSettings>
      saveLauncherConfig: (config: LauncherSettings) => Promise<{ success: boolean; error?: string }>
      launchMinecraft: (launchArgs: {
        javaPath: string
        gameArgs: string[]
        jvmArgs: string[]
        workingDirectory: string
        minecraftVersion?: string
        hwid?: string | null
        launcherConfig?: any
      }) => Promise<{ success: boolean; error?: string; pid?: number }>
      stopMinecraft: () => Promise<{ success: boolean; error?: string }>
      getLauncherDbConfig: (serverId: number) => Promise<{ success: boolean; config?: Record<string, string>; mods?: any[]; error?: string }>
      getServerMods: (serverId: number, apiBaseUrl: string) => Promise<{ success: boolean; mods?: any[]; error?: string }>
      downloadMod: (downloadUrl: string, savePath: string) => Promise<{ success: boolean; error?: string }>
      checkAndUpdateMods: (serverId: number, apiBaseUrl?: string) => Promise<{ success: boolean; updated?: boolean; modsUpdated?: number; modsDir?: string; error?: string }>
      onMinecraftExited: (callback: (code: number | null) => void) => void
      onMinecraftError: (callback: (error: string) => void) => void
      // Config synchronization
      syncConfigWithServer: () => Promise<{ success: boolean; error?: string }>
      checkConfigNeedsUpdate: () => Promise<{ needsUpdate: boolean }>
      getConfigMods: (serverId?: number) => Promise<{ success: boolean; mods?: any[]; error?: string }>
    }
  }
}

