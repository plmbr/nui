// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loop/internal/model"
)

func TestHandleSessionUploads(t *testing.T) {
	sessionID := "sess-upload"
	mu.Lock()
	sessions = []model.Session{{
		ID:        sessionID,
		Name:      "Upload test",
		AgentType: "claude-code",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		sessions = nil
		mu.Unlock()
		_ = os.RemoveAll(filepath.Join(os.TempDir(), "loop-uploads", sessionID))
	})

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID+"/uploads", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handleSessionUploads(rec, req, sessionID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Path == "" || resp.URL == "" || resp.MediaType != "image/png" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if _, err := os.Stat(resp.Path); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}

	serveReq := httptest.NewRequest(http.MethodGet, resp.URL, nil)
	serveRec := httptest.NewRecorder()
	handleSessionUploadServe(serveRec, serveReq, sessionID, filepath.Base(resp.Path))
	if serveRec.Code != http.StatusOK {
		t.Fatalf("serve status = %d", serveRec.Code)
	}
	if ct := serveRec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type = %q, want image/png", ct)
	}
}

func TestHandleSessionUploadsAcceptsTextFile(t *testing.T) {
	sessionID := "sess-upload-txt"
	mu.Lock()
	sessions = []model.Session{{
		ID:        sessionID,
		Name:      "Upload test",
		AgentType: "claude-code",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		sessions = nil
		mu.Unlock()
		_ = os.RemoveAll(filepath.Join(os.TempDir(), "loop-uploads", sessionID))
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+sessionID+"/uploads", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handleSessionUploads(rec, req, sessionID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Path == "" || resp.URL == "" || resp.Filename != "notes.txt" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !strings.HasSuffix(resp.Path, ".txt") {
		t.Fatalf("path = %q, want .txt suffix", resp.Path)
	}
	if _, err := os.Stat(resp.Path); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
}
