## Features
* Per-session FRPC↔FRPS network quality: control RTT, RTT jitter, rolling UDP probe loss and UDP RTT.
* Session upload/download byte totals and 5-second live stream throughput, shown in FRPS Clients/details and the FRPC admin dashboard/API.
* Optional authenticated UDP probe listener: set top-level `networkQualityUDPPort = 7001` in frps.toml and allow UDP 7001. No ICMP or administrator privileges required.

## Downloads
* Linux x86-64: archive includes frps and frpc with embedded dashboards.
* Windows x86-64: archive includes frps.exe and frpc.exe with embedded dashboards.
* Verify downloaded archives against SHA256SUMS.

## Measurement notes
* Throughput measures used stream bandwidth, not maximum available link capacity. No automatic saturation speed test is performed.
* UDP loss is round-trip probe failure over the last 60 completed probes with a 1-second timeout. A blocked UDP port can show 100%; disabled/unavailable measurements are null.
* Both endpoints should use this build for full statistics. Existing endpoint compatibility is retained through optional JSON fields.
* Explicit transport.proxyURL disables direct UDP probes so the proxy is not bypassed.
* Statistics reset on reconnect. See the included network-quality.md for API fields, units and configuration.
