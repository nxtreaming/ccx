export type HealthInfo = {
  status?: string
  timestamp?: string
  uptime?: number
  mode?: string
  version?: {
    version?: string
    buildTime?: string
    gitCommit?: string
  }
  config?: {
    upstreamCount?: number
  }
}

export type DesktopStatus = {
  running: boolean
  starting: boolean
  attached?: boolean
  port: number
  url: string
  pid: number
  binaryPath: string
  dataDir: string
  health?: HealthInfo
  lastError?: string
  logs: string[]
}

export type AgentPlatform = 'claude' | 'codex'
export type AgentProvider = 'ccx' | 'deepseek' | 'mimo'

export type AgentConfigStatus = {
  platform: AgentPlatform
  provider?: string
  targetProvider?: string
  configured: boolean
  matchesCurrentPort: boolean
  needsUpdate: boolean
  currentBaseUrl: string
  targetBaseUrl: string
  configPath: string
  authPath?: string
  hasState: boolean
  lastError?: string
}

export type ApplyAgentConfigRequest = {
  platform: AgentPlatform
  provider?: AgentProvider
  apiKey?: string
  baseUrl?: string
}

export type TabValue = 'status' | 'agent' | 'env' | 'web'
