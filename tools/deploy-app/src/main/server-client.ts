export interface DeploymentStatus {
  online: boolean
  version: string
  uptime: number
  lastDeploy: string
  containerId?: string
  error?: string
}

export interface HealthStatus {
  status: 'ok' | 'error'
  message?: string
}

export class ServerClient {
  private baseUrl: string
  private apiKey: string

  constructor(baseUrl: string, apiKey: string = '') {
    // Ensure no trailing slash
    this.baseUrl = baseUrl.replace(/\/+$/, '')
    this.apiKey = apiKey
  }

  private async fetch(endpoint: string, options: RequestInit = {}): Promise<Response> {
    const url = `${this.baseUrl}${endpoint}`

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {})
    }

    if (this.apiKey) {
      headers['Authorization'] = `Bearer ${this.apiKey}`
    }

    const response = await fetch(url, {
      ...options,
      headers
    })

    return response
  }

  async checkHealth(): Promise<HealthStatus> {
    try {
      const response = await this.fetch('/health')

      if (response.ok) {
        const data = await response.json()
        return { status: 'ok', message: data.status }
      }

      return { status: 'error', message: `HTTP ${response.status}` }
    } catch (err) {
      return { status: 'error', message: String(err) }
    }
  }

  async getDeploymentStatus(): Promise<DeploymentStatus> {
    try {
      // First check basic health
      const health = await this.checkHealth()

      if (health.status !== 'ok') {
        return {
          online: false,
          version: 'unknown',
          uptime: 0,
          lastDeploy: '',
          error: health.message
        }
      }

      // Try to get detailed deploy status
      try {
        const response = await this.fetch('/api/deploy/status')

        if (response.ok) {
          const data = await response.json()
          return {
            online: true,
            version: data.version || 'unknown',
            uptime: data.uptime_seconds || 0,
            lastDeploy: data.last_deploy || '',
            containerId: data.container_id
          }
        }
      } catch {
        // Deploy endpoint may not exist yet, just use health status
      }

      // Fallback to basic online status
      return {
        online: true,
        version: 'unknown',
        uptime: 0,
        lastDeploy: ''
      }
    } catch (err) {
      return {
        online: false,
        version: 'unknown',
        uptime: 0,
        lastDeploy: '',
        error: String(err)
      }
    }
  }

  async getLogs(lines: number = 100): Promise<string[]> {
    try {
      const response = await this.fetch(`/api/deploy/logs?lines=${lines}`)

      if (response.ok) {
        const data = await response.json()
        return data.logs || []
      }

      return []
    } catch {
      return []
    }
  }
}
