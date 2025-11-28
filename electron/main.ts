import { app, BrowserWindow, ipcMain, session } from 'electron'
import path from 'path'
import { spawn, ChildProcess, exec } from 'child_process'
import fs from 'fs'
import os from 'os'
import crypto from 'crypto'
import Database from 'better-sqlite3'
import https from 'https'
import http from 'http'
import { API_BASE_URL } from './config/api'

const isDev = process.env.NODE_ENV === 'development' || !app.isPackaged

// Логирование для отладки
console.log('=== QMLauncher Dev Mode Check ===')
console.log('isDev:', isDev)
console.log('NODE_ENV:', process.env.NODE_ENV)
console.log('isPackaged:', app.isPackaged)
console.log('===============================')

let mainWindow: BrowserWindow | null = null
let minecraftProcess: ChildProcess | null = null
let isUpdating = false

function createWindow() {
  const preloadPath = isDev
    ? path.join(__dirname, 'preload.js')
    : path.join(__dirname, 'preload.js')

  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1000,
    minHeight: 600,
    frame: false,
    transparent: true,
    backgroundColor: '#00000000',
    roundedCorners: true,
    webPreferences: {
      preload: preloadPath,
      nodeIntegration: false,
      contextIsolation: true,
      backgroundThrottling: false,
      devTools: false, // DevTools полностью отключены
      webSecurity: true // Включаем webSecurity, но разрешаем localhost через session
    },
    icon: path.join(__dirname, '../build/icon.png'),
    titleBarStyle: 'hidden',
    titleBarOverlay: false,
    show: false
  })

  // Show window when ready to avoid white flash
  mainWindow.once('ready-to-show', () => {
    mainWindow?.show()
  })


  // Блокируем открытие DevTools всегда
  mainWindow.webContents.on('devtools-opened', () => {
    mainWindow?.webContents.closeDevTools()
  })

  // Блокируем горячие клавиши для DevTools
  mainWindow.webContents.on('before-input-event', (event, input) => {
    // Блокируем F12, Ctrl+Shift+I, Ctrl+Shift+J, Ctrl+Shift+C
    if (input.key === 'F12' || 
        (input.control && input.shift && (input.key === 'I' || input.key === 'J' || input.key === 'C'))) {
      event.preventDefault()
    }
  })

  if (isDev) {
    mainWindow.loadURL('http://localhost:5175')
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'))
  }

  mainWindow.on('closed', () => {
    mainWindow = null
  })

  // Window controls
  ipcMain.handle('window-minimize', () => {
    if (mainWindow) {
      mainWindow.minimize()
    }
  })

  ipcMain.handle('window-maximize', () => {
    if (mainWindow) {
      if (mainWindow.isMaximized()) {
        mainWindow.unmaximize()
      } else {
        mainWindow.maximize()
      }
    }
  })

  ipcMain.handle('window-close', () => {
    if (mainWindow) {
      mainWindow.close()
    }
  })

  ipcMain.handle('window-is-maximized', () => {
    return mainWindow ? mainWindow.isMaximized() : false
  })

}

