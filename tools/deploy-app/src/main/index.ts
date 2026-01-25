import { app, BrowserWindow, ipcMain, shell } from 'electron'
import { join } from 'path'
import Store from 'electron-store'
import { GitService } from './git-service'
import { ServerClient } from './server-client'

interface AppSettings {
  repoPath: string
  serverUrl: string
  serverApiKey: string
  pollInterval: number
  windowBounds?: { width: number; height: number; x?: number; y?: number }
}

const store = new Store<AppSettings>({
  defaults: {
    repoPath: '',
    serverUrl: '',
    serverApiKey: '',
    pollInterval: 30000
  }
})

let mainWindow: BrowserWindow | null = null
let gitService: GitService | null = null
let serverClient: ServerClient | null = null

function createWindow() {
  const bounds = store.get('windowBounds', { width: 800, height: 700 })

  mainWindow = new BrowserWindow({
    ...bounds,
    minWidth: 600,
    minHeight: 500,
    titleBarStyle: 'hiddenInset',
    vibrancy: 'window',
    webPreferences: {
      preload: join(__dirname, '../preload/preload.js'),
      contextIsolation: true,
      nodeIntegration: false
    }
  })

  // Save window position on close
  mainWindow.on('close', () => {
    if (mainWindow) {
      store.set('windowBounds', mainWindow.getBounds())
    }
  })

  mainWindow.on('closed', () => {
    mainWindow = null
  })

  // Load the renderer
  if (process.env.NODE_ENV === 'development') {
    mainWindow.loadURL('http://localhost:5173')
    mainWindow.webContents.openDevTools()
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

// Initialize services
function initServices() {
  const repoPath = store.get('repoPath')
  const serverUrl = store.get('serverUrl')
  const serverApiKey = store.get('serverApiKey')

  if (repoPath) {
    gitService = new GitService(repoPath)
  }

  if (serverUrl) {
    serverClient = new ServerClient(serverUrl, serverApiKey)
  }
}

// IPC Handlers - Settings
ipcMain.handle('settings:get', () => {
  return {
    repoPath: store.get('repoPath'),
    serverUrl: store.get('serverUrl'),
    serverApiKey: store.get('serverApiKey'),
    pollInterval: store.get('pollInterval')
  }
})

ipcMain.handle('settings:save', (_, settings: Partial<AppSettings>) => {
  if (settings.repoPath !== undefined) store.set('repoPath', settings.repoPath)
  if (settings.serverUrl !== undefined) store.set('serverUrl', settings.serverUrl)
  if (settings.serverApiKey !== undefined) store.set('serverApiKey', settings.serverApiKey)
  if (settings.pollInterval !== undefined) store.set('pollInterval', settings.pollInterval)

  // Reinitialize services with new settings
  initServices()
  return true
})

// IPC Handlers - Git
ipcMain.handle('git:status', async () => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    return await gitService.getStatus()
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:log', async (_, count: number = 10) => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    return await gitService.getLog(count)
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:stage', async (_, files: string[]) => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    await gitService.stageFiles(files)
    return { success: true }
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:unstage', async (_, files: string[]) => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    await gitService.unstageFiles(files)
    return { success: true }
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:stageAll', async () => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    await gitService.stageAll()
    return { success: true }
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:commit', async (_, message: string) => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    const result = await gitService.commit(message)
    return { success: true, commit: result }
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:push', async () => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    await gitService.push()
    return { success: true }
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('git:pull', async () => {
  if (!gitService) {
    return { error: 'Repository not configured' }
  }
  try {
    await gitService.pull()
    return { success: true }
  } catch (err) {
    return { error: String(err) }
  }
})

// IPC Handlers - Server
ipcMain.handle('server:status', async () => {
  if (!serverClient) {
    return { error: 'Server not configured' }
  }
  try {
    return await serverClient.getDeploymentStatus()
  } catch (err) {
    return { error: String(err) }
  }
})

ipcMain.handle('server:health', async () => {
  if (!serverClient) {
    return { error: 'Server not configured' }
  }
  try {
    return await serverClient.checkHealth()
  } catch (err) {
    return { error: String(err) }
  }
})

// IPC Handlers - Utility
ipcMain.handle('shell:openExternal', (_, url: string) => {
  shell.openExternal(url)
})

// App lifecycle
app.whenReady().then(() => {
  initServices()
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
