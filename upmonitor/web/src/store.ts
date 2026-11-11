import { reactive } from 'vue'
import type { TileState, WallConfig } from './types'

interface Store {
  config: WallConfig | null
  states: Record<string, TileState>
  connected: boolean
  error: string | null
}

export const store = reactive<Store>({
  config: null,
  states: {},
  connected: false,
  error: null,
})

export async function loadConfig (): Promise<void> {
  try {
    const response = await fetch('/api/config')
    if (!response.ok) {
      throw new Error(`config request failed (${response.status})`)
    }
    store.config = await response.json()
    store.error = null
  } catch (error) {
    store.error = error instanceof Error ? error.message : String(error)
  }
}

export function connectEvents (): void {
  const source = new EventSource('/api/events')

  source.onopen = () => {
    store.connected = true
    store.error = null
  }

  source.addEventListener('tile', (event) => {
    const state = JSON.parse((event as MessageEvent).data) as TileState
    store.states[state.id] = state
  })

  source.onerror = () => {
    store.connected = false
  }
}
