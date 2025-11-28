// Utility for reading server data from API (replaced SQLite)

export interface EmbeddedServer {
  server_id: number
  server_uuid: string
  server_name: string | null
  server_address: string | null
  server_port: number | null
  minecraft_version: string | null
  description: string | null
  preview_image_url: string | null
  enabled: number
}

export async function getEmbeddedServers(): Promise<EmbeddedServer[]> {
  try {
    if (!window.electronAPI || !window.electronAPI.getEmbeddedServers) {
      return []
    }
    
    // Get servers from API
    const servers = await window.electronAPI.getEmbeddedServers()
    return servers || []
  } catch (error) {
    console.error('Error reading embedded servers:', error)
    return []
  }
}