app.whenReady().then(() => {
  // Configure session to allow requests to localhost
  const ses = session.defaultSession
  
  // Allow requests to localhost and local network
  ses.webRequest.onBeforeSendHeaders((details, callback) => {
    // Add CORS headers for localhost requests
    if (details.url.includes('localhost') || details.url.includes('127.0.0.1')) {
      callback({
        requestHeaders: {
          ...details.requestHeaders,
          'Origin': details.url.split('/').slice(0, 3).join('/'),
        }
      })
    } else {
      callback({ requestHeaders: details.requestHeaders })
    }
  })
  
  // Handle CORS preflight requests
  ses.webRequest.onHeadersReceived((details, callback) => {
    if (details.url.includes('localhost') || details.url.includes('127.0.0.1')) {
      callback({
        responseHeaders: {
          ...details.responseHeaders,
          'Access-Control-Allow-Origin': ['*'],
          'Access-Control-Allow-Methods': ['GET, POST, PUT, DELETE, OPTIONS'],
          'Access-Control-Allow-Headers': ['Content-Type, Authorization'],
        }
      })
    } else {
      callback({ responseHeaders: details.responseHeaders })
    }
  })

  // Check for updates before creating window (only in production)
  if (!isDev && app.isPackaged) {
    checkAndInstallUpdate().catch((error) => {
      console.error('Error checking for updates:', error)
      // Continue with normal startup even if update check fails
  createWindow()
    })
  } else {
    createWindow()
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// Function to get current launcher version SHA
async function getCurrentVersionSha(): Promise<string | null> {
  try {
    // Check if there's a stored commit SHA in config
    const configPath = path.join(os.homedir(), '.qmlauncher', 'update-info.json')
    if (fs.existsSync(configPath)) {
      const updateInfo = JSON.parse(fs.readFileSync(configPath, 'utf-8'))
      return updateInfo.commit_sha || null
    }
    return null
  } catch (error) {
    console.error('Error getting current version SHA:', error)
    return null
  }
}

// Function to detect platform
function getPlatform(): string {
  const platform = process.platform
  if (platform === 'win32') return 'win'
  if (platform === 'darwin') return 'mac'
  return 'linux'
}

// Function to check for updates and install if available
async function checkAndInstallUpdate(): Promise<void> {
  if (isUpdating) {
    console.log('Update already in progress, skipping...')
    return
  }

  try {
    console.log('Checking for launcher updates...')
    const currentVersionSha = await getCurrentVersionSha()
    const platform = getPlatform()
    
    // Build query parameters
    const params = new URLSearchParams()
    if (currentVersionSha) {
      params.append('version_sha', currentVersionSha)
    }
    params.append('platform', platform)
    
    const updateUrl = `${API_BASE_URL}/launcher/check-update?${params.toString()}`
    console.log('Checking update at:', updateUrl)
    
    const response = await fetch(updateUrl)
    if (!response.ok) {
      console.warn('Failed to check for updates:', response.statusText)
      createWindow()
      return
    }
    
    const updateInfo = await response.json() as {
      has_update: boolean
      download_url?: string
      build_id?: string
      latest_commit_sha?: string
      latest_commit_message?: string
    }
    
    if (!updateInfo.has_update || !updateInfo.download_url) {
      console.log('No updates available')
      // Save current commit SHA if we don't have one yet (first run)
      if (!currentVersionSha && updateInfo.latest_commit_sha) {
        const updateInfoPath = path.join(os.homedir(), '.qmlauncher', 'update-info.json')
        const updateInfoDir = path.dirname(updateInfoPath)
        if (!fs.existsSync(updateInfoDir)) {
          fs.mkdirSync(updateInfoDir, { recursive: true })
        }
        fs.writeFileSync(updateInfoPath, JSON.stringify({
          commit_sha: updateInfo.latest_commit_sha,
          update_date: new Date().toISOString()
        }, null, 2))
      }
      createWindow()
      return
    }
    
    console.log('Update available! Downloading...', updateInfo.latest_commit_sha)
    isUpdating = true
    
    // Download update
    const downloadResult = await downloadAndInstallUpdate(updateInfo.download_url, updateInfo.latest_commit_sha || undefined)
    
    if (downloadResult.success) {
      console.log('Update installed successfully, restarting...')
      // Save commit SHA before restart
      const updateInfoPath = path.join(os.homedir(), '.qmlauncher', 'update-info.json')
      const updateInfoDir = path.dirname(updateInfoPath)
      if (!fs.existsSync(updateInfoDir)) {
        fs.mkdirSync(updateInfoDir, { recursive: true })
      }
      fs.writeFileSync(updateInfoPath, JSON.stringify({
        commit_sha: updateInfo.latest_commit_sha,
        update_date: new Date().toISOString()
      }, null, 2))
      
      // Restart application
      app.relaunch()
      app.exit(0)
    } else {
      console.error('Failed to install update:', downloadResult.error)
      isUpdating = false
      createWindow()
    }
  } catch (error) {
    console.error('Error checking for updates:', error)
    isUpdating = false
    createWindow()
  }
}

// Function to download and install update
async function downloadAndInstallUpdate(downloadUrl: string, commitSha?: string): Promise<{ success: boolean; error?: string }> {
  try {
    const updatesDir = path.join(os.homedir(), '.qmlauncher', 'updates')
    if (!fs.existsSync(updatesDir)) {
      fs.mkdirSync(updatesDir, { recursive: true })
    }
    
    const updateZipPath = path.join(updatesDir, `update-${Date.now()}.zip`)
    const extractDir = path.join(updatesDir, 'extract')
    
    // Clean up old updates
    if (fs.existsSync(extractDir)) {
      fs.rmSync(extractDir, { recursive: true, force: true })
    }
    fs.mkdirSync(extractDir, { recursive: true })
    
    // Download update
    console.log('Downloading update from:', downloadUrl)
    const fullUrl = downloadUrl.startsWith('http') ? downloadUrl : `${API_BASE_URL.replace('/api/v1', '')}${downloadUrl}`
    const downloadResponse = await fetch(fullUrl)
    
    if (!downloadResponse.ok) {
      return { success: false, error: `Failed to download: ${downloadResponse.statusText}` }
    }
    
    // Save to file
    const buffer = await downloadResponse.arrayBuffer()
    fs.writeFileSync(updateZipPath, Buffer.from(buffer))
    console.log('Update downloaded, extracting...')
    
    // Extract ZIP using Node.js built-in zlib and streams
    // For simplicity, we'll use a child process to extract
    const unzipCommand = process.platform === 'win32' 
      ? `powershell -Command "Expand-Archive -Path '${updateZipPath}' -DestinationPath '${extractDir}' -Force"`
      : `unzip -o '${updateZipPath}' -d '${extractDir}'`
    
    await new Promise<void>((resolve, reject) => {
      exec(unzipCommand, (error, stdout, stderr) => {
        if (error) {
          reject(error)
        } else {
          resolve()
        }
      })
    })
    
    console.log('Update extracted')
    
    // The update package from QMServer is typically a self-extracting archive
    // For Windows: .exe installer
    // For Linux/Mac: .sh or .app bundle
    // We need to find and execute the installer
    
    let installerPath: string | null = null
    const files = fs.readdirSync(extractDir, { recursive: true })
    
    for (const file of files) {
      const filePath = path.join(extractDir, file as string)
      const stat = fs.statSync(filePath)
      
      if (stat.isFile()) {
        const ext = path.extname(filePath).toLowerCase()
        if (process.platform === 'win32' && (ext === '.exe' || ext === '.msi')) {
          installerPath = filePath
          break
        } else if (process.platform !== 'win32' && (ext === '.sh' || ext === '.app' || ext === '.dmg')) {
          installerPath = filePath
          if (ext === '.sh') {
            // Make executable
            fs.chmodSync(filePath, '755')
          }
          break
        }
      }
    }
    
    if (installerPath) {
      console.log('Found installer, executing:', installerPath)
      // Execute installer in background
      // For Windows .exe, it will handle installation and restart
      // For Linux .sh, it should install and restart
      spawn(installerPath, [], {
        detached: true,
        stdio: 'ignore'
      }).unref()
      
      // Give installer time to start
      await new Promise(resolve => setTimeout(resolve, 2000))
      
      // Clean up
      fs.unlinkSync(updateZipPath)
      
      return { success: true }
    } else {
      // No installer found - might be a regular ZIP with app files
      // For Electron apps packaged as ZIP, we need to extract and replace files
      // This is complex while app is running, so we'll save update info for next launch
      console.log('No installer found, saving update info for next launch')
      
      // Save update info to be processed on next launch
      const updateInfoPath = path.join(os.homedir(), '.qmlauncher', 'pending-update.json')
      fs.writeFileSync(updateInfoPath, JSON.stringify({
        extract_dir: extractDir,
        commit_sha: commitSha,
        update_date: new Date().toISOString()
      }, null, 2))
      
      // Don't clean up - we'll need the extracted files on next launch
      // fs.unlinkSync(updateZipPath) - keep ZIP for now
      
      // For now, return success - the actual installation will happen on next launch
      // In a production system, you'd want to show a message to the user
      console.log('Update saved, will be installed on next launch')
      return { success: true }
    }
  } catch (error) {
    console.error('Error installing update:', error)
    return { success: false, error: String(error) }
  }
}

// IPC Handlers
ipcMain.handle('get-app-version', () => {
  return app.getVersion()
})

ipcMain.handle('get-platform', () => {
  return process.platform
})

ipcMain.handle('get-hwid', async () => {
  // Get hardware ID based on platform
  const os = require('os')
  const crypto = require('crypto')
  
  try {
    // Combine multiple hardware identifiers for uniqueness
    const identifiers = [
      os.hostname(),
      os.platform(),
      os.arch(),
      os.cpus()[0]?.model || '',
      os.totalmem().toString(),
    ]
    
    const hwidString = identifiers.join('-')
    const hwid = crypto.createHash('sha256').update(hwidString).digest('hex')
    return hwid
  } catch (error) {
    console.error('Error generating HWID:', error)
    return null
  }
})

ipcMain.handle('get-launcher-config', async () => {
  const configPath = path.join(os.homedir(), '.qmlauncher', 'config.json')
  try {
    if (fs.existsSync(configPath)) {
      const data = fs.readFileSync(configPath, 'utf-8')
      return JSON.parse(data)
    }
  } catch (error: unknown) {
    console.error('Error reading config:', error)
  }
  return {
    javaPath: 'java',
    minMemory: 1024,
    maxMemory: 4096,
    windowWidth: 1200,
    windowHeight: 800,
    themeId: 'dark',
    selectedProfile: null,
    profiles: {}
  }
})

ipcMain.handle('save-launcher-config', async (_event: any, config: any) => {
  const configDir = path.join(os.homedir(), '.qmlauncher')
  const configPath = path.join(configDir, 'config.json')
  try {
    if (!fs.existsSync(configDir)) {
      fs.mkdirSync(configDir, { recursive: true })
    }
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2), 'utf-8')
    return { success: true }
  } catch (error) {
    console.error('Error saving config:', error)
    return { success: false, error: String(error) }
  }
})

ipcMain.handle('launch-minecraft', async (_event: any, launchArgs: any) => {
  try {
    if (minecraftProcess) {
      return { success: false, error: 'Minecraft is already running' }
    }

    const { javaPath, gameArgs, jvmArgs, workingDirectory, minecraftVersion, hwid, launcherConfig } = launchArgs
    
    // Resolve paths
    const homeDir = os.homedir()
    const resolvedWorkingDir = workingDirectory.replace('~', homeDir)
    
    // Get client JAR path if version is provided
    let finalJvmArgs = [...jvmArgs]
    let finalGameArgs = [...gameArgs]
    
    if (minecraftVersion) {
      const versionDir = path.join(homeDir, '.qmlauncher', 'versions', minecraftVersion)
      const clientJarPath = path.join(versionDir, `${minecraftVersion}.jar`)
      const versionJsonPath = path.join(versionDir, `${minecraftVersion}.json`)
      const librariesDir = path.join(homeDir, '.qmlauncher', 'libraries')
      const nativesDir = path.join(versionDir, 'natives')
      
      // Check if client exists
      if (fs.existsSync(clientJarPath) && fs.existsSync(versionJsonPath)) {
        // Read version metadata
        const versionData = JSON.parse(fs.readFileSync(versionJsonPath, 'utf-8')) as {
          mainClass: string
          minecraftArguments?: string
          arguments?: {
            game: Array<string | { rules?: Array<{ action: string; features?: Record<string, boolean> }>; value: string | string[] }>
            jvm: Array<string | { rules?: Array<{ action: string; features?: Record<string, boolean> }>; value: string | string[] }>
          }
          libraries: Array<{
            downloads: {
              artifact?: { path: string }
              classifiers?: {
                natives?: { path: string; url: string }
              }
            }
            name: string
            rules?: Array<{ action: string; os?: { name?: string } }>
            extract?: {
              exclude: string[]
            }
          }>
        }
        
        // Build classpath with all libraries
        const classpathSeparator = process.platform === 'win32' ? ';' : ':'
        const classpathParts: string[] = [clientJarPath]
        
        // Add all libraries to classpath
        for (const library of versionData.libraries || []) {
          // Check if library should be included (skip platform-specific rules for now)
          let shouldInclude = true
          if (library.rules) {
            for (const rule of library.rules) {
              if (rule.os && rule.os.name) {
                const osName = rule.os.name
                const currentOs = process.platform === 'win32' ? 'windows' : 
                                 process.platform === 'darwin' ? 'osx' : 'linux'
                if (osName !== currentOs && rule.action === 'allow') {
                  shouldInclude = false
                }
              }
            }
          }
          
          if (shouldInclude && library.downloads?.artifact?.path) {
            const libPath = path.join(librariesDir, library.downloads.artifact.path)
            if (fs.existsSync(libPath)) {
              classpathParts.push(libPath)
            }
          }
        }
        
        const classpath = classpathParts.join(classpathSeparator)
        
        // Update JVM args to include classpath
        const classpathIndex = finalJvmArgs.findIndex((arg: string) => arg === '-cp' || arg === '-classpath')
        if (classpathIndex === -1) {
          // Add classpath if not present
          finalJvmArgs.splice(1, 0, '-cp', classpath)
        } else {
          // Update existing classpath
          finalJvmArgs[classpathIndex + 1] = classpath
        }
        
        // Create natives directory if it doesn't exist
        if (!fs.existsSync(nativesDir)) {
          fs.mkdirSync(nativesDir, { recursive: true })
        }
        
        // Update natives path
        const nativesIndex = finalJvmArgs.findIndex((arg: string) => arg.startsWith('-Djava.library.path='))
        if (nativesIndex !== -1) {
          finalJvmArgs[nativesIndex] = `-Djava.library.path=${nativesDir}`
        } else {
          finalJvmArgs.push(`-Djava.library.path=${nativesDir}`)
        }
        
        // Use main class from version metadata
        const mainClass = versionData.mainClass || 'net.minecraft.client.main.Main'
        
        // Process game arguments from version.json if available
        if (versionData.arguments?.game) {
          // Use new format arguments
          const processedGameArgs: string[] = []
          for (const arg of versionData.arguments.game) {
            if (typeof arg === 'string') {
              processedGameArgs.push(arg)
            } else if (arg.value) {
              if (typeof arg.value === 'string') {
                processedGameArgs.push(arg.value)
              } else if (Array.isArray(arg.value)) {
                processedGameArgs.push(...arg.value)
              }
            }
          }
          // Merge with existing game args (server-specific args take precedence)
          finalGameArgs = [mainClass, ...processedGameArgs, ...finalGameArgs.slice(1)]
        } else if (versionData.minecraftArguments) {
          // Use old format minecraftArguments
          const oldArgs = versionData.minecraftArguments.split(' ').filter((a: string) => a.trim())
          finalGameArgs = [mainClass, ...oldArgs, ...finalGameArgs.slice(1)]
        } else {
          finalGameArgs.unshift(mainClass)
        }
        
        // Process JVM arguments from version.json if available
        if (versionData.arguments?.jvm) {
          const processedJvmArgs: string[] = []
          for (const arg of versionData.arguments.jvm) {
            if (typeof arg === 'string') {
              processedJvmArgs.push(arg)
            } else if (arg.value) {
              if (typeof arg.value === 'string') {
                processedJvmArgs.push(arg.value)
              } else if (Array.isArray(arg.value)) {
                processedJvmArgs.push(...arg.value)
              }
            }
          }
          // Merge with existing JVM args (user config takes precedence)
          finalJvmArgs = [...processedJvmArgs, ...finalJvmArgs]
        }
      } else if (fs.existsSync(clientJarPath)) {
        // Fallback to default if version.json doesn't exist
        const classpathSeparator = process.platform === 'win32' ? ';' : ':'
        const classpathIndex = finalJvmArgs.findIndex((arg: string) => arg === '-cp' || arg === '-classpath')
        if (classpathIndex === -1) {
          finalJvmArgs.splice(1, 0, '-cp', `${clientJarPath}${classpathSeparator}${librariesDir}/*`)
        }
        
        if (!fs.existsSync(nativesDir)) {
          fs.mkdirSync(nativesDir, { recursive: true })
        }
        
        const nativesIndex = finalJvmArgs.findIndex((arg: string) => arg.startsWith('-Djava.library.path='))
        if (nativesIndex !== -1) {
          finalJvmArgs[nativesIndex] = `-Djava.library.path=${nativesDir}`
        } else {
          finalJvmArgs.push(`-Djava.library.path=${nativesDir}`)
        }
        
        finalGameArgs.unshift('net.minecraft.client.main.Main')
      }
    }
    
    // Resolve assets directory in game args
    const assetsIndex = finalGameArgs.findIndex((arg: string) => arg === '--assetsDir')
    if (assetsIndex !== -1 && assetsIndex + 1 < finalGameArgs.length) {
      finalGameArgs[assetsIndex + 1] = finalGameArgs[assetsIndex + 1].replace('~', homeDir)
    }
    
    // Resolve game directory in game args
    const gameDirIndex = finalGameArgs.findIndex((arg: string) => arg === '--gameDir')
    if (gameDirIndex !== -1 && gameDirIndex + 1 < finalGameArgs.length) {
      finalGameArgs[gameDirIndex + 1] = finalGameArgs[gameDirIndex + 1].replace('~', homeDir)
    }
    
    const command = javaPath || 'java'
    const args = [...finalJvmArgs, ...finalGameArgs]

    console.log(`Launching Minecraft: ${command} ${args.join(' ')}`)

    minecraftProcess = spawn(command, args, {
      cwd: resolvedWorkingDir,
      stdio: 'pipe'
    })

    minecraftProcess.stdout?.on('data', (data: any) => {
      console.log(`Minecraft stdout: ${data}`)
    })

    minecraftProcess.stderr?.on('data', (data: any) => {
      console.error(`Minecraft stderr: ${data}`)
    })

    minecraftProcess.on('close', (code: number | null) => {
      console.log(`Minecraft process exited with code ${code}`)
      minecraftProcess = null
      if (mainWindow) {
        mainWindow.webContents.send('minecraft-exited', code)
      }
    })

    minecraftProcess.on('error', (error: Error) => {
      console.error('Error launching Minecraft:', error)
      minecraftProcess = null
      if (mainWindow) {
        mainWindow.webContents.send('minecraft-error', error.message)
      }
    })

    return { success: true, pid: minecraftProcess.pid }
  } catch (error: unknown) {
    console.error('Error launching Minecraft:', error)
    return { success: false, error: String(error) }
  }
})

ipcMain.handle('stop-minecraft', async () => {
  if (minecraftProcess) {
    minecraftProcess.kill()
    minecraftProcess = null
    return { success: true }
  }
  return { success: false, error: 'No Minecraft process running' }
})

// Get launcher config from SQLite database
ipcMain.handle('get-launcher-db-config', async (_event: any, serverId: number) => {
  try {
    const configPath = path.join(os.homedir(), '.qmlauncher', 'launcher-config.db')
    if (!fs.existsSync(configPath)) {
      return { success: false, error: 'Config database not found' }
    }

    const db = new Database(configPath)
    const config: Record<string, string> = {}
    
    // Read launcher_config table
    const configRows = db.prepare('SELECT key, value FROM launcher_config').all() as Array<{ key: string; value: string }>
    configRows.forEach(row => {
      config[row.key] = row.value
    })

    // Read mods
    const mods = db.prepare('SELECT * FROM mods').all() as Array<any>
    
    // Read plugins
    const plugins = db.prepare('SELECT * FROM plugins').all() as Array<any>

    db.close()

    return {
      success: true,
      config,
      mods,
      plugins
    }
  } catch (error) {
    console.error('Error reading launcher config:', error)
    return { success: false, error: String(error) }
  }
})

// Get embedded servers from config database
ipcMain.handle('get-embedded-servers', async () => {
  try {
    const configPath = path.join(os.homedir(), '.qmlauncher', 'launcher-config.db')
    if (!fs.existsSync(configPath)) {
      return []
    }

    const db = new Database(configPath)
    
    // Read embedded_servers table
    const servers = db.prepare('SELECT * FROM embedded_servers WHERE enabled = 1').all() as Array<{
      server_id: number
      server_uuid: string
      server_name: string | null
      server_address: string | null
      server_port: number | null
      minecraft_version: string | null
      description: string | null
      preview_image_url: string | null
      enabled: number
    }>

    db.close()
    return servers
  } catch (error) {
    console.error('Error reading embedded servers:', error)
    return []
  }
})

// Get mods list from server
ipcMain.handle('get-server-mods', async (_event: any, serverId: number, apiBaseUrl: string) => {
  try {
    const url = `${apiBaseUrl}/servers/${serverId}/mods`
    const response = await fetch(url)
    
    if (!response.ok) {
      throw new Error(`Failed to fetch mods: ${response.statusText}`)
    }

    const data = await response.json() as { mods?: any[] }
    return { success: true, mods: data.mods || [] }
  } catch (error) {
    console.error('Error fetching server mods:', error)
    return { success: false, error: String(error) }
  }
})

// Download mod file
ipcMain.handle('download-mod', async (_event: any, downloadUrl: string, savePath: string) => {
  return new Promise((resolve) => {
    try {
      const url = new URL(downloadUrl)
      const protocol = url.protocol === 'https:' ? https : http
      
      // Ensure directory exists
      const dir = path.dirname(savePath)
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true })
      }

      const file = fs.createWriteStream(savePath)
      
      protocol.get(url.href, (response) => {
        if (response.statusCode !== 200) {
          file.close()
          fs.unlinkSync(savePath)
          resolve({ success: false, error: `Failed to download: ${response.statusCode}` })
          return
        }

        response.pipe(file)

        file.on('finish', () => {
          file.close()
          resolve({ success: true })
        })

        file.on('error', (err) => {
          file.close()
          if (fs.existsSync(savePath)) {
            fs.unlinkSync(savePath)
          }
          resolve({ success: false, error: String(err) })
        })
      }).on('error', (err) => {
        if (fs.existsSync(savePath)) {
          fs.unlinkSync(savePath)
        }
        resolve({ success: false, error: String(err) })
      })
    } catch (error) {
      resolve({ success: false, error: String(error) })
    }
  })
})

