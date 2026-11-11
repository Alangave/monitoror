<template>
  <div class="c-uptime-bars" :title="tooltip">
    <div
      v-for="(bar, index) in bars"
      :key="index"
      class="c-uptime-bars--bar"
      :class="'c-uptime-bars--bar--' + bar.status.toLowerCase()"
    ></div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'
import type { Check } from '../types'

const MAX_BARS = 48
const BAR_MS = 60 * 1000

export default defineComponent({
  name: 'UptimeBars',
  props: {
    history: {
      type: Array as () => Check[],
      required: true,
    },
  },
  setup (props) {
    const bars = computed(() => {
      return props.history.slice(-MAX_BARS).map((check) => ({
        status: check.status,
        at: check.at,
        durationMs: check.durationMs,
        message: check.message,
      }))
    })

    const lastCheck = computed(() => props.history[props.history.length - 1])

    const tooltip = computed(() => {
      const check = lastCheck.value
      if (!check) return ''
      const ago = Math.max(1, Math.round((Date.now() - check.at) / BAR_MS))
      const status = check.status === 'SUCCESS' ? 'UP' : 'DOWN'
      return `${status} ${check.durationMs} ms - last check ${ago} min ago${check.message ? ' - ' + check.message : ''}`
    })

    return { bars, tooltip }
  },
})
</script>

<style lang="css" scoped>
.c-uptime-bars {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  width: 100%;
}

.c-uptime-bars--bar {
  flex: 1;
  height: 100%;
  border-radius: 1px;
  min-width: 2px;
}

.c-uptime-bars--bar--success {
  background: rgba(255, 255, 255, 0.45);
}

.c-uptime-bars--bar--failed {
  background: #2c3e50;
}

.c-uptime-bars--bar--pending {
  background: rgba(0, 0, 0, 0.2);
}
</style>
