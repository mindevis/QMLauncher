import fs from 'fs'
import path from 'path'
import os from 'os'
import { encrypt, decrypt } from './encryption'
import { API_BASE_URL } from '../config/api'

const CONFIG_DIR = path.join(os.homedir(), '.qmlauncher')
const CONFIG_FILE = path.join(CONFIG_DIR, 'config.json')
const CONFIG_VERSION = 1

export interface ServerData {
  id: number
  name: string
  server_name?: string
  server_address?: string
  server_port: number
  minecraft_version: string
  description?: string
  preview_image_url?: string
  server_uuid?: string
  loader_enabled: boolean
  loader_type: string
  loader_version?: string
}

export interface ModData {
  id: number
  name: string
  version?: string
  filename: string
  size: number
  server_id: number
}

export interface LauncherConfigData {
  version: number
  lastSync: string | null
  servers: ServerData[]
  mods: ModData[]
  settings: {
    javaPath: string
    minMemory: number
    maxMemory: number
    windowWidth: number
    windowHeight: number
    themeId: string
    selectedProfile: string | null
    profiles: Record<string, any>
    apiBaseUrl: string
  }
}

/**
 * Get default config structure
 */
function getDefaultConfig(): LauncherConfigData {
  return {
    version: CONFIG_VERSION,
    lastSync: null,
    servers: [],
    mods: [],
    settings: {
      javaPath: 'java',
      minMemory: 1024,
      maxMemory: 4096,
      windowWidth: 1200,
      windowHeight: 800,
      themeId: 'dark',
      selectedProfile: null,
      profiles: {},
      apiBaseUrl: API_BASE_URL
    }
  }
}

/**
 * Load encrypted config from disk
 */
export function loadConfig(): LauncherConfigData {
  try {
    if (!fs.existsSync(CONFIG_FILE)) {
      return getDefaultConfig()
    }

    const encryptedData = fs.readFileSync(CONFIG_FILE, 'utf-8')
    const decryptedData = decrypt(encryptedData)
    const config = JSON.parse(decryptedData) as LauncherConfigData

    // Migrate old configs if needed
    if (config.version !== CONFIG_VERSION) {
      // Handle version migration if needed
      config.version = CONFIG_VERSION
    }

    return config
  } catch (error) {
    console.error('Error loading config:', error)
    // If decryption fails, return default config
    return getDefaultConfig()
  }
}

/**
 * Save encrypted config to disk
 */
export function saveConfig(config: LauncherConfigData): boolean {
  try {
    if (!fs.existsSync(CONFIG_DIR)) {
      fs.mkdirSync(CONFIG_DIR, { recursive: true })
    }

    const jsonData = JSON.stringify(config, null, 2)
    const encryptedData = encrypt(jsonData)
    fs.writeFileSync(CONFIG_FILE, encryptedData, 'utf-8')
    return true
  } catch (error) {
    console.error('Error saving config:', error)
    return false
  }
}

/**
 * Fetch all data from QMServer
 */
export async function fetchDataFromServer(): Promise<{
  servers: ServerData[]
  mods: ModData[]
}> {
  try {
    // Fetch servers
    const serversResponse = await fetch(`${API_BASE_URL}/servers`)
    if (!serversResponse.ok) {
      throw new Error(`Failed to fetch servers: ${serversResponse.statusText}`)
    }
    const serversData = await serversResponse.json() as { servers?: any[] }
    const servers: ServerData[] = (serversData.servers || []).map((server: any) => ({
      id: server.id,
      name: server.name,
      server_name: server.server_name,
      server_address: server.server_address,
      server_port: server.server_port,
      minecraft_version: server.minecraft_version,
      description: server.description,
      preview_image_url: server.preview_image_url,
      server_uuid: server.server_uuid,
      loader_enabled: server.loader_enabled || false,
      loader_type: server.loader_type || 'vanilla',
      loader_version: server.loader_version
    }))

    // Fetch mods for all servers
    const mods: ModData[] = []
    for (const server of servers) {
      try {
        const modsResponse = await fetch(`${API_BASE_URL}/servers/${server.id}/mods`)
        if (modsResponse.ok) {
          const modsData = await modsResponse.json() as { mods?: any[] }
          const serverMods = (modsData.mods || []).map((mod: any) => ({
            id: mod.id || 0,
            name: mod.name,
            version: mod.version,
            filename: mod.filename,
            size: mod.size || 0,
            server_id: server.id
          }))
          mods.push(...serverMods)
        }
      } catch (error) {
        console.error(`Error fetching mods for server ${server.id}:`, error)
      }
    }

    return { servers, mods }
  } catch (error) {
    console.error('Error fetching data from server:', error)
    throw error
  }
}

/**
 * Check if config needs update by comparing with server data
 */
export async function checkConfigNeedsUpdate(): Promise<boolean> {
  try {
    const config = loadConfig()
    
    // If never synced, needs update
    if (!config.lastSync) {
      return true
    }

    // Fetch current data from server
    const serverData = await fetchDataFromServer()

    // Compare servers count and IDs
    const configServerIds = new Set(config.servers.map(s => s.id))
    const serverServerIds = new Set(serverData.servers.map(s => s.id))
    
    if (configServerIds.size !== serverServerIds.size) {
      return true
    }

    for (const serverId of configServerIds) {
      if (!serverServerIds.has(serverId)) {
        return true
      }
    }

    // Compare mods (simplified - check if any server has different mod count)
    const configModsByServer = new Map<number, number>()
    config.mods.forEach(mod => {
      configModsByServer.set(mod.server_id, (configModsByServer.get(mod.server_id) || 0) + 1)
    })

    const serverModsByServer = new Map<number, number>()
    serverData.mods.forEach(mod => {
      serverModsByServer.set(mod.server_id, (serverModsByServer.get(mod.server_id) || 0) + 1)
    })

    for (const [serverId, count] of configModsByServer) {
      if (serverModsByServer.get(serverId) !== count) {
        return true
      }
    }

    return false
  } catch (error) {
    console.error('Error checking config update:', error)
    // On error, assume update is needed
    return true
  }
}

/**
 * Sync config with QMServer
 */
export async function syncConfigWithServer(): Promise<boolean> {
  try {
    const config = loadConfig()
    
    // Fetch fresh data from server
    const serverData = await fetchDataFromServer()

    // Update config with server data
    config.servers = serverData.servers
    config.mods = serverData.mods
    config.lastSync = new Date().toISOString()

    // Save updated config
    return saveConfig(config)
  } catch (error) {
    console.error('Error syncing config with server:', error)
    return false
  }
}

/**
 * Update settings in config
 */
export function updateSettings(settings: Partial<LauncherConfigData['settings']>): boolean {
  const config = loadConfig()
  config.settings = { ...config.settings, ...settings }
  return saveConfig(config)
}

/**
 * Get settings from config
 */
export function getSettings(): LauncherConfigData['settings'] {
  const config = loadConfig()
  return config.settings
}

/**
 * Get servers from config
 */
export function getServers(): ServerData[] {
  const config = loadConfig()
  return config.servers
}

/**
 * Get mods from config
 */
export function getMods(serverId?: number): ModData[] {
  const config = loadConfig()
  if (serverId) {
    return config.mods.filter(mod => mod.server_id === serverId)
  }
  return config.mods
}

