import { contextBridge, ipcRenderer } from 'electron'

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld('electronAPI', {
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),
  getPlatform: () => ipcRenderer.invoke('get-platform'),
  getHwid: () => ipcRenderer.invoke('get-hwid'),
  getLauncherConfig: () => ipcRenderer.invoke('get-launcher-config'),
  saveLauncherConfig: (config: any) => ipcRenderer.invoke('save-launcher-config', config),
  launchMinecraft: (launchArgs: any) => ipcRenderer.invoke('launch-minecraft', launchArgs),
  stopMinecraft: () => ipcRenderer.invoke('stop-minecraft'),
  getLauncherDbConfig: (serverId: number) => ipcRenderer.invoke('get-launcher-db-config', serverId),
  getServerMods: (serverId: number, apiBaseUrl: string) => ipcRenderer.invoke('get-server-mods', serverId, apiBaseUrl),
  downloadMod: (downloadUrl: string, savePath: string) => ipcRenderer.invoke('download-mod', downloadUrl, savePath),
  checkAndUpdateMods: (serverId: number, apiBaseUrl?: string) => ipcRenderer.invoke('check-and-update-mods', serverId, apiBaseUrl),
  onMinecraftExited: (callback: (code: number | null) => void) => {
    ipcRenderer.on('minecraft-exited', (_event, code) => callback(code))
  },
  onMinecraftError: (callback: (error: string) => void) => {
    ipcRenderer.on('minecraft-error', (_event, error) => callback(error))
  },
  // Window controls
  windowMinimize: () => ipcRenderer.invoke('window-minimize'),
  windowMaximize: () => ipcRenderer.invoke('window-maximize'),
  windowClose: () => ipcRenderer.invoke('window-close'),
  windowIsMaximized: () => ipcRenderer.invoke('window-is-maximized'),
  // Theme management
  getCustomThemes: () => ipcRenderer.invoke('get-custom-themes'),
  saveCustomTheme: (theme: any) => ipcRenderer.invoke('save-custom-theme', theme),
  removeCustomTheme: (themeId: string) => ipcRenderer.invoke('remove-custom-theme', themeId),
  // Embedded servers
  getEmbeddedServers: () => ipcRenderer.invoke('get-embedded-servers'),
  // Auth token management
  saveAuthToken: (token: string) => ipcRenderer.invoke('save-auth-token', token),
  getAuthToken: () => ipcRenderer.invoke('get-auth-token'),
  clearAuthToken: () => ipcRenderer.invoke('clear-auth-token'),
  // Client installation check
  checkClientInstalled: (serverId: number) => ipcRenderer.invoke('check-client-installed', serverId),
  // Minecraft client installation
  installMinecraftClient: (version: string) => ipcRenderer.invoke('install-minecraft-client', version),
  // Installation progress listener
  onInstallationProgress: (callback: (progress: { stage: string; progress: number }) => void) => {
    ipcRenderer.on('installation-progress', (_event, progress) => callback(progress))
  },
})

