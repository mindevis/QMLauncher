// Импортируем функции из Wails
import * as WailsApp from '../wailsjs/go/main/App'
import * as WailsConfig from '../wailsjs/go/main/ConfigService'
import * as WailsMinecraft from '../wailsjs/go/main/MinecraftService'
import * as WailsJava from '../wailsjs/go/main/JavaService'

// Типы для API
export interface Server {
  id: number
  name: string
  description?: string
  version: string
  mods?: Mod[]
}

export interface Mod {
  id: number
  name: string
  version: string
  filename: string
  size: number
}

// Settings type matching Go struct
export interface Settings {
  apiBaseUrl: string
  minecraftPath: string
  javaPath: string
  minMemory: number
  maxMemory: number
  jvmArgs: string[]
}

// Wails методы будут доступны через window.go после генерации
// Для разработки используем заглушки
declare global {
  interface Window {
    go?: {
      main: {
        App: {
          GetAppVersion: () => Promise<string>
          GetPlatform: () => Promise<string>
          APIRequest: (url: string, method: string, headers?: Record<string, string>, body?: string) => Promise<any>
          WindowMinimize: () => Promise<void>
          WindowMaximize: () => Promise<void>
          WindowClose: () => Promise<void>
          WindowIsMaximized: () => Promise<boolean>
          GetSettings: () => Promise<Settings>
          SaveSettings: (settings: Settings) => Promise<void>
          InstallJava: (vendor: string, version: string, serverUuid?: string) => Promise<void>
          GetJavaPath: (serverUuid?: string) => Promise<string>
        }
        MinecraftService: {
          LaunchMinecraft: (args: any) => Promise<any>
          StopMinecraft: () => Promise<void>
          InstallMinecraftClient: (version: string, javaVendor?: string, javaVersion?: string, serverUuid?: string) => Promise<any>
          CheckClientInstalled: (serverId: number, serverUuid?: string) => Promise<any>
        }
      }
    }
  }
}

// Wails API обертка - использует сгенерированные Wails функции
export const wailsAPI = {
  // App methods
  getAppVersion: () => WailsApp.GetAppVersion(),
  getPlatform: () => WailsApp.GetPlatform(),
  apiRequest: (url: string, method: string, headers?: Record<string, string>, body?: string) => 
    WailsApp.APIRequest(url, method, headers || {}, body || ''),
  
  // Window methods
  windowMinimize: () => WailsApp.WindowMinimize(),
  windowMaximize: () => WailsApp.WindowMaximize(),
  windowClose: () => WailsApp.WindowClose(),
  windowIsMaximized: () => WailsApp.WindowIsMaximized(),
  
  // Settings
  getSettings: () => WailsConfig.GetSettings(),
  saveSettings: (settings: Settings) => WailsConfig.SaveSettings(settings as any), // Type mismatch - Wails Settings may have different structure
  syncConfigWithServer: () => WailsConfig.SyncConfigWithServer(),
  
  // Java
  installJava: (vendor: string, version: string, serverUuid?: string) => 
    WailsApp.InstallJava(vendor, version, serverUuid || ''),
  getJavaPath: (serverUuid?: string) => 
    WailsApp.GetJavaPath(serverUuid || ''),
  
  // Minecraft
  launchMinecraft: (args: any) => WailsMinecraft.LaunchMinecraft(args),
  stopMinecraft: () => WailsMinecraft.StopMinecraft(),
  isMinecraftRunning: () => (WailsMinecraft as any).IsMinecraftRunning(),
  installMinecraftClient: (version: string, javaVendor?: string, javaVersion?: string, serverUuid?: string) => 
    WailsMinecraft.InstallMinecraftClient(version, javaVendor || '', javaVersion || '', serverUuid || ''),
  checkClientInstalled: (serverId: number, serverUuid?: string) => 
    WailsMinecraft.CheckClientInstalled(serverId, serverUuid || ''),
  
  // Java validation
  validateJavaPath: (javaPath: string) => WailsJava.ValidateJavaPath(javaPath),
  
  // HWID - доступно через динамический импорт
  getHwid: () => (WailsApp as any).GetHWID(),
  
  // Screen resolutions - доступно через динамический импорт
  getScreenResolutions: () => (WailsApp as any).GetScreenResolutions(),
  
  // Embedded servers - доступно через динамический импорт
  getEmbeddedServers: () => (WailsApp as any).GetEmbeddedServers(),
  
  // Launcher config - доступно через динамический импорт
  getLauncherDbConfig: (serverId: number) => (WailsApp as any).GetLauncherDbConfig(serverId),
  
  // Mods - доступно через динамический импорт
  getServerMods: (serverId: number, apiBaseUrl: string) => (WailsApp as any).GetServerMods(serverId, apiBaseUrl),
  downloadMod: (downloadUrl: string, savePath: string) => (WailsApp as any).DownloadMod(downloadUrl, savePath),
  checkAndUpdateMods: (serverId: number, apiBaseUrl: string) => (WailsApp as any).CheckAndUpdateMods(serverId, apiBaseUrl),
  
  // Uninstall - доступно через динамический импорт
  uninstallMinecraft: (serverId: number) => (WailsApp as any).UninstallMinecraft(serverId),
  
  // Check if .qmlauncher directory exists (for first launch detection)
  checkQMLauncherDirExists: () => WailsApp.CheckQMLauncherDirExists(),
  
  // Check if embedded config exists (Mode 3 - built by QMServer)
  hasEmbeddedConfig: () => WailsApp.HasEmbeddedConfig(),
}

