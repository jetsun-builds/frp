// Copyright 2026 The frp Authors
// SPDX-License-Identifier: Apache-2.0

// Package netquality measures one frpc session. Byte counts are logical stream
// bytes, including FRP framing/encryption, excluding transport/IP overhead.
package netquality

import (
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Snapshot struct {
	UpdatedAt      int64    `json:"updatedAt"`
	Supported      bool     `json:"supported"`
	RTTMs          float64  `json:"rttMs"`
	JitterMs       float64  `json:"jitterMs"`
	RTTAt          int64    `json:"rttAt"`
	UDPStatus      string   `json:"udpStatus"`
	UDPProbeCount  int      `json:"udpProbeCount"`
	UDPProbeLost   int      `json:"udpProbeLost"`
	UDPLossPercent *float64 `json:"udpLossPercent"`
	UDPRTTMs       *float64 `json:"udpRttMs"`
	BytesSent      int64    `json:"bytesSent"`
	BytesReceived  int64    `json:"bytesReceived"`
	SendBps        float64  `json:"sendBps"`
	ReceiveBps     float64  `json:"receiveBps"`
}

type Stats struct {
	sent                   atomic.Int64
	received               atomic.Int64
	mu                     sync.Mutex
	snapshot               Snapshot
	lastSample             time.Time
	lastSent, lastReceived int64
	udpResults             []bool
	rtts                   []float64
}

func NewStats() *Stats {
	return &Stats{lastSample: time.Now(), snapshot: Snapshot{UDPStatus: "disabled"}}
}

func (s *Stats) Wrap(c net.Conn) net.Conn { return &countingConn{Conn: c, stats: s} }

type countingConn struct {
	net.Conn
	stats *Stats
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.stats.received.Add(int64(n))
	return n, err
}

func (c *countingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	c.stats.sent.Add(int64(n))
	return n, err
}

func (s *Stats) RecordRTT(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ms := float64(d) / float64(time.Millisecond)
	s.rtts = append(s.rtts, ms)
	if len(s.rtts) > 60 {
		s.rtts = s.rtts[1:]
	}
	var jitter float64
	for i := 1; i < len(s.rtts); i++ {
		jitter += math.Abs(s.rtts[i] - s.rtts[i-1])
	}
	if len(s.rtts) > 1 {
		jitter /= float64(len(s.rtts) - 1)
	}
	s.snapshot.Supported = true
	s.snapshot.RTTMs, s.snapshot.JitterMs = ms, jitter
	s.snapshot.RTTAt = time.Now().Unix()
}

func (s *Stats) SetUDPStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.UDPStatus = status
}

func (s *Stats) RecordUDP(received bool, rtt time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.udpResults = append(s.udpResults, received)
	if len(s.udpResults) > 60 {
		s.udpResults = s.udpResults[1:]
	}
	lost := 0
	for _, ok := range s.udpResults {
		if !ok {
			lost++
		}
	}
	percent := 100 * float64(lost) / float64(len(s.udpResults))
	s.snapshot.UDPProbeCount, s.snapshot.UDPProbeLost = len(s.udpResults), lost
	s.snapshot.UDPLossPercent = &percent
	if received {
		ms := float64(rtt) / float64(time.Millisecond)
		s.snapshot.UDPRTTMs = &ms
		s.snapshot.UDPStatus = "ok"
	} else {
		s.snapshot.UDPRTTMs = nil
		s.snapshot.UDPStatus = "noReply"
	}
}

// Sample is called periodically, never by API readers, to keep rates stable.
func (s *Stats) Sample(now time.Time) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	sent, received := s.sent.Load(), s.received.Load()
	if elapsed := now.Sub(s.lastSample).Seconds(); elapsed > 0 {
		s.snapshot.SendBps = float64(sent-s.lastSent) * 8 / elapsed
		s.snapshot.ReceiveBps = float64(received-s.lastReceived) * 8 / elapsed
		s.lastSample, s.lastSent, s.lastReceived = now, sent, received
	}
	s.snapshot.UpdatedAt = now.Unix()
	s.snapshot.Supported = s.snapshot.RTTAt > 0 && now.Unix()-s.snapshot.RTTAt <= 20
	s.snapshot.BytesSent, s.snapshot.BytesReceived = sent, received
	return s.snapshot
}

func (s *Stats) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot
}
