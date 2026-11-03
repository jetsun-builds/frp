<template>
  <section class="network-quality" aria-label="Network quality">
    <div class="quality-heading">
      <strong>Network quality</strong>
      <span>{{ stale ? "Unavailable / stale" : "FRPC → FRPS · live" }}</span>
    </div>
    <template v-if="quality && !stale">
      <div class="quality-grid">
        <div>
          <small>Control RTT</small
          ><b>{{ quality.supported ? ms(quality.rttMs) : "Unavailable" }}</b>
        </div>
        <div>
          <small>RTT jitter</small
          ><b>{{ quality.supported ? ms(quality.jitterMs) : "—" }}</b>
        </div>
        <div>
          <small>UDP probe loss · {{ quality.udpProbeCount }} samples</small
          ><b>{{
            quality.udpLossPercent == null
              ? "—"
              : quality.udpLossPercent.toFixed(1) + "%"
          }}</b>
        </div>
        <div>
          <small>UDP RTT · {{ quality.udpStatus }}</small
          ><b>{{ quality.udpRttMs == null ? "—" : ms(quality.udpRttMs) }}</b>
        </div>
        <div>
          <small>Upload throughput</small><b>{{ rate(quality.sendBps) }}</b>
        </div>
        <div>
          <small>Download throughput</small
          ><b>{{ rate(quality.receiveBps) }}</b>
        </div>
        <div>
          <small>Session uploaded</small><b>{{ bytes(quality.bytesSent) }}</b>
        </div>
        <div>
          <small>Session downloaded</small
          ><b>{{ bytes(quality.bytesReceived) }}</b>
        </div>
      </div>
      <p>
        Throughput is current stream traffic, not maximum link capacity. UDP
        loss uses the last 60 probes (1 s timeout); a blocked UDP port can show
        100%.
      </p>
    </template>
    <p v-else>Waiting for a connected client and fresh measurements.</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import type { NetworkQualityData } from "../network-quality";
const props = defineProps<{ quality?: NetworkQualityData | null }>();
const now = ref(Date.now());
let timer: ReturnType<typeof setInterval> | undefined;
onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now();
  }, 1000);
});
onUnmounted(() => {
  clearInterval(timer);
});
const stale = computed(
  () => !props.quality || now.value / 1000 - props.quality.updatedAt > 20,
);
const ms = (n: number) => `${n.toFixed(1)} ms`;
const rate = (n: number) =>
  n >= 1e6 ? `${(n / 1e6).toFixed(2)} Mbps` : `${(n / 1e3).toFixed(1)} Kbps`;
const bytes = (n: number) =>
  n >= 1048576
    ? `${(n / 1048576).toFixed(2)} MiB`
    : `${(n / 1024).toFixed(1)} KiB`;
</script>

<style scoped>
.network-quality {
  padding: 20px;
  margin-bottom: 20px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}
.quality-heading {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
  margin-bottom: 16px;
}
.quality-heading span,
small,
p {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.quality-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 18px;
}
small,
b {
  display: block;
}
b {
  margin-top: 6px;
  font-size: 18px;
  font-variant-numeric: tabular-nums;
}
p {
  margin: 16px 0 0;
  line-height: 1.6;
}
</style>
