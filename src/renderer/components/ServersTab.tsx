import { useState, useEffect } from 'react'
import { Server as ServerType } from '../../shared/types'
import { Play, RefreshCw, Loader2, Settings, Server, Download, Users } from 'lucide-react'
import { motion } from 'framer-motion'
import { cn } from '../lib/utils'
import { API_BASE_URL } from '../config/api'
import { getEmbeddedServers, EmbeddedServer } from '../utils/embeddedServers'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { Progress } from './ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Label } from './ui/label'
import { useI18n } from '../contexts/I18nContext'

interface GameAccount {
  id: number
  username: string
  email?: string
  uuid?: string
  server_id: number
}

interface ServersTabProps {
  authToken?: string | null
}

export function ServersTab({ authToken }: ServersTabProps) {
  const { t } = useI18n()
  const [servers, setServers] = useState<(ServerType & { embedded?: EmbeddedServer; players_online?: number; players_max?: number; clientInstalled?: boolean })[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isLaunching, setIsLaunching] = useState(false)
  const [installingServerId, setInstallingServerId] = useState<number | null>(null)
  const [installationProgress, setInstallationProgress] = useState<{ stage: string; progress: number } | null>(null)
  const [gameAccounts, setGameAccounts] = useState<GameAccount[]>([])
  const [selectedGameAccounts, setSelectedGameAccounts] = useState<Record<number, number>>({}) // server_id -> game_account_id

  useEffect(() => {
    loadServers()
    loadGameAccounts()
    
    // Listen for installation progress updates
    if (window.electronAPI?.onInstallationProgress) {
      window.electronAPI.onInstallationProgress((progress: { stage: string; progress: number }) => {
        setInstallationProgress(progress)
      })
    }
  }, [authToken])

  const loadGameAccounts = async () => {
    if (!authToken) return
    
    try {
      const response = await fetch(`${API_BASE_URL}/game-accounts/my-accounts`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
        },
      })
      
      if (response.ok) {
        const data = await response.json()
        setGameAccounts(data.accounts || [])
        
        // Auto-select first account for each server if not already selected
        const newSelections: Record<number, number> = { ...selectedGameAccounts }
        data.accounts?.forEach((account: GameAccount) => {
          if (!newSelections[account.server_id]) {
            newSelections[account.server_id] = account.id
          }
        })
        setSelectedGameAccounts(newSelections)
      }
    } catch (error) {
      console.error('Error loading game accounts:', error)
    }
  }

  useEffect(() => {
    // Check installation status for all servers when servers list changes
    const checkInstallationStatus = async () => {
      if (servers.length > 0 && window.electronAPI?.checkClientInstalled) {
        for (const server of servers) {
          // Only check if status is not already set
          if (server.clientInstalled === undefined) {
            const checkResult = await window.electronAPI.checkClientInstalled(server.id)
            if (checkResult.success) {
              setServers(prev => prev.map(s => 
                s.id === server.id ? { ...s, clientInstalled: checkResult.installed } : s
              ))
            }
          }
        }
      }
    }
    
    checkInstallationStatus()
  }, [servers.length])

  const loadServers = async () => {
    try {
      setIsLoading(true)
      
      // First, try to load embedded servers
      const embeddedServers = await getEmbeddedServers()
      
      if (embeddedServers.length > 0) {
        // Use embedded servers data
        const serversWithEmbedded = embeddedServers.map(embedded => ({
          id: embedded.server_id,
          name: embedded.server_name || 'Unknown Server',
          description: embedded.description || '',
          server_name: embedded.server_name,
          server_address: embedded.server_address,
          server_port: embedded.server_port || 25565,
          minecraft_version: embedded.minecraft_version || '1.20.1',
          preview_image_url: embedded.preview_image_url,
          server_uuid: embedded.server_uuid,
          server_status: 'unknown' as const,
          loader_enabled: false,
          loader_type: 'vanilla',
          embedded
        } as ServerType & { embedded: EmbeddedServer }))
        
        // Check status for each server
        const serversWithStatus = await Promise.all(
          serversWithEmbedded.map(async (server) => {
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
                  return { 
                    ...server, 
                    server_status: (statusData.online ? "online" : "offline") as "online" | "offline",
                    players_online: statusData.players?.online || 0,
                    players_max: statusData.players?.max || 0
                  }
                }
              } catch (error) {
                console.error(`Error checking status for server ${server.id}:`, error)
              }
            }
            return { ...server, server_status: "unknown" as const, players_online: 0, players_max: 0 }
          })
        )
        
        setServers(serversWithStatus)
      } else {
        // Fallback to API if no embedded servers
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
        
        if (authToken) {
          headers['Authorization'] = `Bearer ${authToken}`
        }
        
        const response = await fetch(`${API_BASE_URL}/servers`, {
          headers,
        })
        
        if (response.ok) {
          const data = await response.json()
          const serversList: ServerType[] = data.servers || []
          
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
              return { ...server, server_status: "unknown" as const, clientInstalled: false }
            })
          )
          
          setServers(serversWithStatus)
          
          // Check installation status for API servers
          if (window.electronAPI?.checkClientInstalled) {
            for (const server of serversWithStatus) {
              const checkResult = await window.electronAPI.checkClientInstalled(server.id)
              if (checkResult.success) {
                setServers(prev => prev.map(s => 
                  s.id === server.id ? { ...s, clientInstalled: checkResult.installed } : s
                ))
              }
            }
          }
        }
      }
    } catch (error) {
      console.error('Error loading servers:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleLaunch = async (server: ServerType) => {
    if (!window.electronAPI) {
      alert(t('servers.electronApiUnavailable'))
      return
    }

    try {
      setIsLaunching(true)
      const hwid = await window.electronAPI.getHwid()
      
      // Use embedded server data if available, otherwise use server data
      const serverAddress = (server as any).embedded?.server_address || server.server_address
      const serverPort = (server as any).embedded?.server_port || server.server_port
      
      if (!serverAddress || !serverPort) {
        alert(t('servers.serverAddressMissing'))
        return
      }

      if (server.server_uuid) {
        console.log(`Launching server ${server.name} with UUID: ${server.server_uuid}`)
        console.log(`Server: ${serverAddress}:${serverPort}`)
        console.log(`HWID: ${hwid}`)
      }

      // Get server details from API to get launch parameters
      let serverDetails: any = null
      try {
        const headers: HeadersInit = {
          'Content-Type': 'application/json',
        }
        
        if (authToken) {
          headers['Authorization'] = `Bearer ${authToken}`
        }

        const serverResponse = await fetch(`${API_BASE_URL}/servers/${server.id}`, {
          headers,
        })

        if (serverResponse.ok) {
          serverDetails = await serverResponse.json()
        }
      } catch (error) {
        console.warn('Could not fetch server details from API, using embedded data:', error)
      }

      console.log('Checking and updating mods...')
      const modsCheckResult = await window.electronAPI.checkAndUpdateMods(server.id, API_BASE_URL)
      
      if (!modsCheckResult.success) {
        alert(t('servers.modsCheckError', { error: modsCheckResult.error }))
        return
      }

      if (modsCheckResult.updated) {
        console.log(`Updated ${modsCheckResult.modsUpdated || 0} mod(s)`)
      }

      const launcherConfig = await window.electronAPI.getLauncherConfig()
      const dbConfig = await window.electronAPI.getLauncherDbConfig(server.id)
      
      if (!dbConfig.success || !dbConfig.config) {
        alert(t('servers.serverConfigError'))
        return
      }

      const minecraftVersion = server.minecraft_version || serverDetails?.minecraft_version || '1.20.1'
      const modsDir = modsCheckResult.modsDir || `~/.qmlauncher/mods/${server.id}`
      
      // Check if client is installed
      const clientCheck = await window.electronAPI.checkClientInstalled?.(server.id)
      if (clientCheck && !clientCheck.hasClient) {
        const installResult = await window.electronAPI.installMinecraftClient?.(minecraftVersion)
        if (!installResult?.success) {
          alert(t('servers.clientNotInstalled', { version: minecraftVersion }))
          return
        }
      }
      
      // Build JVM arguments from server config or use defaults
      const jvmArgsBase = serverDetails?.jvm_arguments 
        ? serverDetails.jvm_arguments.split(' ').filter((arg: string) => arg.trim())
        : []
      
      const jvmArgs = [
        `-Xmx${launcherConfig.maxMemory || serverDetails?.memory_mb || 4096}M`,
        `-Xms${launcherConfig.minMemory || 1024}M`,
        '-Djava.library.path=natives',
        '-Dminecraft.launcher.brand=qmlauncher',
        '-Dminecraft.launcher.version=1.0.0',
        ...jvmArgsBase
      ]

      // Build game arguments from server config or use defaults
      const gameArgsBase = serverDetails?.game_arguments
        ? serverDetails.game_arguments.split(' ').filter((arg: string) => arg.trim())
        : []

      const gameArgs = [
        '--username', 'Player',
        '--version', minecraftVersion,
        '--gameDir', modsDir,
        '--assetsDir', '~/.qmlauncher/assets',
        '--assetIndex', minecraftVersion,
        '--uuid', hwid || '00000000-0000-0000-0000-000000000000',
        '--accessToken', 'token',
        '--userType', 'mojang',
        '--versionType', 'release',
        '--server', serverAddress,
        '--port', String(serverPort),
        ...gameArgsBase
      ]

      console.log('Launching Minecraft with:', {
        server: `${serverAddress}:${serverPort}`,
        version: server.minecraft_version,
        jvmArgs: jvmArgs.length,
        gameArgs: gameArgs.length
      })

      const launchResult = await window.electronAPI.launchMinecraft({
        javaPath: launcherConfig.javaPath || 'java',
        jvmArgs,
        gameArgs,
        workingDirectory: modsDir,
        minecraftVersion,
        hwid,
        launcherConfig
      })

      if (!launchResult.success) {
        alert(t('servers.launchError', { error: launchResult.error }))
        return
      }

      console.log(`Minecraft launched with PID: ${launchResult.pid}`)
      console.log(`Auto-connecting to ${serverAddress}:${serverPort}`)
      
      // Update installation status after successful launch
      if (window.electronAPI?.checkClientInstalled) {
        const checkResult = await window.electronAPI.checkClientInstalled(server.id)
        if (checkResult.success && checkResult.installed) {
          setServers(prev => prev.map(s => 
            s.id === server.id ? { ...s, clientInstalled: true } : s
          ))
        }
      }
    } catch (error) {
      console.error('Error launching:', error)
      alert(t('servers.launchErrorGeneric', { error: error instanceof Error ? error.message : String(error) }))
    } finally {
      setIsLaunching(false)
    }
  }

  const getServerImage = (server: ServerType) => {
    if (server.preview_image_url) {
      return server.preview_image_url
    }
    return '/minecraft-server-preview.svg'
  }

  const isClientInstalled = (server: ServerType & { clientInstalled?: boolean }): boolean => {
    // Use cached installation status if available
    if (server.clientInstalled !== undefined) {
      return server.clientInstalled
    }
    // Default to false if not checked yet
    return false
  }

  const handleInstall = async (server: ServerType) => {
    if (!window.electronAPI) {
      alert(t('servers.electronApiUnavailable'))
      return
    }

    try {
      setInstallingServerId(server.id)
      
      // Step 1: Check and download Minecraft client for the version
      const minecraftVersion = server.minecraft_version || '1.20.1'
      console.log(`Installing Minecraft client version ${minecraftVersion}...`)
      
      if (!window.electronAPI?.installMinecraftClient) {
        alert(t('servers.installClientUnavailable'))
        return
      }
      
      // Check if client is already installed
      if (!window.electronAPI?.checkClientInstalled) {
        alert(t('servers.checkClientUnavailable'))
        return
      }
      
      const checkResult = await window.electronAPI.checkClientInstalled(server.id)
      if (checkResult.success && checkResult.hasClient) {
        console.log(`Minecraft client ${minecraftVersion} already installed`)
      } else {
        // Download and install Minecraft client
        console.log(`Downloading Minecraft client ${minecraftVersion}...`)
        const installResult = await window.electronAPI.installMinecraftClient(minecraftVersion)
        
        if (!installResult.success) {
          alert(t('servers.installClientError', { error: installResult.error || t('common.unknownError') }))
          return
        }
        
        if (installResult.alreadyInstalled) {
          console.log(installResult.message)
        } else {
          console.log(installResult.message)
          console.log(`Downloaded ${installResult.librariesDownloaded || 0} libraries`)
        }
      }
      
      // Step 2: Download and install mods
      console.log('Downloading mods...')
      const modsCheckResult = await window.electronAPI.checkAndUpdateMods(server.id, API_BASE_URL)
      
      if (!modsCheckResult.success) {
        alert(t('servers.installModsError', { error: modsCheckResult.error }))
        return
      }

      if (modsCheckResult.updated) {
        console.log(`Downloaded ${modsCheckResult.modsUpdated || 0} mod(s)`)
      }

      // Step 3: Verify installation
      const dbConfig = await window.electronAPI.getLauncherDbConfig(server.id)
      if (!dbConfig.success) {
        alert(t('servers.installConfigError'))
        return
      }

      // Installation complete
      console.log(`Installation complete for server ${server.name}`)
      
      // Update installation status for this server
      if (window.electronAPI && window.electronAPI.checkClientInstalled) {
        const checkResult = await window.electronAPI.checkClientInstalled(server.id)
        if (checkResult.success) {
          setServers(prev => prev.map(s => 
            s.id === server.id ? { ...s, clientInstalled: checkResult.installed } : s
          ))
        }
      }
      
      alert(t('servers.installComplete', { name: server.name }))
    } catch (error) {
      console.error('Error installing:', error)
      setInstallationProgress(null)
      alert(t('servers.installError', { error: error instanceof Error ? error.message : String(error) }))
    } finally {
      setInstallingServerId(null)
      setInstallationProgress(null)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center">
          <Loader2 className="w-12 h-12 animate-spin text-primary mx-auto mb-4" />
          <p className="text-muted-foreground">{t('servers.loading')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 h-full overflow-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6 no-drag">
        <div className="no-drag">
          <h2 className="text-3xl font-bold mb-2 text-foreground">
            {t('servers.title')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {servers.length} {servers.length === 1 ? t('servers.server') : servers.length < 5 ? t('servers.servers2') : t('servers.servers5')}
          </p>
        </div>
        
        <Button
          variant="outline"
          size="icon"
          onClick={async () => {
            await loadServers()
            // Re-check installation status after refresh
            const currentServers = servers
            if (currentServers.length > 0 && window.electronAPI?.checkClientInstalled) {
              for (const server of currentServers) {
                const checkResult = await window.electronAPI.checkClientInstalled(server.id)
                if (checkResult.success) {
                  setServers(prev => prev.map(s => 
                    s.id === server.id ? { ...s, clientInstalled: checkResult.installed } : s
                  ))
                }
              }
            }
          }}
          disabled={isLoading}
          className="no-drag"
          title={t('servers.refresh')}
        >
          <RefreshCw className={cn("w-5 h-5", isLoading && "animate-spin")} />
        </Button>
      </div>

      {/* Servers Grid */}
      {servers.length === 0 ? (
        <div className="text-center py-20 no-drag">
          <div className="w-24 h-24 mx-auto mb-6 rounded-2xl bg-muted flex items-center justify-center">
            <Server className="w-12 h-12 text-muted-foreground" />
          </div>
          <p className="text-xl text-foreground mb-2">Нет доступных серверов</p>
          <p className="text-muted-foreground">Создайте сервер в QMAdmin для отображения здесь</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 no-drag">
          {servers.map((server, index) => (
            <motion.div
              key={server.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: index * 0.05 }}
              whileHover={{ y: -4 }}
            >
              <Card className="group relative overflow-hidden hover:shadow-lg transition-all cursor-pointer no-drag">
                {/* Server Image */}
                <div className="relative w-full h-48 overflow-hidden bg-muted">
                  <img
                    src={getServerImage(server)}
                    alt={server.name}
                    className="w-full h-full object-cover group-hover:scale-110 transition-transform duration-500"
                    onError={(e) => {
                      e.currentTarget.style.display = 'none'
                      const parent = e.currentTarget.parentElement!
                      if (!parent.querySelector('.fallback')) {
                        parent.innerHTML = '<div class="fallback w-full h-full flex items-center justify-center text-6xl">🎮</div>'
                      }
                    }}
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-background via-background/50 to-transparent" />
                  
                  {/* Status Badge */}
                  <div className="absolute top-4 right-4">
                    {server.server_status === "online" && (
                      <Badge variant="default" className="bg-green-500/90 backdrop-blur-sm flex items-center gap-1.5">
                        <span className="w-2 h-2 bg-white rounded-full animate-pulse" />
                        Онлайн
                      </Badge>
                    )}
                    {server.server_status === "offline" && (
                      <Badge variant="destructive" className="backdrop-blur-sm">
                        Оффлайн
                      </Badge>
                    )}
                  </div>
                </div>

                <CardHeader>
                  <CardTitle className="line-clamp-1">
                    {server.server_name || server.name}
                  </CardTitle>
                  {server.server_address && (
                    <CardDescription className="font-mono text-xs">
                      {server.server_address}:{server.server_port}
                    </CardDescription>
                  )}
                </CardHeader>

                <CardContent className="space-y-3">
                  {/* Version and Players */}
                  <div className="flex items-center justify-between">
                    <Badge variant="secondary">
                      MC {server.minecraft_version || 'N/A'}
                    </Badge>
                    
                    {(server as any).players_online !== undefined && (
                      <div className="flex items-center gap-1.5">
                        <Users className="w-4 h-4 text-muted-foreground" />
                        <span className="text-xs font-medium text-muted-foreground">
                          {(server as any).players_online || 0}/{(server as any).players_max || 0}
                        </span>
                      </div>
                    )}
                  </div>

                  {/* Description */}
                  {server.description && (
                    <CardDescription className="line-clamp-2">
                      {server.description}
                    </CardDescription>
                  )}

                  {/* Game Account Selection */}
                  {authToken && gameAccounts.filter(acc => acc.server_id === server.id).length > 0 && (
                    <div className="space-y-2">
                      <Label htmlFor={`game-account-${server.id}`} className="text-xs">
                        {t('servers.selectGameAccount')}
                      </Label>
                      <Select
                        value={selectedGameAccounts[server.id]?.toString() || ''}
                        onValueChange={(value: string) => {
                          setSelectedGameAccounts(prev => ({
                            ...prev,
                            [server.id]: parseInt(value)
                          }))
                        }}
                      >
                        <SelectTrigger id={`game-account-${server.id}`} className="w-full">
                          <SelectValue placeholder={t('servers.selectGameAccount')} />
                        </SelectTrigger>
                        <SelectContent>
                          {gameAccounts
                            .filter(acc => acc.server_id === server.id)
                            .map((account) => (
                              <SelectItem key={account.id} value={account.id.toString()}>
                                {account.username}
                              </SelectItem>
                            ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                </CardContent>

                <CardFooter className="flex gap-2 flex-col">
                  {isClientInstalled(server) ? (
                    <Button
                      onClick={(e) => {
                        e.stopPropagation()
                        handleLaunch(server)
                      }}
                      disabled={isLaunching || server.server_status === "offline" || (authToken && gameAccounts.filter(acc => acc.server_id === server.id).length > 0 && !selectedGameAccounts[server.id]) || false}
                      className="w-full"
                      size="sm"
                    >
                      {isLaunching ? (
                        <Loader2 className="w-4 h-4 animate-spin mr-2" />
                      ) : (
                        <Play className="w-4 h-4 mr-2" />
                      )}
                      {t('servers.play')}
                    </Button>
                  ) : (
                    <div className="flex-1 flex flex-col gap-2">
                      <Button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleInstall(server)
                        }}
                        disabled={installingServerId === server.id}
                        className="w-full"
                        size="sm"
                      >
                        {installingServerId === server.id ? (
                          <>
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                            <span className="truncate text-xs">
                              {installationProgress?.stage || t('servers.installing')}
                            </span>
                          </>
                        ) : (
                          <>
                            <Download className="w-4 h-4 mr-2" />
                            {t('servers.install')}
                          </>
                        )}
                      </Button>
                      {installingServerId === server.id && installationProgress && (
                        <Progress value={installationProgress.progress} className="h-1.5" />
                      )}
                    </div>
                  )}
                  
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={(e) => {
                      e.stopPropagation()
                      // TODO: Open server settings
                    }}
                    title={t('servers.settings')}
                  >
                    <Settings className="w-4 h-4" />
                  </Button>
                </CardFooter>
              </Card>
            </motion.div>
          ))}
        </div>
      )}
    </div>
  )
}
