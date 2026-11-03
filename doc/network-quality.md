# 网络质量统计（0.71.1-network-quality.1）

FRPS 的 Clients 列表、客户端详情页，以及 FRPC 本地管理面板展示每个当前会话的质量。无需管理员权限或 ICMP。

## 配置

FRPS 的 TOML 配置添加以下顶层字段（放在 `[webServer]` 等表之前）：

```toml
bindPort = 7000
networkQualityUDPPort = 7001
```

在 FRPS 主机防火墙和 NAT 放行 **UDP 7001**，外部与内部端口号需一致。绑定地址沿用 `bindAddr`。默认 `0` 关闭 UDP 探测；RTT 和收发统计仍然工作。

FRPC 自动协商，无需新增配置。两端均升级后，大约 10 秒开始显示完整数据。管理面板仍用原来的 `webServer` 地址和认证设置。

## 指标口径

- Control RTT：每 5 秒在已认证、加密的控制连接上发送带序号的 Ping，测量到匹配 Pong 到达的时间；包含排队和服务端处理时间，适用于 TCP 复用、QUIC、WebSocket 和 wire v1/v2。
- RTT jitter：最近 60 个成功控制探测中，相邻 RTT 差的绝对值平均数，单位毫秒。
- UDP loss：独立 UDP 路径每秒一包，1 秒内未收到匹配回包算一次失败，按最近 60 次**已完成**探测计算往返探测丢失率。它不是 TCP 重传率，也不能区分上行与下行丢包。禁用、未协商时为 `null`；防火墙阻止 UDP 时可能为 100%。
- UDP RTT：最近一个探测的往返时间，失败时为 `null`。UDP 端点为当前控制连接的远端 IP。
- Upload/Download throughput：每 5 秒累计字节差除以实际采样时间，单位 bit/s，方向均从 FRPC 观察。包含控制、工作、visitor 逻辑流的 FRP 帧和应用加密开销；不含 TCP/QUIC/IP、TLS、复用层等外层开销，也不含独立 UDP 探测包。
- Session uploaded/downloaded：当前会话累计逻辑流字节，重新连接归零。未经过该 FRPC↔FRPS 流的 P2P 直连数据不计入。

实时吞吐量是已使用带宽，**不是最大可用带宽/运营商线路速率**。本版本不会自动灌流量测速；空闲时低吞吐是正常现象。

统计由 FRPC 自报给 FRPS，不作为计费或安全审计依据。断线时 FRPS 清除质量数据；20 秒没有新数据的详情面板显示 stale。UDP 探测使用随机 256 位会话能力令牌，认证后经加密控制信道发放，断线即撤销；固定 48 字节收发且每个会话最多每 500 ms 回一包，不提供匿名通用 UDP echo。

显式配置 `transport.proxyURL` 时显示 `proxyUnsupported`，不绕过代理直接发送 UDP。控制 RTT 和吞吐仍可用。旧端点忽略新增 JSON 字段，正常代理能力保持兼容；连接旧 FRPS 时 `supported=false`，不伪造 RTT/丢包。

## API

- FRPC：`GET /api/network-quality`，沿用管理接口认证，断线返回 JSON `null`。
- FRPS：原有 `/api/clients`、`/api/clients/{key}` 以及 v2 对应接口增加可选 `quality` 字段。
- `updatedAt` 是采样/报告时间（Unix 秒）；FRPS 用自身接收报告时间替代它。`rttAt` 是 FRPC 最近成功控制探测时间；跨主机时钟可能不同。
- `udpProbeCount`、`udpProbeLost` 是滚动窗口计数，`udpLossPercent` 为百分数，速率字段 `sendBps` / `receiveBps` 为 bit/s。

## 构建发布

`.github/workflows/network-quality-release.yml` 在 `codex/network-quality` 分支 push 或手动触发后，完成测试、构建两套内嵌面板，输出 Linux amd64 `.tar.gz`、Windows amd64 `.zip`（每个包均含 frps/frpc、配置、文档）和 SHA256 校验和。所有验证完成后发布 `v0.71.1-network-quality.1` 预发布版。使用内置 `GITHUB_TOKEN`，无需仓库的 `GPR_TOKEN`。
