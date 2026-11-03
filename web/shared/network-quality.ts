export interface NetworkQualityData {
  updatedAt: number;
  supported: boolean;
  rttMs: number;
  jitterMs: number;
  rttAt: number;
  udpStatus: string;
  udpProbeCount: number;
  udpProbeLost: number;
  udpLossPercent: number | null;
  udpRttMs: number | null;
  bytesSent: number;
  bytesReceived: number;
  sendBps: number;
  receiveBps: number;
}
