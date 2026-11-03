// Copyright 2026 The frp Authors
// SPDX-License-Identifier: Apache-2.0

package netquality

import (
	"context"
	"encoding/hex"
	"net"
	"testing"
	"time"
)

func TestRollingLossRTTAndRates(t *testing.T) {
	s := NewStats()
	if s.Snapshot().UDPLossPercent != nil {
		t.Fatal("unmeasured loss must be null")
	}
	for range 60 {
		s.RecordUDP(false, 0)
	}
	s.RecordUDP(true, 2*time.Millisecond)
	s.RecordRTT(10 * time.Millisecond)
	s.RecordRTT(14 * time.Millisecond)
	s.sent.Add(1000)
	s.received.Add(2000)
	got := s.Sample(s.lastSample.Add(2 * time.Second))
	if got.UDPProbeCount != 60 || got.UDPProbeLost != 59 {
		t.Fatalf("window: %+v", got)
	}
	if got.UDPLossPercent == nil || *got.UDPLossPercent < 98 || *got.UDPLossPercent > 99 {
		t.Fatal("loss percentage")
	}
	if got.JitterMs != 4 || got.RTTMs != 14 || !got.Supported {
		t.Fatal("RTT/jitter")
	}
	if got.SendBps != 4000 || got.ReceiveBps != 8000 {
		t.Fatalf("rates: %+v", got)
	}
	if s.Snapshot().SendBps != got.SendBps {
		t.Fatal("read changed rates")
	}
}

func TestUDPAuthenticationExpiryAndCancellation(t *testing.T) {
	server, err := ListenUDP("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	token, err := server.Register()
	if err != nil {
		t.Fatal(err)
	}
	stats := NewStats()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { ProbeUDP(ctx, server.conn.LocalAddr().String(), token, stats); close(done) }()
	deadline := time.Now().Add(3 * time.Second)
	for stats.Snapshot().UDPProbeCount == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := stats.Snapshot(); got.UDPProbeCount == 0 || got.UDPProbeLost != 0 || got.UDPRTTMs == nil {
		t.Fatalf("echo failed: %+v", got)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation leaked socket/goroutine")
	}
	server.Remove(token)
	addr := server.conn.LocalAddr().(*net.UDPAddr)
	c, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	packet := make([]byte, packetSize)
	raw, _ := hex.DecodeString(token)
	copy(packet, raw)
	for _, p := range [][]byte{packet, make([]byte, packetSize), make([]byte, packetSize+1)} {
		_ = c.SetDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = c.Write(p)
		if n, e := c.Read(make([]byte, 100)); e == nil {
			t.Fatalf("expired/unknown/malformed packet echoed: %d", n)
		}
	}
}

func TestUDPNoReplyIsLoss(t *testing.T) {
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	s := NewStats()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { ProbeUDP(ctx, c.LocalAddr().String(), hex.EncodeToString(make([]byte, 32)), s); close(done) }()
	deadline := time.Now().Add(3 * time.Second)
	for s.Snapshot().UDPProbeCount == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	got := s.Snapshot()
	cancel()
	<-done
	if got.UDPLossPercent == nil || *got.UDPLossPercent != 100 || got.UDPStatus != "noReply" {
		t.Fatalf("lost reply: %+v", got)
	}
}
