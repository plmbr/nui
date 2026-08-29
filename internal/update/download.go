// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ProgressFunc reports download progress (0–1 when total known; bytes otherwise).
type ProgressFunc func(received, total int64)

// Download fetches info.AssetURL into destDir as info.AssetName.
func Download(ctx context.Context, cfg Config, info ReleaseInfo, destDir string, onProgress ProgressFunc) (string, error) {
	cfg = cfg.withDefaults()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, info.AssetName)
	tmp, err := os.CreateTemp(destDir, ".nui-dl-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	res, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
		return "", fmt.Errorf("download %s: %s: %s", info.AssetName, res.Status, strings.TrimSpace(string(body)))
	}
	total := res.ContentLength
	if total <= 0 {
		total = info.Size
	}
	var received int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := res.Body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				return "", werr
			}
			received += int64(n)
			if onProgress != nil {
				onProgress(received, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmpName, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// VerifyChecksum downloads checksums.txt (if needed) and verifies archive SHA-256.
// If checksums.txt is missing (404) and optional is true, verification is skipped.
func VerifyChecksum(ctx context.Context, cfg Config, info ReleaseInfo, archivePath string, optional bool) error {
	cfg = cfg.withDefaults()
	sumURL := info.ChecksumURL
	if sumURL == "" {
		sumURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", cfg.Repo, info.Tag)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sumURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", cfg.UserAgent)
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	res, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound && optional {
		return nil
	}
	if res.StatusCode != http.StatusOK {
		if optional {
			return nil
		}
		return fmt.Errorf("checksums.txt: %s", res.Status)
	}
	expected, err := findChecksum(res.Body, info.AssetName)
	if err != nil {
		if optional && strings.Contains(err.Error(), "not listed") {
			return nil
		}
		return err
	}
	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected, actual) {
		return fmt.Errorf("checksum mismatch for %s: expected %s got %s", info.AssetName, expected, actual)
	}
	return nil
}

func findChecksum(r io.Reader, assetName string) (string, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[len(fields)-1]
		name = filepath.Base(name)
		if name == assetName {
			return fields[0], nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("%s not listed in checksums.txt", assetName)
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