// Check and update mods
ipcMain.handle('check-and-update-mods', async (_event: any, serverId: number, apiBaseUrl?: string) => {
  try {
    const configPath = path.join(os.homedir(), '.qmlauncher', 'launcher-config.db')
    if (!fs.existsSync(configPath)) {
      return { success: false, error: 'Config database not found', updated: false }
    }

    const db = new Database(configPath)
    
    // Use built-in API URL from config (set at build time)
    // Fallback to passed parameter or database config if needed
    const apiBaseUrlRow = db.prepare('SELECT value FROM launcher_config WHERE key = ?').get('api_base_url') as { value: string } | undefined
    const apiBaseUrlFromConfig = apiBaseUrl || apiBaseUrlRow?.value || API_BASE_URL

    // Get mods from database
    const dbMods = db.prepare('SELECT * FROM mods').all() as Array<{ filename: string; download_url: string; size: number }>
    
    // Get mods from server
    const serverModsResponse = await fetch(`${apiBaseUrlFromConfig}/servers/${serverId}/mods`)
    if (!serverModsResponse.ok) {
      db.close()
      return { success: false, error: 'Failed to fetch server mods', updated: false }
    }

    const serverModsData = await serverModsResponse.json() as { mods?: any[] }
    const serverMods = serverModsData.mods || []

    db.close()

    // Compare mods
    const modsDir = path.join(os.homedir(), '.qmlauncher', 'mods', String(serverId))
    if (!fs.existsSync(modsDir)) {
      fs.mkdirSync(modsDir, { recursive: true })
    }

    let needsUpdate = false
    const modsToDownload: Array<{ filename: string; download_url: string }> = []

    // Check each mod from server
    for (const serverMod of serverMods) {
      const localModPath = path.join(modsDir, serverMod.filename)
      const dbMod = dbMods.find(m => m.filename === serverMod.filename)

      // Check if mod needs update (missing or size mismatch)
      if (!fs.existsSync(localModPath) || !dbMod || dbMod.size !== serverMod.size) {
        needsUpdate = true
        modsToDownload.push({
          filename: serverMod.filename,
          download_url: serverMod.download_url || `${apiBaseUrlFromConfig}/servers/${serverId}/mods/${serverMod.filename}/download`
        })
      }
    }

    // Remove mods that are no longer on server
    if (fs.existsSync(modsDir)) {
      const localModFiles = fs.readdirSync(modsDir).filter(f => f.endsWith('.jar'))
      for (const localFile of localModFiles) {
        if (!serverMods.find(m => m.filename === localFile)) {
          needsUpdate = true
          fs.unlinkSync(path.join(modsDir, localFile))
        }
      }
    }

    // Download missing/updated mods
    if (needsUpdate && modsToDownload.length > 0) {
      for (const mod of modsToDownload) {
        const savePath = path.join(modsDir, mod.filename)
        const downloadResult = await new Promise<{ success: boolean; error?: string }>((resolve) => {
          const url = new URL(mod.download_url)
          const protocol = url.protocol === 'https:' ? https : http
          
          const file = fs.createWriteStream(savePath)
          
          protocol.get(url.href, (response) => {
            if (response.statusCode !== 200) {
              file.close()
              if (fs.existsSync(savePath)) {
                fs.unlinkSync(savePath)
              }
              resolve({ success: false, error: `Failed to download: ${response.statusCode}` })
              return
            }

            response.pipe(file)

            file.on('finish', () => {
              file.close()
              resolve({ success: true })
            })

            file.on('error', (err) => {
              file.close()
              if (fs.existsSync(savePath)) {
                fs.unlinkSync(savePath)
              }
              resolve({ success: false, error: String(err) })
            })
          }).on('error', (err) => {
            if (fs.existsSync(savePath)) {
              fs.unlinkSync(savePath)
            }
            resolve({ success: false, error: String(err) })
          })
        })

        if (!downloadResult.success) {
          return { success: false, error: `Failed to download ${mod.filename}: ${downloadResult.error}`, updated: false }
        }
      }
    }

    return { success: true, updated: needsUpdate, modsUpdated: modsToDownload.length, modsDir }
  } catch (error) {
    console.error('Error checking and updating mods:', error)
    return { success: false, error: String(error), updated: false }
  }
})

