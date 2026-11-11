<template>
  <div class="c-monitoror-tile-icon">
    <svg viewBox="0 0 64 64">
      <use :xlink:href="'./icons.svg#' + iconId"></use>
    </svg>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent } from 'vue'

import TileIconId from '@/enums/tileIconId'
import TileType from '@/enums/tileType'

export default defineComponent({
  name: 'MonitororTileIcon',
  props: {
    tileType: {
      type: String as () => TileType,
      required: true,
    },
  },
  setup (props) {
    const iconId = computed(() => {
      switch (props.tileType) {
        case TileType.Group:
          return TileIconId.Group

        case TileType.HttpStatus:
        case TileType.HttpRaw:
        case TileType.HttpFormatted:
          return TileIconId.Http

        default:
          return TileIconId.Http
      }
    })

    return {
      iconId,
    }
  },
})
</script>

<style lang="scss">
  .c-monitoror-tile-icon svg {
    fill: currentColor;
  }
</style>
