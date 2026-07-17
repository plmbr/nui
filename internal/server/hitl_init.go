// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"sync"

	"nui/internal/agent"
	"nui/internal/extensions"
	"nui/internal/hitl"
	"nui/internal/model"
)

var (
	hitlCoord     *hitl.Coordinator
	hitlCoordOnce sync.Once
	hitlBroadcast = struct {
		mu   sync.RWMutex
		subs map[string]map[chan hitl.Request]struct{}
	}{
		subs: map[string]map[chan hitl.Request]struct{}{},
	}
)

func initHITL() {
	hitlCoordOnce.Do(func() {
		hitlCoord = hitl.NewCoordinator(func(event string, req hitl.Request, resp *hitl.Response) {
			if event == "created" && req.SessionID != "" {
				broadcastHITLRequest(req.SessionID, req)
				deliverExtensionHITLChannels(req)
			}
		})
		hitlCoord.SetPolicyFn(resolveSessionHITLMode)
		agent.SetOrchestrationGate(hitlCoord)
	})
}

func deliverExtensionHITLChannels(req hitl.Request) {
	if extensions.Default == nil {
		return
	}
	var payload map[string]any
	data, err := json.Marshal(req)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &payload)
	for _, ch := range req.Routing.Channels {
		if !extensions.IsExtRef(ch) {
			continue
		}
		_ = extensions.Default.DeliverExtensionHITL(ch, payload, "", req.SessionID)
	}
}

func coordinator() *hitl.Coordinator {
	initHITL()
	return hitlCoord
}

func resolveSessionHITLMode(sessionID, _ string) (string, bool) {
	mu.RLock()
	session, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		return hitl.ModeInteractive, true
	}
	def, hasDef := findADLDef(session.AgentType)
	if !hasDef {
		return hitl.ModeInteractive, true
	}
	if def.HITL.Required {
		return hitl.ModeInteractive, true
	}
	return hitl.EffectiveMode(def, session.AgentConfig), true
}

func subscribeHITL(sessionID string) (chan hitl.Request, func()) {
	ch := make(chan hitl.Request, 8)
	hitlBroadcast.mu.Lock()
	if hitlBroadcast.subs[sessionID] == nil {
		hitlBroadcast.subs[sessionID] = map[chan hitl.Request]struct{}{}
	}
	hitlBroadcast.subs[sessionID][ch] = struct{}{}
	hitlBroadcast.mu.Unlock()
	return ch, func() {
		hitlBroadcast.mu.Lock()
		delete(hitlBroadcast.subs[sessionID], ch)
		if len(hitlBroadcast.subs[sessionID]) == 0 {
			delete(hitlBroadcast.subs, sessionID)
		}
		hitlBroadcast.mu.Unlock()
		close(ch)
	}
}

func broadcastHITLRequest(sessionID string, req hitl.Request) {
	hitlBroadcast.mu.RLock()
	subs := make([]chan hitl.Request, 0, len(hitlBroadcast.subs[sessionID]))
	for ch := range hitlBroadcast.subs[sessionID] {
		subs = append(subs, ch)
	}
	hitlBroadcast.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- req:
		default:
		}
	}
}

func defaultHITLChannels(in hitl.CreateInput, adlDef model.ADLDefinition) hitl.Routing {
	if len(in.Routing.Channels) > 0 {
		return in.Routing
	}
	channels := []string{hitl.ChannelnuiUI}
	if len(adlDef.HITL.Channels) > 0 {
		channels = append([]string{}, adlDef.HITL.Channels...)
	}
	return hitl.Routing{Channels: channels}
}
