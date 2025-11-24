import { useState, useEffect } from 'react'
import { Server as ServerType } from '../../shared/types'
import { Play, X, RefreshCw, Loader2 } from 'lucide-react'
import { cn } from '../lib/utils'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000/api/v1'

export function ServersTab() {
  const [servers, setServers] = useState<ServerType[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isLaunching, setIsLaunching] = useState(false)

  useEffect(() => {
    loadServers()
  }, [])

  const loadServers = async () => {
    try {
      setIsLoading(true)
      // Get HWID from Electron
      let hwid: string | null = null
      if (window.electronAPI) {
        hwid = await window.electronAPI.getHwid()
      }
      
      const headers: HeadersInit = {
        'Content-Type': 'application/json',
      }
      
      if (hwid) {
        headers['X-HWID'] = hwid
      }
      
      const response = await fetch(`${API_BASE_URL}/servers`, {
        headers,
      })
      
      if (response.ok) {
        const data = await response.json()
        const serversList: ServerType[] = data.servers || []
        
        // Check status for servers with address and port
        const serversWithStatus = await Promise.all(
          serversList.map(async (server) => {
            if (server.server_address && server.server_port) {
              try {
                const statusResponse = await fetch(`${API_BASE_URL}/minecraft-server/status`, {
                  method: "POST",
                  headers: {
                    "Content-Type": "application/json",
                  },
                  body: JSON.stringify({
                    host: server.server_address,
                    port: server.server_port,
                    method: "auto",
                  }),
                })
                if (statusResponse.ok) {
                  const statusData = await statusResponse.json()
                  return { ...server, server_status: (statusData.online ? "online" : "offline") as "online" | "offline" }
                }
              } catch (error) {
                console.error(`Error checking status for server ${server.id}:`, error)
              }
            }
            return { ...server, server_status: "unknown" as const }
          })
        )
        
        setServers(serversWithStatus)
      }
    } catch (error) {
      console.error('Error loading servers:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleLaunch = async (server: ServerType) => {
    if (!window.electronAPI) {
      alert('Electron API не доступен')
      return
    }

    try {
      setIsLaunching(true)
      const hwid = await window.electronAPI.getHwid()
      
      // Validate server_uuid if available
      if (server.server_uuid) {
        console.log(`Launching server ${server.name} with UUID: ${server.server_uuid}`)
        console.log(`HWID: ${hwid}`)
        // TODO: Send validation request to QMServer with server_uuid and hwid
      }

      // Check and update mods before launch
      console.log('Checking and updating mods...')
      const modsCheckResult = await window.electronAPI.checkAndUpdateMods(server.id, API_BASE_URL)
      
      if (!modsCheckResult.success) {
        alert(`Ошибка при проверке модов: ${modsCheckResult.error}`)
        return
      }

      if (modsCheckResult.updated) {
        console.log(`Updated ${modsCheckResult.modsUpdated || 0} mod(s)`)
        // Show notification that mods were updated
      }

      // Get launcher config
      const launcherConfig = await window.electronAPI.getLauncherConfig()
      
      // Get server config from database
      const dbConfig = await window.electronAPI.getLauncherDbConfig(server.id)
      
      if (!dbConfig.success || !dbConfig.config) {
        alert('Не удалось загрузить конфигурацию сервера')
        return
      }

      // Prepare mods directory path
      const modsDir = modsCheckResult.modsDir || `~/.qmlauncher/mods/${server.id}`
      
      // TODO: Prepare JVM and game arguments from server config
      // For now, use basic arguments
      const jvmArgs = [
        `-Xmx${launcherConfig.maxMemory || 4096}M`,
        `-Xms${launcherConfig.minMemory || 1024}M`,
        '-Djava.library.path=natives',
        '-Dminecraft.launcher.brand=qmlauncher',
        '-Dminecraft.launcher.version=1.0.0'
      ]

      const gameArgs = [
        '--username', 'Player',
        '--version', server.minecraft_version,
        '--gameDir', modsDir,
        '--assetsDir', '~/.qmlauncher/assets',
        '--assetIndex', server.minecraft_version,
        '--uuid', hwid || '00000000-0000-0000-0000-000000000000',
        '--accessToken', 'token',
        '--userType', 'mojang',
        '--versionType', 'release',
        '--server', server.server_address || '',
        '--port', String(server.server_port || 25565)
      ]

      // Launch Minecraft
      const launchResult = await window.electronAPI.launchMinecraft({
        javaPath: launcherConfig.javaPath || 'java',
        jvmArgs,
        gameArgs,
        workingDirectory: modsDir
      })

      if (!launchResult.success) {
        alert(`Ошибка при запуске Minecraft: ${launchResult.error}`)
        return
      }

      console.log(`Minecraft launched with PID: ${launchResult.pid}`)
    } catch (error) {
      console.error('Error launching:', error)
      alert(`Ошибка при запуске: ${error instanceof Error ? error.message : String(error)}`)
    } finally {
      setIsLaunching(false)
    }
  }

  const getServerImage = (server: ServerType) => {
    if (server.preview_image_url) {
      return server.preview_image_url
    }
    // Default Minecraft-themed placeholder
    return '/minecraft-server-preview.svg'
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <Loader2 className="w-12 h-12 animate-spin text-blue-500 mx-auto mb-4" />
          <p className="text-gray-400">Загрузка серверов...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Filter/Header Bar */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4 text-sm text-gray-400">
          <span>Сервер</span>
          <span>Над</span>
          <span>Нашии</span>
        </div>
        
        <div className="flex items-center gap-2">
          <button className="w-8 h-8 rounded-full bg-gray-700 hover:bg-gray-600 flex items-center justify-center text-white text-sm">
            1
          </button>
          <button className="w-8 h-8 rounded-full bg-gray-700 hover:bg-gray-600 flex items-center justify-center text-white">
            @
          </button>
          <button className="w-8 h-8 rounded-full bg-gray-700 hover:bg-gray-600 flex items-center justify-center text-gray-400">
            <RefreshCw className="w-4 h-4" />
          </button>
          <button className="w-8 h-8 rounded-full bg-gray-700 hover:bg-gray-600 flex items-center justify-center text-gray-400">
            <X className="w-4 h-4" />
          </button>
          <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold">
            Суптирить
          </button>
        </div>
      </div>

      {/* Servers Grid - Minecraft Style */}
      {servers.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-xl text-gray-400 mb-2">Нет доступных серверов</p>
          <p className="text-gray-500">Создайте сервер в QMAdmin для отображения здесь</p>
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-4">
          {servers.map((server, index) => (
            <div
              key={server.id}
              className={cn(
                "relative bg-gray-800/90 border-2 border-gray-700/50 rounded-lg p-4 hover:border-gray-600 transition-all cursor-pointer group",
                "minecraft-block-style"
              )}
            >
              {/* Close button */}
              <button
                className="absolute top-2 right-2 w-6 h-6 rounded-full bg-gray-700/50 hover:bg-gray-600 flex items-center justify-center text-gray-400 hover:text-white opacity-0 group-hover:opacity-100 transition-opacity"
                onClick={(e) => {
                  e.stopPropagation()
                  // TODO: Remove server from list
                }}
              >
                <X className="w-3 h-3" />
              </button>

              {/* Server Image */}
              <div className="w-full h-32 mb-3 rounded-lg overflow-hidden bg-gradient-to-br from-gray-700 to-gray-800">
                <img
                  src={getServerImage(server)}
                  alt={server.name}
                  className="w-full h-full object-cover"
                  onError={(e) => {
                    e.currentTarget.src = '/minecraft-server-preview.svg'
                  }}
                />
              </div>

              {/* Server Info */}
              <div className="space-y-2">
                <h3 className="text-lg font-bold text-white">
                  {server.server_name || server.name}
                </h3>
                
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {server.server_status === "online" && (
                      <span className="px-2 py-1 bg-green-500 text-white text-xs rounded font-semibold">
                        Online
                      </span>
                    )}
                    {server.server_status === "offline" && (
                      <span className="px-2 py-1 bg-red-500 text-white text-xs rounded font-semibold">
                        Offline
                      </span>
                    )}
                  </div>
                  
                  <span className="text-gray-400 text-xs font-mono">
                    {server.server_address ? `${server.server_address}:${server.server_port}` : 'N/A'}
                  </span>
                </div>

                {/* Player count or version */}
                <div className="flex items-center justify-between text-sm">
                  <span className="text-gray-300">MC {server.minecraft_version}</span>
                  <span className="text-gray-500 font-mono">
                    {2020 + (index % 10) * 10}
                  </span>
                </div>

                {/* Action Buttons */}
                <div className="flex gap-2 mt-3">
                  <button
                    className="flex-1 bg-green-600 hover:bg-green-700 text-white py-2 px-3 rounded font-semibold text-sm flex items-center justify-center gap-2"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleLaunch(server)
                    }}
                    disabled={isLaunching || server.server_status === "offline"}
                  >
                    {isLaunching ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <Play className="w-4 h-4" />
                    )}
                    Плии
                  </button>
                  <button
                    className="px-3 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded font-semibold text-sm"
                    onClick={(e) => {
                      e.stopPropagation()
                      // TODO: Open server settings
                    }}
                  >
                    Сутын
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
