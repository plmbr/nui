// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"nui/internal/store"

	"github.com/google/uuid"
)

const maxUploadSize = 10 << 20 // 10 MiB

var allowedImageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func uploadExtension(originalName, mediaType string) (string, bool) {
	if ext, ok := allowedImageTypes[mediaType]; ok {
		return ext, true
	}
	if ext := sanitizeUploadExt(filepath.Ext(strings.TrimSpace(originalName))); ext != "" {
		return ext, true
	}
	if exts, _ := mime.ExtensionsByType(mediaType); len(exts) > 0 {
		if ext := sanitizeUploadExt(exts[0]); ext != "" {
			return ext, true
		}
	}
	return ".bin", true
}

func sanitizeUploadExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" || ext == "." {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if len(ext) > 16 {
		return ""
	}
	for _, r := range ext[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return ext
}

type uploadResponse struct {
	Path      string `json:"path"`
	URL       string `json:"url"`
	MediaType string `json:"mediaType"`
	Filename  string `json:"filename"`
}

func handleSessionUploads(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	_, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large or invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read upload", http.StatusInternalServerError)
		return
	}
	if len(data) == 0 {
		http.Error(w, "empty file", http.StatusBadRequest)
		return
	}

	mediaType := http.DetectContentType(data)
	ext, ok := uploadExtension(header.Filename, mediaType)
	if !ok {
		http.Error(w, "unsupported file type", http.StatusBadRequest)
		return
	}

	dir, err := store.SessionUploadsDir(sessionID)
	if err != nil {
		http.Error(w, "failed to prepare upload directory", http.StatusInternalServerError)
		return
	}

	filename := uuid.NewString() + ext
	dest := filepath.Join(dir, filename)
	if err := os.WriteFile(dest, data, 0600); err != nil {
		http.Error(w, "failed to save upload", http.StatusInternalServerError)
		return
	}

	originalName := strings.TrimSpace(header.Filename)
	if originalName == "" {
		originalName = filename
	}

	writeJSON(w, http.StatusCreated, uploadResponse{
		Path:      dest,
		URL:       fmt.Sprintf("/api/sessions/%s/uploads/%s", sessionID, filename),
		MediaType: mediaType,
		Filename:  originalName,
	})
}

func handleSessionUploadServe(w http.ResponseWriter, r *http.Request, sessionID, filename string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mu.RLock()
	_, ok := findSession(sessionID)
	mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	filename = filepath.Base(filename)
	if filename == "." || filename == ".." || strings.Contains(filename, string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}

	dir, err := store.SessionUploadsDir(sessionID)
	if err != nil {
		http.Error(w, "failed to locate upload", http.StatusInternalServerError)
		return
	}

	path := filepath.Join(dir, filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	mediaType := mime.TypeByExtension(filepath.Ext(filename))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, path)
}
