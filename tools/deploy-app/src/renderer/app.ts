// Type definitions for the exposed API
declare const window: Window & {
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
      status: () => Promise<GitStatusResult>
      log: (count?: number) => Promise<CommitInfo[] | { error: string }>
      stage: (files: string[]) => Promise<{ success?: boolean; error?: string }>
      unstage: (files: string[]) => Promise<{ success?: boolean; error?: string }>
      stageAll: () => Promise<{ success?: boolean; error?: string }>
      commit: (message: string) => Promise<{ success?: boolean; commit?: unknown; error?: string }>
      push: () => Promise<{ success?: boolean; error?: string }>
      pull: () => Promise<{ success?: boolean; error?: string }>
    }
    server: {
      status: () => Promise<ServerStatus>
      health: () => Promise<{ status: string; message?: string }>
    }
    shell: {
      openExternal: (url: string) => Promise<void>
    }
  }
}

interface GitStatusResult {
  branch: string
  tracking: string | null
  ahead: number
  behind: number
  staged: FileChange[]
  unstaged: FileChange[]
  untracked: string[]
  lastCommit: CommitInfo | null
  error?: string
}

interface FileChange {
  path: string
  status: 'modified' | 'added' | 'deleted' | 'renamed' | 'copied'
}

interface CommitInfo {
  hash: string
  shortHash: string
  message: string
  author: string
  date: string
}

interface ServerStatus {
  online: boolean
  version: string
  uptime: number
  lastDeploy: string
  containerId?: string
  error?: string
}

// State
let selectedFiles: Set<string> = new Set()
let gitStatus: GitStatusResult | null = null
let pollInterval: number = 30000
let pollTimer: number | null = null

// DOM Elements
const elements = {
  // Settings
  settingsPanel: document.getElementById('settings-panel')!,
  mainContent: document.getElementById('main-content')!,
  repoPath: document.getElementById('repo-path') as HTMLInputElement,
  serverUrl: document.getElementById('server-url') as HTMLInputElement,
  serverApiKey: document.getElementById('server-api-key') as HTMLInputElement,
  saveSettings: document.getElementById('save-settings')!,
  cancelSettings: document.getElementById('cancel-settings')!,
  openSettings: document.getElementById('open-settings')!,

  // Git
  refreshGit: document.getElementById('refresh-git')!,
  gitBranch: document.getElementById('git-branch')!,
  gitSyncStatus: document.getElementById('git-sync-status')!,
  gitLastCommit: document.getElementById('git-last-commit')!,
  gitCommitMessage: document.getElementById('git-commit-message')!,
  fileList: document.getElementById('file-list')!,
  stageAll: document.getElementById('stage-all')!,
  unstageAll: document.getElementById('unstage-all')!,
  commitMessage: document.getElementById('commit-message') as HTMLTextAreaElement,
  btnCommit: document.getElementById('btn-commit') as HTMLButtonElement,
  btnCommitPush: document.getElementById('btn-commit-push') as HTMLButtonElement,

  // Server
  refreshServer: document.getElementById('refresh-server')!,
  serverOnlineIndicator: document.getElementById('server-online-indicator')!,
  serverOnlineText: document.getElementById('server-online-text')!,
  serverVersion: document.getElementById('server-version')!,
  serverUptime: document.getElementById('server-uptime')!,
  serverLastDeploy: document.getElementById('server-last-deploy')!,

  // Overlay
  overlay: document.getElementById('overlay')!,
  overlayMessage: document.getElementById('overlay-message')!,
  toastContainer: document.getElementById('toast-container')!
}

// Utility functions
function showOverlay(message: string) {
  elements.overlayMessage.textContent = message
  elements.overlay.classList.remove('hidden')
}

function hideOverlay() {
  elements.overlay.classList.add('hidden')
}

function showToast(message: string, type: 'success' | 'error' | 'warning' = 'success') {
  const toast = document.createElement('div')
  toast.className = `toast ${type}`
  toast.textContent = message
  elements.toastContainer.appendChild(toast)

  setTimeout(() => {
    toast.remove()
  }, 4000)
}

