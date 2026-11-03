// Copyright 2026 The frp Authors
// SPDX-License-Identifier: Apache-2.0

package netquality

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"sync"
	"time"
)

// Packets are fixed-size, with a random per-session 256-bit capability and
// a fresh nonce. Unknown/expired sessions receive no reply; replies never amplify.
const packetSize = 48

type udpSession struct{ lastReply time.Time }
type UDPServer struct {
	conn     *net.UDPConn
	mu       sync.Mutex
	sessions map[string]*udpSession
}

func ListenUDP(addr string) (*UDPServer, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	c, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, err
	}
	s := &UDPServer{conn: c, sessions: make(map[string]*udpSession)}
	go s.serve()
	return s, nil
}
func (s *UDPServer) Port() int    { return s.conn.LocalAddr().(*net.UDPAddr).Port }
func (s *UDPServer) Close() error { return s.conn.Close() }
func (s *UDPServer) Register() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[string(token)] = &udpSession{}
	return hex.EncodeToString(token), nil
}
func (s *UDPServer) Remove(token string) {
	raw, _ := hex.DecodeString(token)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, string(raw))
}
func (s *UDPServer) serve() {
	buf := make([]byte, packetSize+1)
	for {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if n != packetSize {
			continue
		}
		now := time.Now()
		s.mu.Lock()
		session, ok := s.sessions[string(buf[:32])]
		allowed := ok && now.Sub(session.lastReply) >= 500*time.Millisecond
		if allowed {
			session.lastReply = now
		}
		s.mu.Unlock()
		if allowed {
			_, _ = s.conn.WriteToUDP(buf[:n], addr)
		}
	}
}

// ProbeUDP uses a separate UDP path; it never estimates loss from reliable TCP.
// Each sample has a one-second deadline. Late/duplicate datagrams are ignored.
func ProbeUDP(ctx context.Context, addr, token string, stats *Stats) {
	raw, err := hex.DecodeString(token)
	if err != nil || len(raw) != 32 {
		stats.SetUDPStatus("invalidToken")
		return
	}
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		stats.SetUDPStatus("resolveError")
		return
	}
	c, err := net.DialUDP("udp", nil, a)
	if err != nil {
		stats.SetUDPStatus("dialError")
		return
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	stats.SetUDPStatus("probing")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	packet, reply := make([]byte, packetSize), make([]byte, packetSize+1)
	copy(packet, raw)
	for {
		if ctx.Err() != nil {
			return
		}
		if _, err := rand.Read(packet[32:]); err != nil {
			stats.SetUDPStatus("randomError")
			return
		}
		start := time.Now()
		_ = c.SetDeadline(start.Add(time.Second))
		_, err := c.Write(packet)
		received := false
		if err == nil {
			for {
				n, readErr := c.Read(reply)
				if readErr != nil {
					err = readErr
					break
				}
				if n == packetSize && bytes.Equal(packet, reply[:n]) {
					received = true
					break
				}
			}
		}
		if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
			return
		}
		stats.RecordUDP(received, time.Since(start))
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