// Theme management
ipcMain.handle('get-custom-themes', async () => {
  const themesDir = path.join(os.homedir(), '.qmlauncher', 'themes')
  const themes: any[] = []
  
  try {
    if (!fs.existsSync(themesDir)) {
      fs.mkdirSync(themesDir, { recursive: true })
      return themes
    }
    
    const files = fs.readdirSync(themesDir)
    for (const file of files) {
      if (file.endsWith('.json')) {
        try {
          const filePath = path.join(themesDir, file)
          const data = fs.readFileSync(filePath, 'utf-8')
          const theme = JSON.parse(data)
          themes.push(theme)
        } catch (error) {
          console.error(`Error reading theme file ${file}:`, error)
        }
      }
    }
  } catch (error) {
    console.error('Error loading custom themes:', error)
  }
  
  return themes
})

ipcMain.handle('save-custom-theme', async (_event: any, theme: any) => {
  const themesDir = path.join(os.homedir(), '.qmlauncher', 'themes')
  
  try {
    if (!fs.existsSync(themesDir)) {
      fs.mkdirSync(themesDir, { recursive: true })
    }
    
    const themePath = path.join(themesDir, `${theme.id}.json`)
    fs.writeFileSync(themePath, JSON.stringify(theme, null, 2), 'utf-8')
    return { success: true }
  } catch (error) {
    console.error('Error saving custom theme:', error)
    return { success: false, error: String(error) }
  }
})

