import { contextBridge, ipcRenderer } from 'electron'

// Expose protected methods to the renderer process
contextBridge.exposeInMainWorld('api', {
  // Settings
  settings: {
    get: () => ipcRenderer.invoke('settings:get'),
    save: (settings: Record<string, unknown>) => ipcRenderer.invoke('settings:save', settings)
  },

  // Git operations
  git: {
    status: () => ipcRenderer.invoke('git:status'),
    log: (count?: number) => ipcRenderer.invoke('git:log', count),
    stage: (files: string[]) => ipcRenderer.invoke('git:stage', files),
    unstage: (files: string[]) => ipcRenderer.invoke('git:unstage', files),
    stageAll: () => ipcRenderer.invoke('git:stageAll'),
    commit: (message: string) => ipcRenderer.invoke('git:commit', message),
    push: () => ipcRenderer.invoke('git:push'),
    pull: () => ipcRenderer.invoke('git:pull')
  },

  // Server operations
  server: {
    status: () => ipcRenderer.invoke('server:status'),
    health: () => ipcRenderer.invoke('server:health')
  },

  // Utility
  shell: {
    openExternal: (url: string) => ipcRenderer.invoke('shell:openExternal', url)
  }
})

// Type definitions for the exposed API
declare global {
  interface Window {
    api: {
      settings: {
        get: () => Promise<{
          repoPath: string
          serverUrl: string
          serverApiKey: string
          pollInterval: number
        }>
        save: (settings: Record<string, unknown>) => Promise<boolean>
      }
      git: {
        status: () => Promise<{
          branch: string
          tracking: string | null
          ahead: number
          behind: number
          staged: Array<{ path: string; status: string }>
          unstaged: Array<{ path: string; status: string }>
          untracked: string[]
          lastCommit: {
            hash: string
            shortHash: string
            message: string
            author: string
            date: string
          } | null
          error?: string
        }>
        log: (count?: number) => Promise<Array<{
          hash: string
          shortHash: string
          message: string
          author: string
          date: string
        }> | { error: string }>
        stage: (files: string[]) => Promise<{ success?: boolean; error?: string }>
        unstage: (files: string[]) => Promise<{ success?: boolean; error?: string }>
        stageAll: () => Promise<{ success?: boolean; error?: string }>
        commit: (message: string) => Promise<{ success?: boolean; commit?: unknown; error?: string }>
        push: () => Promise<{ success?: boolean; error?: string }>
        pull: () => Promise<{ success?: boolean; error?: string }>
      }
      server: {
        status: () => Promise<{
          online: boolean
          version: string
          uptime: number
          lastDeploy: string
          containerId?: string
          error?: string
        }>
        health: () => Promise<{ status: string; message?: string }>
      }
      shell: {
        openExternal: (url: string) => Promise<void>
      }
    }
  }
}
