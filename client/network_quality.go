// Copyright 2026 The frp Authors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"net"
	"strconv"
	"time"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/netquality"
)

type qualityConnector struct {
	Connector
	stats *netquality.Stats
}

func (c *qualityConnector) Connect() (net.Conn, error) {
	conn, err := c.Connector.Connect()
	if err != nil {
		return nil, err
	}
	return c.stats.Wrap(conn), nil
}

func (ctl *Control) qualityWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var sequence uint64
	for {
		select {
		case <-ctl.qualityCtx.Done():
			return
		case now := <-ticker.C:
			sequence++
			snapshot := ctl.sessionCtx.Quality.Sample(now)
			ping := &msg.Ping{QualitySequence: sequence, Quality: &snapshot}
			if err := ctl.sessionCtx.Auth.Setter.SetPing(ping); err != nil {
				continue
			}
			ctl.qualityMu.Lock()
			for seq, start := range ctl.qualityPending {
				if now.Sub(start) > 15*time.Second {
					delete(ctl.qualityPending, seq)
				}
			}
			ctl.qualityPending[sequence] = time.Now()
			ctl.qualityMu.Unlock()
			if err := ctl.msgDispatcher.Send(ping); err != nil {
				return
			}
		}
	}
}

func (ctl *Control) handleQualityPong(pong *msg.Pong) {
	if pong.QualitySequence == 0 {
		return
	}
	ctl.qualityMu.Lock()
	start, ok := ctl.qualityPending[pong.QualitySequence]
	delete(ctl.qualityPending, pong.QualitySequence)
	ctl.qualityMu.Unlock()
	if !ok {
		return
	}
	ctl.sessionCtx.Quality.RecordRTT(time.Since(start))
	if pong.QualityUDPPort <= 0 || pong.QualityUDPPort > 65535 || pong.QualityUDPToken == "" {
		return
	}
	ctl.qualityUDPOnce.Do(func() {
		// A direct UDP socket would bypass an explicitly configured HTTP/SOCKS proxy.
		if ctl.sessionCtx.Common.Transport.ProxyURL != "" {
			ctl.sessionCtx.Quality.SetUDPStatus("proxyUnsupported")
			return
		}
		host, _, err := net.SplitHostPort(ctl.sessionCtx.Conn.RemoteAddr().String())
		if err != nil {
			ctl.sessionCtx.Quality.SetUDPStatus("addressUnavailable")
			return
		}
		go netquality.ProbeUDP(ctl.qualityCtx, net.JoinHostPort(host, strconv.Itoa(pong.QualityUDPPort)), pong.QualityUDPToken, ctl.sessionCtx.Quality)
	})
}