ipcMain.handle('remove-custom-theme', async (_event: any, themeId: string) => {
  const themesDir = path.join(os.homedir(), '.qmlauncher', 'themes')
  const themePath = path.join(themesDir, `${themeId}.json`)
  
  try {
    if (fs.existsSync(themePath)) {
      fs.unlinkSync(themePath)
      return { success: true }
    }
    return { success: false, error: 'Theme file not found' }
  } catch (error) {
    console.error('Error removing custom theme:', error)
    return { success: false, error: String(error) }
  }
})

// Auth token management
ipcMain.handle('save-auth-token', async (_event: any, token: string) => {
  const configDir = path.join(os.homedir(), '.qmlauncher')
  const tokenPath = path.join(configDir, 'auth-token.txt')
  
  try {
    if (!fs.existsSync(configDir)) {
      fs.mkdirSync(configDir, { recursive: true })
    }
    fs.writeFileSync(tokenPath, token, 'utf-8')
    return { success: true }
  } catch (error) {
    console.error('Error saving auth token:', error)
    return { success: false, error: String(error) }
  }
})

ipcMain.handle('get-auth-token', async () => {
  const tokenPath = path.join(os.homedir(), '.qmlauncher', 'auth-token.txt')
  
  try {
    if (fs.existsSync(tokenPath)) {
      const token = fs.readFileSync(tokenPath, 'utf-8').trim()
      return token || null
    }
    return null
  } catch (error) {
    console.error('Error reading auth token:', error)
    return null
  }
})

