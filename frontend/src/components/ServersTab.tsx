import { useState, useEffect } from 'react'
import { Server as ServerType } from '../shared/types'
import { Play, RefreshCw, Loader2, Server, Download, Users, Trash2, Info, Settings, Search } from 'lucide-react'
import { motion } from 'framer-motion'
import { cn } from '../lib/utils'
import { API_BASE_URL } from '../config/api'
import { getEmbeddedServers, EmbeddedServer } from '../utils/embeddedServers'
import { api } from '../utils/api-client'
import { wailsAPI } from '../bridge'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useDebounce } from '../utils/debounce'
import { logger } from '../utils/logger'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { Progress } from './ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { Label } from './ui/label'
import { Input } from './ui/input'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog'
import { ServerSettingsDialog } from './ServerSettingsDialog'
import { useI18n } from '../contexts/I18nContext'
import { toast } from 'sonner'
import { Skeleton } from './ui/skeleton'
import { LogInfo, LogError } from '../../wailsjs/runtime/runtime'
import forgeIcon from '../assets/loaders/forge.png'
import fabricIcon from '../assets/loaders/fabric.png'
import quiltIcon from '../assets/loaders/quilt.png'
import neoforgeIcon from '../assets/loaders/neoforge.png'
import vanillaIcon from '../assets/loaders/vanila.png'

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
  const logBackend = (prefix: string, data?: any) => {
    try {
      const serialized = data === undefined ? '' : ` ${JSON.stringify(data, null, 2)}`
      LogInfo(`${prefix}${serialized}`)
    } catch (err) {
      console.error('Backend log failed:', err)
    }
  }
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const [servers, setServers] = useState<(ServerType & { embedded?: EmbeddedServer; players_online?: number; players_max?: number; clientInstalled?: boolean })[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isLaunching, setIsLaunching] = useState(false)
  const [installingServerId, setInstallingServerId] = useState<number | null>(null)
  const [installationProgress, setInstallationProgress] = useState<{ stage: string; progress: number } | null>(null)
  const [gameAccounts, setGameAccounts] = useState<GameAccount[]>([])
  const [selectedGameAccounts, setSelectedGameAccounts] = useState<Record<number, number>>({}) // server_id -> game_account_id
  const [selectedServerForSettings, setSelectedServerForSettings] = useState<ServerType | null>(null)
  const [selectedServerForInfo, setSelectedServerForInfo] = useState<ServerType | null>(null)
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false)
  const [serverToUninstall, setServerToUninstall] = useState<ServerType | null>(null)
  const [uninstallingServerId, setUninstallingServerId] = useState<number | null>(null)
  const [uninstallProgress, setUninstallProgress] = useState<{ stage: string; progress: number } | null>(null)
  const [runningGameAccounts, setRunningGameAccounts] = useState<Record<number, string | null>>({})
  const [searchQuery, setSearchQuery] = useState('')
  const debouncedSearchQuery = useDebounce(searchQuery, 300)

  const handleAuthError = async () => {
    // Reload the app to show login form
    window.location.reload()
  }

  useEffect(() => {
    loadServers()
    loadGameAccounts()
  }, [authToken])

  // Периодически проверяем, запущен ли Minecraft, и обновляем состояние кнопки
  useEffect(() => {
    const checkMinecraftRunning = async () => {
      try {
        const isRunning = await wailsAPI.isMinecraftRunning()
        if (!isRunning) {
          // Если Minecraft не запущен, очищаем состояние запущенных игр
          setRunningGameAccounts({})
        }
      } catch (error) {
        console.error('[ServersTab] Error checking Minecraft status:', error)
      }
    }

    // Проверяем сразу при монтировании
    checkMinecraftRunning()

    // Проверяем каждые 2 секунды
    const interval = setInterval(checkMinecraftRunning, 2000)

    return () => clearInterval(interval)
  }, [])

  const loadGameAccounts = async () => {
    if (!authToken) return
    
    try {
      const response = await api.get<{ accounts: GameAccount[] }>('/game-accounts/my-accounts', {
        authToken,
      })
      
      // Handle 401 Unauthorized - token is invalid
      if (response.status === 401) {
        console.error('[ServersTab] Unauthorized when loading game accounts, token is invalid')
        await handleAuthError()
        return
      }
      
      if (response.ok && response.data.accounts) {
        setGameAccounts(response.data.accounts)
        
        // Auto-select first account for each server if not already selected
        const newSelections: Record<number, number> = { ...selectedGameAccounts }
        response.data.accounts.forEach((account: GameAccount) => {
          if (!newSelections[account.server_id]) {
            newSelections[account.server_id] = account.id
          }
        })
        setSelectedGameAccounts(newSelections)
      } else {
        setGameAccounts([])
      }
    } catch (error) {
      logger.error('Error loading game accounts', error instanceof Error ? error : new Error(String(error)))
      setGameAccounts([])
    }
  }

  useEffect(() => {
    // Check installation status for all servers when servers list changes
    // Оптимизация: параллельные запросы вместо последовательных
    const checkInstallationStatus = async () => {
      if (servers.length > 0) {
        // Создаем массив промисов для параллельного выполнения
        const checkPromises = servers.map(async (server) => {
          const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
          try {
            const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuid)
            return {
              serverId: server.id,
              isInstalled: checkResult.installed || checkResult.hasClient,
              checkResult
            }
          } catch (error) {
            console.error(`Error checking installation for server ${server.id}:`, error)
            return {
              serverId: server.id,
              isInstalled: undefined,
              error
            }
          }
        })
        
        // Ждем все проверки параллельно
        const results = await Promise.allSettled(checkPromises)
        
        // Обновляем состояние одним батчем
        setServers(prev => prev.map(s => {
          const result = results.find((r, idx) => 
            r.status === 'fulfilled' && r.value.serverId === s.id
          )
          
          if (result && result.status === 'fulfilled') {
            const { isInstalled } = result.value
            const currentInstalled = s.clientInstalled
            // Only update if check says installed, or if current state is undefined
            if (isInstalled !== undefined && (isInstalled || currentInstalled === undefined)) {
              return { ...s, clientInstalled: isInstalled }
            }
          }
          return s
        }))
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
        
        // Сначала показываем серверы со статусом unknown
        const serversWithUnknownStatus = serversWithEmbedded.map(server => ({
          ...server,
          server_status: "unknown" as const,
          players_online: 0,
          players_max: 0,
          clientInstalled: undefined
        }))
        setServers(serversWithUnknownStatus)
        
        // Затем асинхронно обновляем статусы с таймаутом
        const updateStatusPromises = serversWithEmbedded.map(async (server) => {
          if (server.server_address && server.server_port) {
            try {
              const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error('Timeout')), 3000)
              )
              
              const statusPromise = api.post<{ online: boolean; players_online?: number; players_max?: number }>(
                '/minecraft-server/status',
                {
                  host: server.server_address,
                  port: server.server_port,
                  method: "auto",
                }
              )
              
              const statusResponse = await Promise.race([statusPromise, timeoutPromise]) as typeof statusPromise extends Promise<infer T> ? T : never
              
              if (statusResponse.ok && statusResponse.data) {
                const statusData = statusResponse.data
                setServers(prev => prev.map(s => 
                  s.id === server.id 
                    ? { 
                        ...s, 
                        server_status: (statusData.online ? "online" : "offline") as "online" | "offline",
                        players_online: statusData.players_online || 0,
                        players_max: statusData.players_max || 0
                      }
                    : s
                ))
              }
            } catch (error) {
              // Игнорируем ошибки, статус остается unknown
            }
          }
        })
        
        // Не ждем завершения всех проверок
        Promise.all(updateStatusPromises).catch(() => {})
        
        // Check installation status for embedded servers
        for (const server of serversWithUnknownStatus) {
          const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
          try {
            const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuid)
            // ClientCheckResult fields are camelCase in TypeScript models
            const isInstalled = checkResult.installed || checkResult.hasClient
            if (isInstalled) {
              setServers(prev => prev.map(s => {
                if (s.id === server.id) {
                  // Don't overwrite true with false
                  const isInstalled = checkResult.installed || checkResult.hasClient
                  const currentInstalled = s.clientInstalled
                  if (isInstalled || currentInstalled === undefined) {
                    return { ...s, clientInstalled: isInstalled }
                  }
                  return s
                }
                return s
              }))
            }
          } catch (error) {
            console.error(`Error checking installation for server ${server.id}:`, error)
          }
        }
      } else {
        // Fallback to API if no embedded servers
        let hwid: string | null = null
        try {
          hwid = await wailsAPI.getHwid()
        } catch (error) {
          console.error('Error getting HWID:', error)
        }
        
        const headers: HeadersInit = {
          'Content-Type': 'application/json',
        }
        
        if (hwid) {
          headers['X-HWID'] = hwid
        }
        
        const response = await api.get<{ servers: ServerType[] }>('/servers', {
          authToken,
        })
        
        // Handle 401 Unauthorized - token is invalid
        if (response.status === 401) {
          console.error('[ServersTab] Unauthorized, token is invalid')
          await handleAuthError()
          return
        }
        
        if (response.ok && response.data.servers) {
          const serversList: ServerType[] = response.data.servers
          
          // Сначала показываем серверы со статусом unknown
          const serversWithUnknownStatus = serversList.map(server => ({
            ...server,
            server_status: "unknown" as const,
            clientInstalled: undefined
          }))
          setServers(serversWithUnknownStatus)
          
          // Затем асинхронно обновляем статусы с таймаутом
          const updateStatusPromises = serversList.map(async (server) => {
            if (server.server_address && server.server_port) {
              try {
                const timeoutPromise = new Promise((_, reject) => 
                  setTimeout(() => reject(new Error('Timeout')), 3000)
                )
                
                const statusPromise = api.post<{ online: boolean; players_online?: number; players_max?: number }>(
                  '/minecraft-server/status',
                  {
                    host: server.server_address,
                    port: server.server_port,
                    method: "auto",
                  }
                )
                
                const statusResponse = await Promise.race([statusPromise, timeoutPromise]) as typeof statusPromise extends Promise<infer T> ? T : never
                
                if (statusResponse.ok && statusResponse.data) {
                  const statusData = statusResponse.data
                  setServers(prev => prev.map(s => 
                    s.id === server.id 
                      ? { 
                          ...s, 
                          server_status: (statusData.online ? "online" : "offline") as "online" | "offline",
                          players_online: statusData.players_online || 0,
                          players_max: statusData.players_max || 0
                        }
                      : s
                  ))
                }
              } catch (error) {
                // Игнорируем ошибки, статус остается unknown
              }
            }
          })
          
          // Не ждем завершения всех проверок
          Promise.all(updateStatusPromises).catch(() => {})
          
          // Check installation status for API servers
          for (const server of serversWithUnknownStatus) {
            const serverUuid = server.server_uuid || String(server.id)
            try {
              const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuid)
              // ClientCheckResult doesn't have Success field, check Installed or HasClient directly
              if (checkResult.installed || checkResult.hasClient) {
                setServers(prev => prev.map(s => {
                  if (s.id === server.id) {
                    // Don't overwrite true with false
                    const isInstalled = checkResult.installed || checkResult.hasClient
                    const currentInstalled = s.clientInstalled
                    if (isInstalled || currentInstalled === undefined) {
                      return { ...s, clientInstalled: isInstalled }
                    }
                    return s
                  }
                  return s
                }))
              }
            } catch (error) {
              console.error(`Error checking installation for server ${server.id}:`, error)
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
    try {
      setInstallingServerId(server.id)
      setInstallationProgress({ stage: t('servers.checkingJava') || 'Проверка Java...', progress: 10 })
      
      // Step 1: Check and download Minecraft client for the version
      const minecraftVersion = server.minecraft_version || '1.20.1'
      
      // Get server UUID for installation
      const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
      
      setInstallationProgress({ stage: t('servers.checkingClient') || 'Проверка клиента...', progress: 20 })
      const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuid)
      if (!checkResult.hasClient) {
        // Get Java vendor and version from server, or use defaults
        const serverDetails = (server as any).embedded || server
        const javaVendor = serverDetails?.java_vendor || 'openjdk'
        const javaVersion = serverDetails?.java_version || '17'
        
        // Download and install Java
        setInstallationProgress({ stage: t('servers.installingJava') || 'Установка Java...', progress: 30 })
        try {
          await wailsAPI.installJava(javaVendor, javaVersion, serverUuid)
          logger.debug('Java installation completed')
        } catch (err) {
          logger.error('Exception from installJava', err instanceof Error ? err : new Error(String(err)))
          toast.error(t('servers.installJavaError') || 'Ошибка установки Java', {
            description: err instanceof Error ? err.message : String(err)
          })
          return
        }
        
        // Download and install Minecraft client
        setInstallationProgress({ stage: t('servers.installingClient') || 'Установка клиента Minecraft...', progress: 50 })
      logger.debug('Calling installMinecraftClient')
      logBackend('[Install] Calling installMinecraftClient', { version: minecraftVersion, serverUuid, javaVendor, javaVersion })
        try {
          const installResult = await wailsAPI.installMinecraftClient(minecraftVersion, javaVendor, javaVersion, serverUuid)
          if (import.meta.env.DEV) {
            logger.debug('installMinecraftClient result', { 
              type: typeof installResult,
              keys: installResult ? Object.keys(installResult) : null
            })
          }
        logBackend('[Install] installMinecraftClient result', installResult)
          
          // Если функция вернула результат без исключения, считаем установку успешной
          // Проверяем только явные ошибки
          if (installResult) {
            const hasError = installResult?.Error || (installResult as any)?.error
            const isFailed = installResult?.Success === false && !installResult?.AlreadyInstalled && !(installResult as any)?.success && !(installResult as any)?.alreadyInstalled
            
            if (hasError && isFailed) {
              const desc = hasError || installResult?.Message || (installResult as any)?.message || t('common.unknownError')
              logger.error('Installation failed', undefined, { error: desc })
              toast.error(t('servers.installClientError') || 'Ошибка установки клиента', {
                description: desc
              })
              return
            }
          }
          
          logger.debug('Client installation completed successfully')
        } catch (err) {
          logger.error('Exception from installMinecraftClient', err instanceof Error ? err : new Error(String(err)))
        logBackend('[Install] Exception from installMinecraftClient', { error: err instanceof Error ? err.message : String(err) })
          toast.error(t('servers.installClientError') || 'Ошибка установки клиента', {
            description: err instanceof Error ? err.message : String(err)
          })
          return
        }
        
        // Client installation completed
      }
      
      // Step 2: Download and install mods (only after client is installed)
      logger.debug('Step 2: Checking mods')
      logBackend('[Install] Step 2: Checking mods', { serverId: server.id })
      setInstallationProgress({ stage: t('servers.checkingMods') || 'Проверка модов...', progress: 80 })
      let modsCheckResult
      try {
        modsCheckResult = await wailsAPI.checkAndUpdateMods(server.id, API_BASE_URL)
        if (import.meta.env.DEV) {
          logger.debug('checkAndUpdateMods result', { result: modsCheckResult })
        }
        logBackend('[Install] checkAndUpdateMods result', modsCheckResult)
      } catch (err) {
        logger.error('Exception from checkAndUpdateMods', err instanceof Error ? err : new Error(String(err)))
        logBackend('[Install] Exception from checkAndUpdateMods', { error: err instanceof Error ? err.message : String(err) })
        throw err
      }
      
      if (!modsCheckResult.success) {
        logger.error('Mods check failed', undefined, { error: modsCheckResult.error })
        toast.error(t('servers.installModsError') || 'Ошибка установки модов', {
          description: modsCheckResult.error
        })
        return
      }

      logger.debug('Mods check completed')
      // Mods check completed (no notification if no mods)

      // Step 3: Verify installation
      logger.debug('Step 3: Verifying installation')
      logBackend('[Install] Step 3: Verifying installation', { serverId: server.id })
      let dbConfig
      try {
        dbConfig = await wailsAPI.getLauncherDbConfig(server.id)
        if (import.meta.env.DEV) {
          logger.debug('getLauncherDbConfig result', { config: dbConfig })
        }
        logBackend('[Install] getLauncherDbConfig result', dbConfig)
      } catch (err) {
        logger.error('Exception from getLauncherDbConfig', err instanceof Error ? err : new Error(String(err)))
        logBackend('[Install] Exception from getLauncherDbConfig', { error: err instanceof Error ? err.message : String(err) })
        throw err
      }
      
      if (!dbConfig.success) {
        logger.error('Config check failed')
        toast.error(t('servers.installConfigError') || 'Ошибка получения конфигурации')
        return
      }
      
      logger.debug('All steps completed successfully')

      // Installation complete - show success notification
      setInstallationProgress({ stage: t('servers.installationComplete') || 'Установка завершена', progress: 100 })
      toast.success(t('servers.installComplete') || 'Установка завершена', {
        description: `Сервер: ${server.name}`
      })
      
      // Update installation status for this server - force update after successful installation
      // First, optimistically set as installed
      setServers(prev => prev.map(s => 
        s.id === server.id ? { ...s, clientInstalled: true } : s
      ))
      
      // Reset installing state immediately (before verification)
      setInstallingServerId(null)
      setInstallationProgress(null)
      
      // Then verify with actual check (async, don't wait for it to block UI)
      // Note: We set clientInstalled to true optimistically, so even if check fails,
      // the button will remain "Играть"
      setTimeout(async () => {
        const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
        try {
          const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuid)
          if (import.meta.env.DEV) {
            logger.debug(`Verification check for server ${server.id}`, { checkResult })
          }
          // ClientCheckResult doesn't have Success field, check Installed or HasClient directly
          if (checkResult.installed || checkResult.hasClient) {
            // Use HasClient as primary indicator (client is installed if JAR exists)
            // Installed flag may require mods, but HasClient just checks for JAR
            const isInstalled = checkResult.hasClient || checkResult.installed
            if (import.meta.env.DEV) {
              logger.debug(`Server ${server.id} installation status`, { hasClient: checkResult.hasClient, installed: checkResult.installed, settingTo: isInstalled })
            }
            // Only update if check confirms installation, don't overwrite true with false
            if (isInstalled) {
              setServers(prev => prev.map(s => 
                s.id === server.id ? { ...s, clientInstalled: true } : s
              ))
            } else {
              logger.warn(`Check returned false for server ${server.id}, but keeping optimistic true state`)
              // Keep optimistic true state - don't overwrite
            }
          } else {
            // If check failed, keep the optimistic true value
            logger.warn(`Check failed for server ${server.id}, keeping optimistic installed state`)
          }
        } catch (error) {
          logger.error(`Error during verification check for server ${server.id}`, error instanceof Error ? error : new Error(String(error)))
          // Keep optimistic true state on error
        }
      }, 3000) // Increased timeout to 3 seconds to ensure files are fully written
      
      // No alert - installation is complete, button will change automatically
      logger.debug('Installation flow completed without errors')
    } catch (error) {
      logger.error('Error installing (frontend)', error instanceof Error ? error : new Error(String(error)), {
        name: error instanceof Error ? error.name : 'Unknown',
        type: typeof error,
      })
      setInstallationProgress(null)
      setInstallingServerId(null)
      const errorMessage = error instanceof Error ? error.message : String(error)
      toast.error(t('servers.installError') || 'Ошибка установки', {
        description: errorMessage || t('common.unknownError')
      })
    } finally {
      logger.debug('Finally block: resetting installation state')
      setInstallingServerId(null)
      setInstallationProgress(null)
    }
  }

  const handleLaunch = async (server: ServerType) => {
    const currentAccountId = selectedGameAccounts[server.id] ? String(selectedGameAccounts[server.id]) : 'default'
    const alreadyRunningForAccount = runningGameAccounts[server.id] === currentAccountId

    if (alreadyRunningForAccount) {
      toast.info(t('servers.alreadyRunning') || 'Клиент уже запущен для этого пользователя')
      return
    }

    try {
      setIsLaunching(true)
      const launchToastId = toast.loading(t('servers.launching') || 'Запуск Minecraft...')
      const hwid = await wailsAPI.getHwid()
      
      // Use embedded server data if available, otherwise use server data
      const serverAddress = (server as any).embedded?.server_address || server.server_address
      const serverPort = (server as any).embedded?.server_port || server.server_port
      
      if (!serverAddress || !serverPort) {
        toast.error(t('servers.serverAddressMissing') || 'Адрес сервера не указан', { id: launchToastId })
        return
      }

      // Launching server

      // Get server details from API to get launch parameters
      let serverDetails: any = null
      try {
        const serverResponse = await api.get(`/servers/${server.id}`, {
          authToken,
        })

        if (serverResponse.ok && serverResponse.data) {
          serverDetails = serverResponse.data
        }
      } catch (error) {
        console.warn('Could not fetch server details from API, using embedded data:', error)
      }

      const settings = await wailsAPI.getSettings()
      const dbConfig = await wailsAPI.getLauncherDbConfig(server.id)
      
      if (!dbConfig.success || !dbConfig.config) {
        toast.error(t('servers.serverConfigError') || 'Ошибка конфигурации сервера')
        return
      }

      const minecraftVersion = server.minecraft_version || serverDetails?.minecraft_version || '1.20.1'
      // Use server_uuid if available, otherwise fallback to server.id
      const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
      // Use custom path from settings if provided, otherwise use default with server_uuid
      const minecraftBasePath = settings.minecraftPath || `~/.qmlauncher/${serverUuid}/minecraft`
      const modsDir = `${minecraftBasePath}/mods`
      
      // Get Java vendor and version from server, or use defaults
      const javaVendor = serverDetails?.java_vendor || 'openjdk'
      const javaVersion = serverDetails?.java_version || '17'
      
      // Check if client is installed
      const clientCheck = await wailsAPI.checkClientInstalled(server.id, serverUuid)
      
      // Check and update mods (only if client is installed)
      if (clientCheck && clientCheck.hasClient) {
        const modsCheckResult = await wailsAPI.checkAndUpdateMods(server.id, API_BASE_URL)
        
        if (!modsCheckResult.success) {
          toast.error(t('servers.modsCheckError') || 'Ошибка проверки модов', {
            description: modsCheckResult.error
          })
          return
        }
      }
      if (clientCheck && !clientCheck.hasClient) {
        const installResult = await wailsAPI.installMinecraftClient(minecraftVersion, javaVendor, javaVersion, serverUuid)
        if (!installResult?.Success) {
          toast.error(t('servers.clientNotInstalled') || 'Клиент не установлен', {
            description: `Версия: ${minecraftVersion}`
          })
          return
        }
      }
      
      // Build JVM arguments - prioritize server JVM args, fallback to settings defaults
      // Xmx is always 2G, Xms from settings memory slider
      const memoryMb = settings.minMemory || 1024
      
      let jvmArgs: string[] = []
      
      // If server has JVM arguments, use them (they should already include Xmx and Xms)
      if (serverDetails?.jvm_arguments) {
        const serverJvmArgs = serverDetails.jvm_arguments.split(' ').filter((arg: string) => arg.trim())
        // Ensure Xmx is always 2G and Xms is from memory slider
        const filteredServerArgs = serverJvmArgs.filter((arg: string) => 
          !arg.startsWith('-Xmx') && !arg.startsWith('-Xms')
        )
        jvmArgs = [
          '-Xmx2G', // Always 2G
          `-Xms${memoryMb}M`, // From memory slider
          ...filteredServerArgs,
          '-Djava.library.path=natives',
          '-Dminecraft.launcher.brand=qmlauncher',
          '-Dminecraft.launcher.version=1.0.0',
        ]
      } else {
        // Use default JVM args from settings if server doesn't have them
        const defaultJvmArgs = settings.jvmArgs || []
        // Remove existing Xmx and Xms from default args to avoid duplicates
        const filteredDefaultArgs = defaultJvmArgs.filter((arg: string) => 
          !arg.startsWith('-Xmx') && !arg.startsWith('-Xms')
        )
        
        jvmArgs = [
          '-Xmx2G', // Always 2G
          `-Xms${memoryMb}M`, // From memory slider
          ...filteredDefaultArgs,
          '-Djava.library.path=natives',
          '-Dminecraft.launcher.brand=qmlauncher',
          '-Dminecraft.launcher.version=1.0.0',
        ]
      }

      // Build game arguments from server config or use defaults
      let gameArgsBase = serverDetails?.game_arguments
        ? serverDetails.game_arguments.split(' ').filter((arg: string) => arg.trim())
        : []

      // Удаляем --server и --port из gameArgsBase, если они там есть
      // чтобы наши значения имели приоритет
      gameArgsBase = gameArgsBase.filter((arg: string, index: number) => {
        // Пропускаем --server и следующий за ним аргумент
        if (arg === '--server') {
          return false
        }
        // Пропускаем аргумент после --server (адрес сервера)
        if (index > 0 && gameArgsBase[index - 1] === '--server') {
          return false
        }
        // Пропускаем --port и следующий за ним аргумент
        if (arg === '--port') {
          return false
        }
        // Пропускаем аргумент после --port (порт сервера)
        if (index > 0 && gameArgsBase[index - 1] === '--port') {
          return false
        }
        return true
      })

      // Get resolution from settings - use defaults for now
      const windowWidth = 1920
      const windowHeight = 1080

      // Формируем строку подключения к серверу для quickPlayMultiplayer
      // Формат: server:port (например, "example.com:25565")
      const serverConnectionString = `${serverAddress}:${serverPort}`
      
      const gameArgs = [
        '--username', 'Player',
        '--version', minecraftVersion,
        '--gameDir', minecraftBasePath, // Use minecraftBasePath, not modsDir
        '--assetsDir', `${minecraftBasePath}/assets`,
        '--assetIndex', minecraftVersion,
        '--uuid', hwid || '00000000-0000-0000-0000-000000000000',
        '--accessToken', '', // Пустой токен для офлайн режима (Minecraft требует этот аргумент)
        '--userType', 'mojang',
        '--versionType', 'release',
        '--width', String(windowWidth),
        '--height', String(windowHeight),
        ...gameArgsBase, // Сначала добавляем аргументы из конфига
        // Используем оба варианта для максимальной совместимости
        '--server', serverAddress, // Старый формат для совместимости
        '--port', String(serverPort), // Старый формат для совместимости
        '--quickPlayMultiplayer', serverConnectionString // Новый формат для автоматического подключения
      ]
      
      // Логирование только в dev режиме
      if (import.meta.env.DEV) {
        logger.debug('Server connection string', { serverConnectionString })
        logger.debug('Game args with server', { 
          args: gameArgs.filter((arg, i) => 
            arg === '--server' || arg === '--port' || arg === '--quickPlayMultiplayer' || 
            (i > 0 && (gameArgs[i-1] === '--server' || gameArgs[i-1] === '--port' || gameArgs[i-1] === '--quickPlayMultiplayer'))
          )
        })
      }

      // Launch Minecraft
      
      // Ensure Java is installed with the correct vendor and version for this server
      try {
        await wailsAPI.installJava(javaVendor, javaVersion, serverUuid)
      } catch (error) {
        console.warn('Failed to install Java:', error)
      }
      
      // Get Java path
      const platform = await wailsAPI.getPlatform()
      let javaExecutable = 'java'
      if (platform === 'windows') {
        javaExecutable = 'java.exe'
      }
      let javaPath = `~/.qmlauncher/${serverUuid}/java/bin/${javaExecutable}`
      try {
        const javaPathResult = await wailsAPI.getJavaPath(serverUuid)
        if (javaPathResult) {
          javaPath = javaPathResult
        }
      } catch (error) {
        console.warn('Error getting Java path:', error)
      }

      // Create launch args object with explicit field names matching Go struct
      const launchArgs = {
        ServerUuid: serverUuid,
        MinecraftVersion: minecraftVersion,
        JavaPath: javaPath,
        JVMArgs: jvmArgs,
        GameArgs: gameArgs,
        WorkingDirectory: modsDir,
        HWID: hwid || '',
        Username: 'Player',
        LauncherConfig: {} as Record<string, any>,
      }
      
      console.log('[Launch] Launch args:', {
        ServerUuid: launchArgs.ServerUuid,
        MinecraftVersion: launchArgs.MinecraftVersion,
        JavaPath: launchArgs.JavaPath,
        HWID: launchArgs.HWID,
      })
      logBackend('[Launch] Launch args', launchArgs)
      
      const launchResult = await wailsAPI.launchMinecraft(launchArgs)
      const lrAny = launchResult as any

      const launchSucceeded =
        launchResult?.Success === true ||
        lrAny?.success === true ||
        (!launchResult?.Success && !lrAny?.success && !launchResult?.Error && !lrAny?.error)

      if (!launchSucceeded) {
        const errText = launchResult?.Error || lrAny?.error || t('servers.launchErrorGeneric')
        toast.error(t('servers.launchError') || 'Ошибка запуска', {
          description: errText,
          id: launchToastId
        })
        return
      }

      // Minecraft launched successfully
      setRunningGameAccounts(prev => ({ ...prev, [server.id]: currentAccountId }))
      toast.success(t('servers.launchSuccess') || 'Minecraft запущен', { id: launchToastId, duration: 2000 })
      
      // Update installation status after successful launch
      const serverUuidCheck = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
      const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuidCheck)
      if (checkResult.installed || checkResult.hasClient) {
        setServers(prev => prev.map(s => 
          s.id === server.id ? { ...s, clientInstalled: true } : s
        ))
      }
    } catch (error) {
      console.error('Error launching:', error)
      toast.error(t('servers.launchErrorGeneric') || 'Ошибка запуска', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setIsLaunching(false)
    }
  }

  if (isLoading) {
    return (
      <div className="p-8 h-full overflow-auto hide-scrollbar">
        {/* Header Skeleton */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <Skeleton className="h-9 w-48 mb-2" />
            <Skeleton className="h-5 w-32" />
          </div>
          <Skeleton className="h-10 w-10 rounded-md" />
        </div>

        {/* Servers Grid Skeleton */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-6">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="rounded-lg border bg-card">
              {/* Image Skeleton */}
              <Skeleton className="h-48 w-full rounded-t-lg" />
              
              {/* Content Skeleton */}
              <div className="p-6 space-y-4">
                <div className="space-y-2">
                  <Skeleton className="h-6 w-3/4" />
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-2/3" />
                </div>
                
                {/* Badges Skeleton */}
                <div className="flex gap-2">
                  <Skeleton className="h-5 w-16 rounded-full" />
                  <Skeleton className="h-5 w-20 rounded-full" />
                </div>
                
                {/* Stats Skeleton */}
                <div className="flex items-center gap-4 pt-2">
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-4 w-24" />
                </div>
                
                {/* Buttons Skeleton */}
                <div className="flex gap-2 pt-4">
                  <Skeleton className="h-10 flex-1" />
                  <Skeleton className="h-10 w-10" />
                  <Skeleton className="h-10 w-10" />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 h-full overflow-auto hide-scrollbar">
      {/* Header */}
      <div className="flex items-center justify-between mb-6 no-drag">
        <div className="no-drag">
          <h2 className="text-3xl font-bold mb-2 text-foreground">
            {t('servers.title')}
          </h2>
          {servers.length > 1 && (
            <p className="text-sm text-muted-foreground">
              {servers.length} {servers.length < 5 ? t('servers.servers2') : t('servers.servers5')}
            </p>
          )}
        </div>
        
        <div className="flex items-center gap-3">
          {/* Search Input - показываем только если серверов больше 2 */}
          {servers.length > 2 && (
            <div className="relative w-64">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                type="text"
                placeholder={t('servers.search') || 'Поиск серверов...'}
                value={searchQuery}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchQuery(e.target.value)}
                className="pl-10 no-drag"
              />
            </div>
          )}
          
          <Button
            variant="outline"
            size="icon"
            onClick={async () => {
              await loadServers()
              // Re-check installation status after refresh
              setTimeout(async () => {
                const currentServers = servers
                if (currentServers.length > 0) {
                  for (const server of currentServers) {
                    const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
                    try {
                      const checkResult = await wailsAPI.checkClientInstalled(server.id, serverUuid)
                      // ClientCheckResult doesn't have Success field, check Installed or HasClient directly
                      if (checkResult.installed || checkResult.hasClient) {
                        setServers(prev => prev.map(s => {
                          if (s.id === server.id) {
                            // Don't overwrite true with false
                            const isInstalled = checkResult.installed || checkResult.hasClient
                            const currentInstalled = s.clientInstalled
                            if (isInstalled || currentInstalled === undefined) {
                              return { ...s, clientInstalled: isInstalled }
                            }
                            return s
                          }
                          return s
                        }))
                      }
                    } catch (error) {
                      console.error(`Error checking installation for server ${server.id}:`, error)
                    }
                  }
                }
              }, 500)
            }}
            disabled={isLoading}
            className="no-drag"
            title={t('servers.refresh')}
          >
            <RefreshCw className={cn("w-5 h-5", isLoading && "animate-spin")} />
          </Button>
        </div>
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
        <div className={`grid grid-cols-1 md:grid-cols-2 gap-6 no-drag ${
          servers.filter(server => {
            if (!debouncedSearchQuery) return true
            const query = debouncedSearchQuery.toLowerCase()
            return (
              server.name?.toLowerCase().includes(query) ||
              server.server_name?.toLowerCase().includes(query) ||
              server.description?.toLowerCase().includes(query) ||
              server.minecraft_version?.toLowerCase().includes(query) ||
              server.server_address?.toLowerCase().includes(query)
            )
          }).length > 2 ? 'overflow-y-auto hide-scrollbar' : ''
        }`}>
          {servers
            .filter(server => {
              if (!debouncedSearchQuery) return true
              const query = debouncedSearchQuery.toLowerCase()
              return (
                server.name?.toLowerCase().includes(query) ||
                server.server_name?.toLowerCase().includes(query) ||
                server.description?.toLowerCase().includes(query) ||
                server.minecraft_version?.toLowerCase().includes(query) ||
                server.server_address?.toLowerCase().includes(query)
              )
            })
            .map((server, index) => (
            <motion.div
              key={server.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: index * 0.05 }}
              whileHover={{ y: -4 }}
            >
              <Card className="group relative hover:shadow-lg transition-all cursor-pointer no-drag">
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
                  
                  {/* Градиент только снизу, не перекрывает верхний правый угол */}
                  <div className="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-background via-background/50 to-transparent z-0 pointer-events-none" />
                </div>
                
                {/* Status Badge - вне overflow-hidden контейнера, но внутри Card */}
                <div className="absolute right-4 z-50" style={{ position: 'absolute', top: '2rem', right: '1rem', zIndex: 50 }}>
                  {(() => {
                    const status = server.server_status || "unknown"
                    if (status === "online") {
                      return (
                        <Badge variant="default" className="bg-green-500 text-white flex items-center gap-1.5 shadow-lg border-2 border-green-400/50" style={{ backgroundColor: 'rgba(34, 197, 94, 0.95)' }}>
                          <span className="w-2 h-2 bg-white rounded-full animate-pulse" />
                          {t('servers.serverOnline') || 'Онлайн'}
                        </Badge>
                      )
                    } else if (status === "offline") {
                      return (
                        <Badge variant="destructive" className="bg-red-500 text-white shadow-lg border-2 border-red-400/50" style={{ backgroundColor: 'rgba(239, 68, 68, 0.95)' }}>
                          {t('servers.serverOffline') || 'Оффлайн'}
                        </Badge>
                      )
                    } else {
                      return (
                        <Badge variant="secondary" className="bg-gray-500 text-white shadow-lg border-2 border-gray-400/50" style={{ backgroundColor: 'rgba(107, 114, 128, 0.95)' }}>
                          {t('servers.statusUnknown') || 'Неизвестно'}
                        </Badge>
                      )
                    }
                  })()}
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
                        value={selectedGameAccounts[server.id]?.toString() || gameAccounts.filter(acc => acc.server_id === server.id)[0]?.id.toString() || ''}
                        onValueChange={(value: string) => {
                          setSelectedGameAccounts(prev => ({
                            ...prev,
                            [server.id]: parseInt(value)
                          }))
                        }}
                      >
                        <SelectTrigger id={`game-account-${server.id}`} className="w-full no-drag">
                          <SelectValue placeholder={t('servers.selectGameAccount')} />
                        </SelectTrigger>
                        <SelectContent className="no-drag">
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
                  <div className="flex gap-2 w-full justify-end">
                    {isClientInstalled(server) ? (
                      <Button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleLaunch(server)
                        }}
                        disabled={
                          isLaunching ||
                          server.server_status === "offline" ||
                          (authToken && gameAccounts.filter(acc => acc.server_id === server.id).length > 0 && !selectedGameAccounts[server.id]) ||
                          runningGameAccounts[server.id] === (selectedGameAccounts[server.id] ? String(selectedGameAccounts[server.id]) : 'default') ||
                          false
                        }
                        className="flex-1"
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
                      <Button
                        onClick={(e) => {
                          e.stopPropagation()
                          handleInstall(server)
                        }}
                        disabled={installingServerId === server.id}
                        className="flex-1"
                        size="sm"
                      >
                        {installingServerId === server.id ? (
                          <>
                            <Loader2 className="w-4 h-4 animate-spin mr-2" />
                            <span className="truncate text-xs">
                              {t('servers.installingButton') || 'Установка'}
                            </span>
                          </>
                        ) : (
                          <>
                            <Download className="w-4 h-4 mr-2" />
                            {t('servers.install')}
                          </>
                        )}
                      </Button>
                    )}
                    
                    {isClientInstalled(server) && (
                      <Button
                        variant="destructive"
                        size="icon"
                        onClick={(e) => {
                          e.stopPropagation()
                          setServerToUninstall(server)
                          setUninstallDialogOpen(true)
                        }}
                        title={t('servers.uninstall') || 'Удалить игру'}
                        className="no-drag"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    )}
                    
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={(e) => {
                        e.stopPropagation()
                        setSelectedServerForInfo(server)
                      }}
                      title={t('servers.info')}
                      className="no-drag"
                    >
                      <Info className="w-4 h-4" />
                    </Button>
                    
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={(e) => {
                        e.stopPropagation()
                        setSelectedServerForSettings(server)
                      }}
                      title={t('servers.clientSettings')}
                      className="no-drag"
                    >
                      <Settings className="w-4 h-4" />
                    </Button>
                  </div>
                  {installingServerId === server.id && installationProgress && (
                    <div className="w-full space-y-1 mt-2">
                      <Progress value={installationProgress.progress} className="h-1.5" />
                      <p className="text-xs text-muted-foreground text-center">
                        {installationProgress.stage}
                      </p>
                    </div>
                  )}
                </CardFooter>
              </Card>
            </motion.div>
          ))}
        </div>
      )}

      {/* Server Info Dialog */}
      <Dialog open={!!selectedServerForInfo} onOpenChange={(open) => !open && setSelectedServerForInfo(null)}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Info className="w-5 h-5" />
              {t('servers.info')}: {selectedServerForInfo?.name}
            </DialogTitle>
            <DialogDescription>
              {t('servers.server')} ID: {selectedServerForInfo?.id}
            </DialogDescription>
          </DialogHeader>
          
          {selectedServerForInfo && (
            <div className="space-y-4 mt-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-sm font-semibold text-muted-foreground">
                    {t('servers.serverName') || 'Server Name'}
                  </Label>
                  <p className="text-sm text-foreground mt-1">{selectedServerForInfo.name}</p>
                </div>
                
                <div>
                  <Label className="text-sm font-semibold text-muted-foreground">
                    {t('servers.server') || 'Server ID'}
                  </Label>
                  <p className="text-sm text-foreground mt-1">#{selectedServerForInfo.id}</p>
                </div>
                
                {selectedServerForInfo.description && (
                  <div className="col-span-2">
                    <Label className="text-sm font-semibold text-muted-foreground">
                      {t('servers.description') || 'Description'}
                    </Label>
                    <p className="text-sm text-foreground mt-1">{selectedServerForInfo.description}</p>
                  </div>
                )}
                
                {selectedServerForInfo.server_address && (
                  <div>
                    <Label className="text-sm font-semibold text-muted-foreground">
                      {t('servers.serverAddress') || 'Server Address'}
                    </Label>
                    <p className="text-sm text-foreground mt-1">{selectedServerForInfo.server_address}</p>
                  </div>
                )}
                
                <div>
                  <Label className="text-sm font-semibold text-muted-foreground">
                    {t('servers.serverPort') || 'Server Port'}
                  </Label>
                  <p className="text-sm text-foreground mt-1">{selectedServerForInfo.server_port}</p>
                </div>
                
                <div>
                  <Label className="text-sm font-semibold text-muted-foreground">
                    {t('servers.minecraftVersion') || 'Minecraft Version'}
                  </Label>
                  <p className="text-sm text-foreground mt-1">{selectedServerForInfo.minecraft_version}</p>
                </div>
                
                <div>
                  <Label className="text-sm font-semibold text-muted-foreground">
                    {t('servers.loaderType') || 'Loader Type'}
                  </Label>
                  <div className="flex items-center gap-2 mt-1">
                    {(() => {
                      const loaderType = selectedServerForInfo.loader_enabled 
                        ? (selectedServerForInfo.loader_type || 'vanilla')
                        : 'vanilla'
                      const loaderName = loaderType === 'forge' ? 'Forge' 
                        : loaderType === 'fabric' ? 'Fabric'
                        : loaderType === 'quilt' ? 'Quilt'
                        : loaderType === 'neoforge' ? 'NeoForge'
                        : 'Vanilla'
                      const loaderIcon = loaderType === 'forge' ? forgeIcon
                        : loaderType === 'fabric' ? fabricIcon
                        : loaderType === 'quilt' ? quiltIcon
                        : loaderType === 'neoforge' ? neoforgeIcon
                        : vanillaIcon
                      return (
                        <>
                          <img src={loaderIcon} alt={loaderName} className="w-4 h-4" />
                          <p className="text-sm text-foreground">{loaderName}</p>
                        </>
                      )
                    })()}
                  </div>
                </div>
                
                {selectedServerForInfo.loader_version && (
                  <div>
                    <Label className="text-sm font-semibold text-muted-foreground">
                      {t('servers.loaderVersion') || 'Loader Version'}
                    </Label>
                    <p className="text-sm text-foreground mt-1">{selectedServerForInfo.loader_version}</p>
                  </div>
                )}
                
                {selectedServerForInfo.server_status && (
                  <div>
                    <Label className="text-sm font-semibold text-muted-foreground">
                      {t('servers.serverStatus') || 'Server Status'}
                    </Label>
                    <div className="mt-1">
                      <Badge 
                        variant={selectedServerForInfo.server_status === 'online' ? 'default' : 'secondary'}
                      >
                        {selectedServerForInfo.server_status === 'online' 
                          ? (t('servers.serverOnline') || 'Online')
                          : selectedServerForInfo.server_status === 'offline'
                          ? (t('servers.serverOffline') || 'Offline')
                          : (t('servers.statusUnknown') || 'Unknown')}
                      </Badge>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Server Settings Dialog */}
      <ServerSettingsDialog
        server={selectedServerForSettings}
        isOpen={!!selectedServerForSettings}
        onClose={() => setSelectedServerForSettings(null)}
      />

      {/* Uninstall Confirmation Dialog */}
      <Dialog open={uninstallDialogOpen} onOpenChange={setUninstallDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('servers.uninstall') || 'Удалить игру'}</DialogTitle>
            <DialogDescription>
              {serverToUninstall && (t('servers.confirmUninstall', { name: serverToUninstall.name }) || `Удалить игру для сервера ${serverToUninstall.name}?`)}
            </DialogDescription>
          </DialogHeader>
          {uninstallProgress && (
            <div className="space-y-2">
              <Progress value={uninstallProgress.progress} className="h-2" />
              <p className="text-sm text-muted-foreground">{uninstallProgress.stage}</p>
            </div>
          )}
          <div className="flex justify-end gap-2 mt-4">
            <Button
              variant="outline"
              onClick={() => {
                setUninstallDialogOpen(false)
                setServerToUninstall(null)
                setUninstallProgress(null)
              }}
              disabled={uninstallingServerId !== null}
            >
              {t('common.cancel') || 'Отмена'}
            </Button>
            <Button
              variant="destructive"
              onClick={async () => {
                if (!serverToUninstall) return
                setUninstallingServerId(serverToUninstall.id)
                setUninstallProgress({ stage: t('servers.uninstalling') || 'Удаление...', progress: 0 })
                try {
                  setUninstallProgress({ stage: t('servers.uninstalling') || 'Удаление...', progress: 50 })
                  const result = await wailsAPI.uninstallMinecraft(serverToUninstall.id)
                  if (result.success) {
                    setUninstallProgress({ stage: t('servers.uninstallComplete') || 'Удаление завершено', progress: 100 })
                    setServers(prev => prev.map(s => 
                      s.id === serverToUninstall.id ? { ...s, clientInstalled: false } : s
                    ))
                    setTimeout(() => {
                      setUninstallDialogOpen(false)
                      setServerToUninstall(null)
                      setUninstallProgress(null)
                      toast.success(t('servers.uninstallComplete'))
                    }, 500)
                  } else {
                    toast.error(t('servers.uninstallError'), {
                      description: result.error || 'Неизвестная ошибка'
                    })
                    setUninstallDialogOpen(false)
                    setServerToUninstall(null)
                    setUninstallProgress(null)
                  }
                } catch (error) {
                  console.error('Error uninstalling:', error)
                  toast.error(t('servers.uninstallError'), {
                    description: error instanceof Error ? error.message : String(error)
                  })
                  setUninstallDialogOpen(false)
                  setServerToUninstall(null)
                  setUninstallProgress(null)
                } finally {
                  setUninstallingServerId(null)
                }
              }}
              disabled={uninstallingServerId !== null}
            >
              {uninstallingServerId !== null ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  {t('servers.uninstalling') || 'Удаление...'}
                </>
              ) : (
                t('servers.uninstall') || 'Удалить'
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
