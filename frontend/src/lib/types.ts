export type InitialState = {
  filter: string
  intervalMs: number
  pingEnabled: boolean
  paused: boolean
  ready: boolean
  scanning: boolean
  lastScan: number
  lastError: string
}

export type TrackerStatus = {
  ready: boolean
  scanning: boolean
  lastScan: number
  lastError: string
}

export type Connection = {
  key: string
  pid: number
  appName: string
  protocol: string
  direction: string
  local: string
  remote: string
  state: string
  pingMs: number
  loss: number
  txRate: string
  rxRate: string
  age: string
  paused: boolean
}

export type PingSample = {
  time: number
  rttMs: number
  loss: number
}

export type SortField = 'app' | 'pid' | 'ping' | 'loss' | 'state' | 'tx' | 'rx'
