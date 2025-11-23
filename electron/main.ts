import { app, BrowserWindow, ipcMain } from 'electron'
import path from 'path'
import { spawn, ChildProcess } from 'child_process'
import fs from 'fs'
import os from 'os'

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

