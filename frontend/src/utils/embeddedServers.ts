// Utility for reading server data from API (replaced SQLite)

import { wailsAPI } from '../bridge'

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
    const servers = await wailsAPI.getEmbeddedServers()
    return servers || []
  } catch (error) {
    console.error('Error reading embedded servers:', error)
    return []
  }
}

