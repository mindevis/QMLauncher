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

