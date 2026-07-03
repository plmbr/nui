// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestCollectingEventsDrainsBeforeUpstreamClose(t *testing.T) {
	upstream := make(chan Event, 8)
	collecting := &collectingEvents{upstream: upstream}
	events := collecting.start()
	events <- Event{Type: EventText, Content: "hello"}
	events <- Event{Type: EventDone, SessionID: "sess-1"}
	collecting.finish()

	close(upstream)
	got := ""
	for ev := range upstream {
		if ev.Type == EventText {
			got += ev.Content
		}
	}
	if got != "hello" {
		t.Fatalf("text = %q, want hello", got)
	}
	if collecting.text != "hello" {
		t.Fatalf("collecting.text = %q, want hello", collecting.text)
	}
}
