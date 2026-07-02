// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	sessionChangeMu       sync.Mutex
	sessionChangeListeners = map[chan struct{}]struct{}{}
)

func notifySessionsChanged() {
	sessionChangeMu.Lock()
	listeners := make([]chan struct{}, 0, len(sessionChangeListeners))
	for ch := range sessionChangeListeners {
		listeners = append(listeners, ch)
	}
	sessionChangeMu.Unlock()

	for _, ch := range listeners {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func subscribeSessionChanges() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	sessionChangeMu.Lock()
	sessionChangeListeners[ch] = struct{}{}
	sessionChangeMu.Unlock()

	unsub := func() {
		sessionChangeMu.Lock()
		delete(sessionChangeListeners, ch)
		sessionChangeMu.Unlock()
	}
	return ch, unsub
}

func handleSessionEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	writeSessionSSE(w, flusher, `{"type":"connected"}`)

	ch, unsub := subscribeSessionChanges()
	defer unsub()

	for {
		select {
		case <-r.Context().Done():
			return
		case _, open := <-ch:
			if !open {
				return
			}
			writeSessionSSE(w, flusher, `{"type":"changed"}`)
		}
	}
}

func writeSessionSSE(w http.ResponseWriter, flusher http.Flusher, data string) {
	fmt.Fprintf(w, "event: sessions\ndata: %s\n\n", data)
	flusher.Flush()
}
