<template>
  <div class="c-app">
    <div v-if="store.config" class="c-app--tiles-container" :style="gridStyle">
      <uptime-tile v-for="tile in store.config.tiles" :key="tile.id" :info="tile"></uptime-tile>
    </div>

    <div v-else class="c-app--message">
      <template v-if="store.error">Unable to load configuration: {{ store.error }}</template>
      <template v-else>Loading...</template>
    </div>

    <div class="c-app--connection" :class="{ 'c-app--connection--offline': !store.connected }">
      {{ store.connected ? 'live' : 'offline' }}
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted } from 'vue'
import { store, loadConfig, connectEvents } from './store'
import UptimeTile from './components/UptimeTile.vue'

export default defineComponent({
  name: 'App',
  components: { UptimeTile },
  setup () {
    const gridStyle = computed(() => ({
      '--columns': store.config?.columns ?? 4,
    }))

    onMounted(() => {
      void loadConfig().then(connectEvents)
    })

    return { store, gridStyle }
  },
})
</script>

<style lang="css" scoped>
.c-app {
  min-height: 100vh;
}

.c-app--tiles-container {
  display: grid;
  grid-template-columns: repeat(var(--columns), 1fr);
  gap: 10px;
  padding: 10px;
}

.c-app--message {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  font-size: 1.2rem;
  color: #888;
}

.c-app--connection {
  position: fixed;
  bottom: 8px;
  right: 10px;
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--color-succeeded);
  opacity: 0.7;
}

.c-app--connection--offline {
  color: var(--color-failed);
}
</style>
