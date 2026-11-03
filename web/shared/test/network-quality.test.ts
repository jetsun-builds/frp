import { afterEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";
import NetworkQuality from "../components/NetworkQuality.vue";
import type { NetworkQualityData } from "../network-quality";

const sample: NetworkQualityData = {
  updatedAt: Math.floor(Date.now() / 1000),
  supported: true,
  rttMs: 12,
  jitterMs: 2,
  rttAt: Math.floor(Date.now() / 1000),
  udpStatus: "disabled",
  udpProbeCount: 0,
  udpProbeLost: 0,
  udpLossPercent: null,
  udpRttMs: null,
  bytesSent: 2048,
  bytesReceived: 4096,
  sendBps: 1e6,
  receiveBps: 2e6,
};

afterEach(() => vi.useRealTimers());

describe("NetworkQuality availability", () => {
  it("keeps unmeasured UDP loss distinct from measured zero loss", async () => {
    const wrapper = mount(NetworkQuality, { props: { quality: sample } });
    expect(wrapper.text()).not.toContain("0.0%");
    await wrapper.setProps({
      quality: { ...sample, udpLossPercent: 0, udpProbeCount: 5 },
    });
    expect(wrapper.text()).toContain("0.0%");
    wrapper.unmount();
  });

  it("expires stale measurements instead of showing old rates as live", async () => {
    vi.useFakeTimers();
    const wrapper = mount(NetworkQuality, { props: { quality: sample } });
    expect(wrapper.text()).toContain("1.00 Mbps");
    await vi.advanceTimersByTimeAsync(21000);
    expect(wrapper.text()).toContain("Unavailable / stale");
    expect(wrapper.text()).not.toContain("1.00 Mbps");
    wrapper.unmount();
  });
});
