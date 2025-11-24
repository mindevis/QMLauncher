import { app, BrowserWindow, ipcMain } from 'electron'
import path from 'path'
import { spawn, ChildProcess } from 'child_process'
import fs from 'fs'
import os from 'os'
import Database from 'better-sqlite3'
import https from 'https'
import http from 'http'

const isDev = process.env.NODE_ENV === 'development' || !app.isPackaged

let mainWindow: BrowserWindow | null = null
let minecraftProcess: ChildProcess | null = null

function createWindow() {
  const preloadPath = isDev
    ? path.join(__dirname, 'preload.js')
    : path.join(__dirname, 'preload.js')

  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    webPreferences: {
      preload: preloadPath,
      nodeIntegration: false,
      contextIsolation: true
    },
    icon: path.join(__dirname, '../build/icon.png')
  })

  if (isDev) {
    mainWindow.loadURL('http://localhost:5175')
    mainWindow.webContents.openDevTools()
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'))
  }

  mainWindow.on('closed', () => {
    mainWindow = null
  })
}

app.whenReady().then(() => {
  createWindow()

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

    const { javaPath, gameArgs, jvmArgs, workingDirectory } = launchArgs
    
    const command = javaPath || 'java'
    const args = [...jvmArgs, ...gameArgs]

    minecraftProcess = spawn(command, args, {
      cwd: workingDirectory,
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
    
    // Get API base URL from config
    const apiBaseUrlRow = db.prepare('SELECT value FROM launcher_config WHERE key = ?').get('api_base_url') as { value: string } | undefined
    const apiBaseUrlFromConfig = apiBaseUrlRow?.value || apiBaseUrl || 'http://localhost:8000/api/v1'

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

