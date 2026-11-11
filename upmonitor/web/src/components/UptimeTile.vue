<template>
  <div class="c-uptime-tile" :class="statusClass">
    <div class="c-uptime-tile--content" v-if="state">
      <div class="c-uptime-tile--label">{{ info.label || info.id }}</div>

      <div class="c-uptime-tile--message" v-if="state.message">{{ state.message }}</div>

      <div class="c-uptime-tile--value">{{ displayedUptime }}</div>
      <div class="c-uptime-tile--response-time" v-if="state.status !== 'PENDING'">
        {{ state.responseTimeMs }} ms
      </div>

      <uptime-bars :history="state.history" class="c-uptime-tile--bars"></uptime-bars>

      <div class="c-uptime-tile--since" v-if="sinceLabel">{{ sinceLabel }}</div>
    </div>

    <div class="c-uptime-tile--content" v-else>
      <div class="c-uptime-tile--label">{{ info.label || info.id }}</div>
      <div class="c-uptime-tile--value">...</div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onUnmounted, ref } from 'vue'
import type { TileInfo, TileState } from '../types'
import { store } from '../store'
import UptimeBars from './UptimeBars.vue'

export default defineComponent({
  name: 'UptimeTile',
  components: { UptimeBars },
  props: {
    info: {
      type: Object as () => TileInfo,
      required: true,
    },
  },
  setup (props) {
    const state = computed<TileState | undefined>(() => store.states[props.info.id])

    const statusClass = computed(() => {
      if (!state.value) return 'c-uptime-tile--pending'
      return `c-uptime-tile--${state.value.status.toLowerCase()}`
    })

    const displayedUptime = computed(() => {
      if (!state.value) return '...'
      return (state.value.uptime * 100).toFixed(2) + '%'
    })

    const sinceLabel = computed(() => {
      if (!state.value || !state.value.since) return ''
      const duration = now.value - state.value.since
      const minutes = Math.floor(duration / 60000)
      if (minutes < 1) return 'just now'
      if (minutes < 60) return `${minutes}m`
      const hours = Math.floor(minutes / 60)
      if (hours < 24) return `${hours}h ${minutes % 60}m`
      return `${Math.floor(hours / 24)}d ${hours % 24}h`
    })

    const now = ref(Date.now())
    const timer = window.setInterval(() => { now.value = Date.now() }, 1000)
    onUnmounted(() => window.clearInterval(timer))

    return { state, statusClass, displayedUptime, sinceLabel, now }
  },
})
</script>

<style lang="css" scoped>
.c-uptime-tile {
  position: relative;
  aspect-ratio: 2 / 1;
  border-radius: 4px;
  overflow: hidden;
  color: #fff;
  background: var(--tile-background, var(--color-pending))
    linear-gradient(rgba(255, 255, 255, 0.1), transparent);
  box-shadow: inset 0 -3px 0 rgba(0, 0, 0, 0.25);
}

.c-uptime-tile--success { --tile-background: var(--color-succeeded); }
.c-uptime-tile--failed { --tile-background: var(--color-failed); }
.c-uptime-tile--pending { --tile-background: var(--color-pending); }

.c-uptime-tile--content {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 12px 14px;
}

.c-uptime-tile--label {
  font-size: 1rem;
  font-weight: 700;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.c-uptime-tile--message {
  font-size: 0.7rem;
  opacity: 0.85;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.c-uptime-tile--value {
  flex: 1;
  display: flex;
  align-items: center;
  font-size: 2rem;
  font-weight: 700;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.c-uptime-tile--response-time {
  font-size: 0.8rem;
  opacity: 0.85;
  margin-bottom: 6px;
}

.c-uptime-tile--bars {
  height: 34px;
}

.c-uptime-tile--since {
  margin-top: 4px;
  font-size: 0.65rem;
  opacity: 0.7;
  text-align: right;
}
</style>