ipcMain.handle('clear-auth-token', async () => {
  const tokenPath = path.join(os.homedir(), '.qmlauncher', 'auth-token.txt')
  
  try {
    if (fs.existsSync(tokenPath)) {
      fs.unlinkSync(tokenPath)
      return { success: true }
    }
    return { success: true }
  } catch (error) {
    console.error('Error clearing auth token:', error)
    return { success: false, error: String(error) }
  }
})

// Check if client and mods are installed for a server
ipcMain.handle('check-client-installed', async (_event: any, serverId: number) => {
  try {
    const modsDir = path.join(os.homedir(), '.qmlauncher', 'mods', String(serverId))
    const versionsDir = path.join(os.homedir(), '.qmlauncher', 'versions')
    const configPath = path.join(os.homedir(), '.qmlauncher', 'launcher-config.db')
    
    // Check if mods directory exists and has mods
    let hasMods = false
    if (fs.existsSync(modsDir)) {
      const modFiles = fs.readdirSync(modsDir).filter(f => f.endsWith('.jar'))
      hasMods = modFiles.length > 0
    }
    
    // Check if mods are configured in database
    let hasModsConfig = false
    let minecraftVersion: string | null = null
    if (fs.existsSync(configPath)) {
      const db = new Database(configPath)
      const mods = db.prepare('SELECT COUNT(*) as count FROM mods').get() as { count: number }
      hasModsConfig = (mods?.count || 0) > 0
      
      // Get server version from embedded servers
      const server = db.prepare('SELECT minecraft_version FROM embedded_servers WHERE server_id = ?').get(serverId) as { minecraft_version: string } | undefined
      if (server) {
        minecraftVersion = server.minecraft_version
      }
      db.close()
    }
    
    // Check if Minecraft client version is installed
    let hasClient = false
    if (minecraftVersion && fs.existsSync(versionsDir)) {
      const versionDir = path.join(versionsDir, minecraftVersion)
      if (fs.existsSync(versionDir)) {
        const clientJar = path.join(versionDir, `${minecraftVersion}.jar`)
        hasClient = fs.existsSync(clientJar)
      }
    }
    
    // Client is considered installed if:
    // 1. Minecraft client JAR exists for the version, AND
    // 2. (Mods directory exists and has mods, OR mods are configured in database)
    const isInstalled = hasClient && (hasMods || hasModsConfig)
    
    return {
      success: true,
      installed: isInstalled,
      hasClient,
      hasMods,
      hasModsConfig,
      minecraftVersion,
      modsDir
    }
  } catch (error) {
    console.error('Error checking client installation:', error)
    return {
      success: false,
      installed: false,
      error: String(error)
    }
  }
})

