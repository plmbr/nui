// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"testing"
)

func TestNotifySessionsChanged(t *testing.T) {
	ch, unsub := subscribeSessionChanges()
	defer unsub()

	notifySessionsChanged()

	select {
	case <-ch:
	default:
		t.Fatal("expected session change notification")
	}
}

func TestNotifySessionsChanged_nonBlockingWhenListenerBusy(t *testing.T) {
	ch, unsub := subscribeSessionChanges()
	defer unsub()

	ch <- struct{}{}
	notifySessionsChanged()
}
