// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package hitl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"nui/internal/store"
)

type persistedStore struct {
	mu       sync.RWMutex
	requests map[string]*Request
	responses map[string]*Response
}

func newStore() *persistedStore {
	s := &persistedStore{
		requests:  map[string]*Request{},
		responses: map[string]*Response{},
	}
	s.load()
	return s
}

func (s *persistedStore) path() (string, error) {
	dir, err := store.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hitl-requests.json"), nil
}

type diskData struct {
	Requests  []*Request  `json:"requests"`
	Responses []*Response `json:"responses"`
}

func (s *persistedStore) load() {
	path, err := s.path()
	if err != nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var wrap diskData
	if err := json.Unmarshal(data, &wrap); err != nil {
		return
	}
	for _, r := range wrap.Requests {
		if r != nil && r.RequestID != "" {
			s.requests[r.RequestID] = r
		}
	}
	for _, r := range wrap.Responses {
		if r != nil && r.RequestID != "" {
			s.responses[r.RequestID] = r
		}
	}
}

func (s *persistedStore) save() error {
	path, err := s.path()
	if err != nil {
		return err
	}
	s.mu.RLock()
	wrap := diskData{
		Requests:  make([]*Request, 0, len(s.requests)),
		Responses: make([]*Response, 0, len(s.responses)),
	}
	for _, r := range s.requests {
		wrap.Requests = append(wrap.Requests, r)
	}
	for _, r := range s.responses {
		wrap.Responses = append(wrap.Responses, r)
	}
	s.mu.RUnlock()
	data, err := json.MarshalIndent(wrap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (s *persistedStore) putRequest(req *Request) error {
	s.mu.Lock()
	s.requests[req.RequestID] = req
	s.mu.Unlock()
	return s.save()
}

func (s *persistedStore) putResponse(resp *Response) error {
	s.mu.Lock()
	s.responses[resp.RequestID] = resp
	s.mu.Unlock()
	return s.save()
}

func (s *persistedStore) getRequest(id string) (*Request, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.requests[id]
	if !ok {
		return nil, false
	}
	copy := *r
	return &copy, true
}

func (s *persistedStore) getResponse(id string) (*Response, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.responses[id]
	if !ok {
		return nil, false
	}
	copy := *r
	return &copy, true
}

func (s *persistedStore) listRequests(filter ListFilter) []Request {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Request
	for _, r := range s.requests {
		if filter.SessionID != "" && r.SessionID != filter.SessionID {
			continue
		}
		if filter.RunID != "" && r.RunID != filter.RunID {
			continue
		}
		if filter.Status != "" && r.Status != filter.Status {
			continue
		}
		if filter.PendingOnly && !isPendingStatus(r.Status) {
			continue
		}
		out = append(out, *r)
	}
	return out
}

// deleteBySession removes all requests and responses for a session.
func (s *persistedStore) deleteBySession(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	s.mu.Lock()
	removed := 0
	for id, r := range s.requests {
		if r != nil && r.SessionID == sessionID {
			delete(s.requests, id)
			delete(s.responses, id)
			removed++
		}
	}
	s.mu.Unlock()
	if removed == 0 {
		return nil
	}
	return s.save()
}

func isPendingStatus(status string) bool {
	return status == StatusPending || status == StatusDelivered
}
