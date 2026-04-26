import type { Connection, InitialState, PingSample, SortField, TrackerStatus } from './types'

type Backend = {
  main?: {
    App?: {
      GetInitialState?: () => Promise<InitialState>
      GetStatus?: () => Promise<TrackerStatus>
      GetConnections?: (filter: string, sortField: SortField, sortAsc: boolean) => Promise<Connection[]>
      GetHistory?: (key: string) => Promise<PingSample[] | null>
      Pause?: () => Promise<boolean>
      Resume?: () => Promise<boolean>
      IsPaused?: () => Promise<boolean>
    }
  }
}

const backend = () => (window as Window & { go?: Backend }).go?.main?.App

function unavailable<T>(fallback: T): Promise<T> {
  return Promise.resolve(fallback)
}

export function getInitialState(): Promise<InitialState> {
  return backend()?.GetInitialState?.() ?? unavailable(emptyInitialState())
}

export function getStatus(): Promise<TrackerStatus> {
  return backend()?.GetStatus?.() ?? unavailable(emptyStatus())
}

export function getConnections(filter: string, sortField: SortField, sortAsc: boolean): Promise<Connection[]> {
  return backend()?.GetConnections?.(filter, sortField, sortAsc) ?? unavailable([])
}

export function getHistory(key: string): Promise<PingSample[]> {
  return backend()?.GetHistory?.(key).then((value) => value ?? []) ?? unavailable([])
}

export function pause(): Promise<boolean> {
  return backend()?.Pause?.() ?? unavailable(true)
}

export function resume(): Promise<boolean> {
  return backend()?.Resume?.() ?? unavailable(false)
}

export function isPaused(): Promise<boolean> {
  return backend()?.IsPaused?.() ?? unavailable(false)
}

function emptyInitialState(): InitialState {
  return { ...emptyStatus(), filter: '', intervalMs: 3000, pingEnabled: true, paused: false }
}

function emptyStatus(): TrackerStatus {
  return { ready: false, scanning: false, lastScan: 0, lastError: '' }
}