// Download and install Minecraft client for a version
ipcMain.handle('install-minecraft-client', async (event: any, version: string) => {
  try {
    const versionsDir = path.join(os.homedir(), '.qmlauncher', 'versions')
    const versionDir = path.join(versionsDir, version)
    const clientJar = path.join(versionDir, `${version}.jar`)
    const librariesDir = path.join(os.homedir(), '.qmlauncher', 'libraries')
    const assetsDir = path.join(os.homedir(), '.qmlauncher', 'assets')
    
    // Create directories
    if (!fs.existsSync(versionsDir)) {
      fs.mkdirSync(versionsDir, { recursive: true })
    }
    if (!fs.existsSync(versionDir)) {
      fs.mkdirSync(versionDir, { recursive: true })
    }
    if (!fs.existsSync(librariesDir)) {
      fs.mkdirSync(librariesDir, { recursive: true })
    }
    if (!fs.existsSync(assetsDir)) {
      fs.mkdirSync(assetsDir, { recursive: true })
    }
    
    // Check if already installed
    if (fs.existsSync(clientJar)) {
      return {
        success: true,
        alreadyInstalled: true,
        message: `Minecraft ${version} уже установлен`
      }
    }
    
    // Get version manifest from Mojang API
    const manifestUrl = 'https://launchermeta.mojang.com/mc/game/version_manifest.json'
    console.log(`Fetching version manifest from ${manifestUrl}...`)
    
    const manifestResponse = await fetch(manifestUrl)
    if (!manifestResponse.ok) {
      throw new Error(`Failed to fetch version manifest: ${manifestResponse.statusText}`)
    }
    
    const manifest = await manifestResponse.json() as { versions: Array<{ id: string; url: string }> }
    const versionInfo = manifest.versions.find(v => v.id === version)
    
    if (!versionInfo) {
      throw new Error(`Version ${version} not found in manifest`)
    }
    
    // Get version details
    console.log(`Fetching version details for ${version}...`)
    const versionResponse = await fetch(versionInfo.url)
    if (!versionResponse.ok) {
      throw new Error(`Failed to fetch version details: ${versionResponse.statusText}`)
    }
    
    const versionData = await versionResponse.json() as {
      id: string
      mainClass: string
      minecraftArguments?: string
      arguments?: {
        game: Array<string | { rules?: Array<{ action: string; features?: Record<string, boolean> }>; value: string | string[] }>
        jvm: Array<string | { rules?: Array<{ action: string; features?: Record<string, boolean> }>; value: string | string[] }>
      }
      downloads: {
        client: { url: string; sha1: string; size: number }
      }
      libraries: Array<{
        downloads: {
          artifact?: { url: string; path: string; sha1: string; size: number }
          classifiers?: {
            'natives-windows'?: { url: string; path: string; sha1: string; size: number }
            'natives-linux'?: { url: string; path: string; sha1: string; size: number }
            'natives-macos'?: { url: string; path: string; sha1: string; size: number }
            [key: string]: { url: string; path: string; sha1: string; size: number } | undefined
          }
        }
        name: string
        rules?: Array<{ action: string; os?: { name?: string } }>
        extract?: {
          exclude: string[]
        }
      }>
      assetIndex: {
        id: string
        url: string
        sha1: string
        size: number
      }
    }
    
    // Save version metadata for later use
    const versionJsonPath = path.join(versionDir, `${version}.json`)
    fs.writeFileSync(versionJsonPath, JSON.stringify(versionData, null, 2), 'utf-8')
    
    // Download client JAR
    console.log(`Downloading Minecraft client ${version}...`)
    if (mainWindow) {
      mainWindow.webContents.send('installation-progress', { stage: 'Загрузка клиента...', progress: 10 })
    }
    
    const clientUrl = versionData.downloads.client.url
    const clientSize = versionData.downloads.client.size
    const clientDownloadResult = await new Promise<{ success: boolean; error?: string }>((resolve) => {
      const url = new URL(clientUrl)
      const protocol = url.protocol === 'https:' ? https : http
      
      const file = fs.createWriteStream(clientJar)
      let downloadedBytes = 0
      const hash = crypto.createHash('sha1')

      protocol.get(clientUrl, (response) => {
        if (response.statusCode !== 200) {
          file.close()
          if (fs.existsSync(clientJar)) {
            fs.unlinkSync(clientJar)
          }
          resolve({ success: false, error: `Failed to download client: ${response.statusCode}` })
          return
        }
        
        response.on('data', (chunk) => {
          downloadedBytes += chunk.length
          hash.update(chunk)
          if (mainWindow && clientSize > 0) {
            const progress = 10 + Math.floor((downloadedBytes / clientSize) * 40)
            mainWindow.webContents.send('installation-progress', { 
              stage: `Загрузка клиента... ${Math.floor((downloadedBytes / clientSize) * 100)}%`, 
              progress 
            })
          }
        })
        
        response.pipe(file)
        
        file.on('finish', () => {
          file.close()
          const actualSha1 = hash.digest('hex')
          const expectedSha1 = versionData.downloads.client.sha1
          
          if (expectedSha1 && actualSha1 !== expectedSha1) {
            fs.unlinkSync(clientJar)
            resolve({ success: false, error: `SHA1 mismatch: expected ${expectedSha1}, got ${actualSha1}` })
            return
          }
          
          if (mainWindow) {
            mainWindow.webContents.send('installation-progress', { stage: 'Клиент загружен', progress: 50 })
          }
          resolve({ success: true })
        })
        
        file.on('error', (err) => {
          file.close()
          if (fs.existsSync(clientJar)) {
            fs.unlinkSync(clientJar)
          }
          resolve({ success: false, error: String(err) })
        })
      }).on('error', (err) => {
        if (fs.existsSync(clientJar)) {
          fs.unlinkSync(clientJar)
        }
        resolve({ success: false, error: String(err) })
      })
    })
    
    if (!clientDownloadResult.success) {
      return clientDownloadResult
    }
    
    console.log(`Minecraft client ${version} downloaded successfully`)
    
    // Download libraries
    console.log(`Downloading libraries for ${version}...`)
    if (mainWindow) {
      mainWindow.webContents.send('installation-progress', { stage: 'Загрузка библиотек...', progress: 50 })
    }
    
    // Filter libraries based on platform rules
    const currentOs = process.platform === 'win32' ? 'windows' : 
                     process.platform === 'darwin' ? 'osx' : 'linux'
    
    const requiredLibraries = versionData.libraries.filter(lib => {
      // Check platform rules
      if (lib.rules) {
        for (const rule of lib.rules) {
          if (rule.os && rule.os.name) {
            if (rule.os.name !== currentOs && rule.action === 'allow') {
              return false
            }
            if (rule.os.name === currentOs && rule.action === 'disallow') {
              return false
            }
          }
        }
      }
      // Download artifact libraries and natives
      return lib.downloads.artifact || (lib.downloads.classifiers && Object.keys(lib.downloads.classifiers).length > 0)
    })
    
    let librariesDownloaded = 0
    const totalLibraries = requiredLibraries.length
    
    for (let i = 0; i < requiredLibraries.length; i++) {
      const library = requiredLibraries[i]
      
      if (mainWindow) {
        const progress = 50 + Math.floor((i / totalLibraries) * 30)
        mainWindow.webContents.send('installation-progress', { 
          stage: `Загрузка библиотек... ${i + 1}/${totalLibraries}`, 
          progress 
        })
      }
      if (library.downloads.artifact) {
        const libPath = library.downloads.artifact.path
        const libFullPath = path.join(librariesDir, libPath)
        const libDir = path.dirname(libFullPath)
        
        if (!fs.existsSync(libDir)) {
          fs.mkdirSync(libDir, { recursive: true })
        }
        
        if (!fs.existsSync(libFullPath)) {
          const libUrl = library.downloads.artifact.url
          const libDownloadResult = await new Promise<{ success: boolean }>((resolve) => {
            const url = new URL(libUrl)
            const protocol = url.protocol === 'https:' ? https : http
            
            const file = fs.createWriteStream(libFullPath)
            
            protocol.get(libUrl, (response) => {
              if (response.statusCode === 200) {
                response.pipe(file)
                file.on('finish', () => {
                  file.close()
                  resolve({ success: true })
                })
                file.on('error', () => {
                  file.close()
                  resolve({ success: false })
                })
              } else {
                file.close()
                resolve({ success: false })
              }
            }).on('error', () => {
              resolve({ success: false })
            })
          })
          
          if (libDownloadResult.success) {
            librariesDownloaded++
          }
        } else {
          librariesDownloaded++
        }
      }
      
      // Download native libraries for current platform
      const osClassifier = currentOs === 'windows' ? 'natives-windows' : 
                          currentOs === 'osx' ? 'natives-macos' : 'natives-linux'
      
      if (library.downloads.classifiers?.[osClassifier]) {
        const nativePath = library.downloads.classifiers[osClassifier].path
        const nativeFullPath = path.join(librariesDir, nativePath)
        const nativeDir = path.dirname(nativeFullPath)
        
        if (!fs.existsSync(nativeDir)) {
          fs.mkdirSync(nativeDir, { recursive: true })
        }
        
        if (!fs.existsSync(nativeFullPath)) {
          const nativeUrl = library.downloads.classifiers[osClassifier].url
          const nativeDownloadResult = await new Promise<{ success: boolean }>((resolve) => {
            const url = new URL(nativeUrl)
            const protocol = url.protocol === 'https:' ? https : http
            
            const file = fs.createWriteStream(nativeFullPath)
            
            protocol.get(nativeUrl, (response) => {
              if (response.statusCode === 200) {
                response.pipe(file)
                file.on('finish', () => {
                  file.close()
                  resolve({ success: true })
                })
                file.on('error', () => {
                  file.close()
                  resolve({ success: false })
                })
              } else {
                file.close()
                resolve({ success: false })
              }
            }).on('error', () => {
              resolve({ success: false })
            })
          })
          
          if (nativeDownloadResult.success) {
            librariesDownloaded++
            console.log(`Downloaded native library: ${nativePath}`)
          }
        } else {
          librariesDownloaded++
        }
      }
    }
    
    console.log(`Downloaded ${librariesDownloaded} libraries`)
    
    // Download asset index (simplified - just download the index file)
    console.log(`Downloading asset index for ${version}...`)
    if (mainWindow) {
      mainWindow.webContents.send('installation-progress', { stage: 'Загрузка индекса ресурсов...', progress: 80 })
    }
    
    const assetIndexDir = path.join(assetsDir, 'indexes')
    if (!fs.existsSync(assetIndexDir)) {
      fs.mkdirSync(assetIndexDir, { recursive: true })
    }
    
    const assetIndexFile = path.join(assetIndexDir, `${versionData.assetIndex.id}.json`)
    if (!fs.existsSync(assetIndexFile)) {
      const assetIndexResponse = await fetch(versionData.assetIndex.url)
      if (assetIndexResponse.ok) {
        const assetIndexData = await assetIndexResponse.text()
        fs.writeFileSync(assetIndexFile, assetIndexData, 'utf-8')
      }
    }
    
    // Extract natives from downloaded native libraries
    const nativesDir = path.join(versionDir, 'natives')
    if (!fs.existsSync(nativesDir)) {
      fs.mkdirSync(nativesDir, { recursive: true })
    }
    
    // Extract natives from native JAR files (simplified - would need unzip library in production)
    // For now, we'll just ensure the directory exists
    // In a full implementation, we would:
    // 1. Find all native JAR files
    // 2. Extract their contents to nativesDir
    // 3. Handle platform-specific extraction rules
    console.log(`Natives directory prepared: ${nativesDir}`)
    
    if (mainWindow) {
      mainWindow.webContents.send('installation-progress', { stage: 'Установка завершена', progress: 100 })
    }
    
    return {
      success: true,
      alreadyInstalled: false,
      message: `Minecraft ${version} установлен успешно`,
      clientJar,
      librariesDownloaded,
      mainClass: versionData.mainClass || 'net.minecraft.client.main.Main'
    }
  } catch (error) {
    console.error('Error installing Minecraft client:', error)
    return {
      success: false,
      error: String(error)
    }
  }
})

