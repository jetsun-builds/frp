<template><NetworkQuality :quality="quality" /></template>
<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import NetworkQuality from '@shared/components/NetworkQuality.vue'
import type { NetworkQualityData } from '@shared/network-quality'
import { http } from '../api/http'
const quality = ref<NetworkQualityData | null>(null)
let stopped = false
let timer: ReturnType<typeof setTimeout> | undefined
const refresh = async () => {
  try {
    quality.value = await http.get<NetworkQualityData | null>(
      '/api/network-quality',
    )
  } catch {
    quality.value = null
  }
  if (!stopped) timer = setTimeout(refresh, 5000)
}
onMounted(refresh)
onUnmounted(() => {
  stopped = true
  clearTimeout(timer)
})
</script>
