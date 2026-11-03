// Copyright 2026 The frp Authors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"

	"github.com/fatedier/frp/pkg/netquality"
)

func TestQualityRejectsReplacedGenerationAndClearsOnDisconnect(t *testing.T) {
	r := NewClientRegistry()
	key, _ := r.RegisterWithControlID("u", "c", "run", "", "", "", "", 1)
	r.UpdateQuality("run", 1, netquality.Snapshot{RTTMs: 10})
	got, _ := r.GetByKey(key)
	if got.Quality == nil {
		t.Fatal("missing quality")
	}
	r.RegisterWithControlID("u", "c", "run", "", "", "", "", 2)
	r.UpdateQuality("run", 1, netquality.Snapshot{RTTMs: 99})
	got, _ = r.GetByKey(key)
	if got.Quality != nil {
		t.Fatal("accepted stale generation")
	}
	r.UpdateQuality("run", 2, netquality.Snapshot{RTTMs: 20})
	r.MarkOfflineByRunIDAndControlID("run", 2)
	got, _ = r.GetByKey(key)
	if got.Quality != nil || got.Online {
		t.Fatal("stale offline measurement")
	}
}