function formatRelativeTime(dateStr: string): string {
  if (!dateStr) return '--'

  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins} min ago`
  if (diffHours < 24) return `${diffHours}h ago`
  return `${diffDays}d ago`
}

function formatUptime(seconds: number): string {
  if (!seconds) return '--'

  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)

  if (hours > 24) {
    const days = Math.floor(hours / 24)
    return `${days}d ${hours % 24}h`
  }
  return `${hours}h ${mins}m`
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    modified: 'M',
    added: 'A',
    deleted: 'D',
    renamed: 'R',
    copied: 'C',
    untracked: '?'
  }
  return labels[status] || '?'
}

// Settings functions
async function loadSettings() {
  const settings = await window.api.settings.get()

  elements.repoPath.value = settings.repoPath || ''
  elements.serverUrl.value = settings.serverUrl || ''
  elements.serverApiKey.value = settings.serverApiKey || ''
  pollInterval = settings.pollInterval || 30000

  // Show settings panel if not configured
  if (!settings.repoPath) {
    showSettingsPanel()
  }

  return settings
}

function showSettingsPanel() {
  elements.settingsPanel.classList.remove('hidden')
  elements.mainContent.classList.add('hidden')
}

function hideSettingsPanel() {
  elements.settingsPanel.classList.add('hidden')
  elements.mainContent.classList.remove('hidden')
}

async function saveSettings() {
  const settings = {
    repoPath: elements.repoPath.value.trim(),
    serverUrl: elements.serverUrl.value.trim(),
    serverApiKey: elements.serverApiKey.value.trim()
  }

  if (!settings.repoPath) {
    showToast('Repository path is required', 'error')
    return
  }

  await window.api.settings.save(settings)
  hideSettingsPanel()
  showToast('Settings saved')

  // Refresh data
  await refreshAll()
}

// Git functions
async function refreshGitStatus() {
  const result = await window.api.git.status()

  if (result.error) {
    elements.gitBranch.textContent = '--'
    elements.gitSyncStatus.textContent = ''
    elements.gitLastCommit.textContent = '--'
    elements.gitCommitMessage.textContent = result.error
    elements.fileList.innerHTML = `<div class="empty-state">${result.error}</div>`
    gitStatus = null
    return
  }

  gitStatus = result

  // Update branch info
  elements.gitBranch.textContent = result.branch

  // Sync status
  if (result.ahead > 0 && result.behind > 0) {
    elements.gitSyncStatus.textContent = `${result.ahead} ahead, ${result.behind} behind`
    elements.gitSyncStatus.className = 'sync-status'
  } else if (result.ahead > 0) {
    elements.gitSyncStatus.textContent = `${result.ahead} ahead`
    elements.gitSyncStatus.className = 'sync-status ahead'
  } else if (result.behind > 0) {
    elements.gitSyncStatus.textContent = `${result.behind} behind`
    elements.gitSyncStatus.className = 'sync-status behind'
  } else {
    elements.gitSyncStatus.textContent = ''
  }

  // Last commit
  if (result.lastCommit) {
    elements.gitLastCommit.textContent = result.lastCommit.shortHash
    elements.gitCommitMessage.textContent = result.lastCommit.message
    elements.gitCommitMessage.title = result.lastCommit.message
  } else {
    elements.gitLastCommit.textContent = '--'
    elements.gitCommitMessage.textContent = 'No commits yet'
  }

  // Render file list
  renderFileList()
  updateCommitButtons()
}

function renderFileList() {
  if (!gitStatus) {
    elements.fileList.innerHTML = '<div class="empty-state">No changes detected</div>'
    return
  }

  const allFiles = [
    ...gitStatus.staged.map(f => ({ ...f, staged: true })),
    ...gitStatus.unstaged.map(f => ({ ...f, staged: false })),
    ...gitStatus.untracked.map(path => ({ path, status: 'untracked' as const, staged: false }))
  ]

  if (allFiles.length === 0) {
    elements.fileList.innerHTML = '<div class="empty-state">No changes detected</div>'
    selectedFiles.clear()
    return
  }

  elements.fileList.innerHTML = allFiles.map(file => {
    const isSelected = selectedFiles.has(file.path) || file.staged
    const statusClass = file.status

    return `
      <div class="file-item ${file.staged ? 'staged' : ''}" data-path="${file.path}" data-staged="${file.staged}">
        <input type="checkbox" ${isSelected ? 'checked' : ''}>
        <span class="file-status ${statusClass}">${getStatusLabel(file.status)}</span>
        <span class="file-path" title="${file.path}">${file.path}</span>
      </div>
    `
  }).join('')

  // Add click handlers
  elements.fileList.querySelectorAll('.file-item').forEach(item => {
    const checkbox = item.querySelector('input[type="checkbox"]') as HTMLInputElement
    const path = item.getAttribute('data-path')!

    item.addEventListener('click', (e) => {
      if (e.target !== checkbox) {
        checkbox.checked = !checkbox.checked
      }

      if (checkbox.checked) {
        selectedFiles.add(path)
      } else {
        selectedFiles.delete(path)
      }

      updateCommitButtons()
    })
  })
}

function updateCommitButtons() {
  const hasMessage = elements.commitMessage.value.trim().length > 0
  const hasStaged = gitStatus && gitStatus.staged.length > 0
  const hasSelected = selectedFiles.size > 0

  elements.btnCommit.disabled = !hasMessage || (!hasStaged && !hasSelected)
  elements.btnCommitPush.disabled = !hasMessage || (!hasStaged && !hasSelected)
}

async function stageSelectedFiles() {
  if (selectedFiles.size === 0) return

  showOverlay('Staging files...')
  const result = await window.api.git.stage(Array.from(selectedFiles))
  hideOverlay()

  if (result.error) {
    showToast(result.error, 'error')
  } else {
    selectedFiles.clear()
    await refreshGitStatus()
  }
}

async function stageAllFiles() {
  showOverlay('Staging all files...')
  const result = await window.api.git.stageAll()
  hideOverlay()

  if (result.error) {
    showToast(result.error, 'error')
  } else {
    selectedFiles.clear()
    await refreshGitStatus()
  }
}

async function unstageAllFiles() {
  if (!gitStatus || gitStatus.staged.length === 0) return

  showOverlay('Unstaging files...')
  const files = gitStatus.staged.map(f => f.path)
  const result = await window.api.git.unstage(files)
  hideOverlay()

  if (result.error) {
    showToast(result.error, 'error')
  } else {
    await refreshGitStatus()
  }
}

async function commit(andPush: boolean = false) {
  const message = elements.commitMessage.value.trim()
  if (!message) {
    showToast('Please enter a commit message', 'error')
    return
  }

  // Stage selected files first
  if (selectedFiles.size > 0) {
    await stageSelectedFiles()
  }

  showOverlay('Committing...')
  const commitResult = await window.api.git.commit(message)

  if (commitResult.error) {
    hideOverlay()
    showToast(commitResult.error, 'error')
    return
  }

  if (andPush) {
    showOverlay('Pushing to remote...')
    const pushResult = await window.api.git.push()

    hideOverlay()

    if (pushResult.error) {
      showToast(`Committed but push failed: ${pushResult.error}`, 'warning')
    } else {
      showToast('Committed and pushed successfully!')
      elements.commitMessage.value = ''
    }
  } else {
    hideOverlay()
    showToast('Committed successfully!')
    elements.commitMessage.value = ''
  }

  await refreshGitStatus()
}

// Server functions
async function refreshServerStatus() {
  const status = await window.api.server.status()

  if (status.online) {
    elements.serverOnlineIndicator.className = 'indicator online'
    elements.serverOnlineText.textContent = 'Online'
  } else {
    elements.serverOnlineIndicator.className = 'indicator offline'
    elements.serverOnlineText.textContent = status.error || 'Offline'
  }

  elements.serverVersion.textContent = status.version || '--'
  elements.serverUptime.textContent = formatUptime(status.uptime)
  elements.serverLastDeploy.textContent = status.lastDeploy ? formatRelativeTime(status.lastDeploy) : '--'
}

async function refreshAll() {
  await Promise.all([
    refreshGitStatus(),
    refreshServerStatus()
  ])
}

function startPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
  }

  pollTimer = window.setInterval(() => {
    refreshServerStatus()
  }, pollInterval)
}

// Event listeners
elements.openSettings.addEventListener('click', showSettingsPanel)
elements.cancelSettings.addEventListener('click', hideSettingsPanel)
elements.saveSettings.addEventListener('click', saveSettings)

elements.refreshGit.addEventListener('click', () => refreshGitStatus())
elements.refreshServer.addEventListener('click', () => refreshServerStatus())

elements.stageAll.addEventListener('click', stageAllFiles)
elements.unstageAll.addEventListener('click', unstageAllFiles)

elements.commitMessage.addEventListener('input', updateCommitButtons)
elements.btnCommit.addEventListener('click', () => commit(false))
elements.btnCommitPush.addEventListener('click', () => commit(true))

// Initialize
async function init() {
  await loadSettings()
  await refreshAll()
  startPolling()
}

init()
