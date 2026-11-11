export type CheckStatus = 'SUCCESS' | 'FAILED' | 'PENDING'

export interface Check {
  at: number
  status: CheckStatus
  durationMs: number
  message?: string
}

export interface TileState {
  id: string
  type: string
  label: string
  status: CheckStatus
  message?: string
  responseTimeMs: number
  uptime: number
  since: number
  tiles?: string[]
  history: Check[]
}

export interface TileInfo {
  id: string
  type: string
  label: string
  tiles?: TileInfo[]
}

export interface WallConfig {
  version: string
  columns: number
  tiles: TileInfo[]
}
