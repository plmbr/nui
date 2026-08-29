// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nui/internal/appversion"
)

// State is the Electron-style updater lifecycle state.
type State string

const (
	StateIdle        State = "idle"
	StateChecking    State = "checking"
	StateAvailable   State = "available"
	StateUpToDate    State = "upToDate"
	StateDownloading State = "downloading"
	StateReady       State = "ready"
	StateInstalling  State = "installing"
	StateError       State = "error"
)

// Target selects where a CLI update is applied.
type Target string

const (
	TargetSelf    Target = "self"    // replace the running nui binary
	TargetPathCLI Target = "pathCli" // replace ~/.local/bin/nui (desktop PATH install)
)

// Status is a snapshot of updater progress for one track (CLI or desktop).
type Status struct {
	State            State   `json:"state"`
	Kind             Kind    `json:"kind"`
	Target           Target  `json:"target,omitempty"`
	CurrentVersion   string  `json:"currentVersion"`
	AvailableVersion string  `json:"availableVersion,omitempty"`
	AvailableTag     string  `json:"availableTag,omitempty"`
	Notes            string  `json:"notes,omitempty"`
	AssetName        string  `json:"assetName,omitempty"`
	BytesReceived    int64   `json:"bytesReceived,omitempty"`
	BytesTotal       int64   `json:"bytesTotal,omitempty"`
	Progress         float64 `json:"progress,omitempty"`
	Error            string  `json:"error,omitempty"`
	LastCheckedAt    string  `json:"lastCheckedAt,omitempty"`
	ArchivePath      string  `json:"-"`
}

// Manager tracks check/download/apply for one Kind (cli or desktop).
type Manager struct {
	mu     sync.Mutex
	cfg    Config
	status Status
	info   ReleaseInfo
}

// NewManager creates a manager for kind. currentVersion defaults to appversion.Get().
func NewManager(kind Kind, currentVersion string) *Manager {
	if currentVersion == "" {
		currentVersion = appversion.Get()
	}
	return &Manager{
		cfg: Config{Kind: kind, CurrentVersion: currentVersion},
		status: Status{
			State:          StateIdle,
			Kind:           kind,
			CurrentVersion: currentVersion,
		},
	}
}

// Status returns a copy of the current status.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// SetCurrentVersion updates the baseline version used for comparisons.
func (m *Manager) SetCurrentVersion(v string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.CurrentVersion = v
	m.status.CurrentVersion = v
}

// Check queries GitHub for updates. force treats same version as available.
func (m *Manager) Check(ctx context.Context, force bool) (Status, error) {
	m.mu.Lock()
	m.status.State = StateChecking
	m.status.Error = ""
	cfg := m.cfg
	m.mu.Unlock()

	info, err := Check(ctx, cfg)
	now := time.Now().UTC().Format(time.RFC3339)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.LastCheckedAt = now
	if err != nil {
		// Check failures (rate limits, network) are transient — don't block the UI.
		m.status.State = StateIdle
		m.status.Error = ""
		return m.status, err
	}
	m.info = info
	m.status.AvailableVersion = info.Version
	m.status.AvailableTag = info.Tag
	m.status.Notes = info.Notes
	m.status.AssetName = info.AssetName
	newer := IsNewer(info.Version, cfg.CurrentVersion)
	if newer || force {
		m.status.State = StateAvailable
	} else {
		m.status.State = StateUpToDate
	}
	return m.status, nil
}

// Download downloads the previously checked release archive.
func (m *Manager) Download(ctx context.Context) (Status, error) {
	m.mu.Lock()
	if m.status.State != StateAvailable && m.status.State != StateReady && m.status.State != StateError {
		st := m.status
		m.mu.Unlock()
		return st, fmt.Errorf("no update available to download (state=%s)", st.State)
	}
	info := m.info
	cfg := m.cfg
	if info.AssetURL == "" {
		m.mu.Unlock()
		return m.status, fmt.Errorf("run check first")
	}
	m.status.State = StateDownloading
	m.status.Error = ""
	m.status.BytesReceived = 0
	m.status.BytesTotal = info.Size
	m.status.Progress = 0
	m.mu.Unlock()

	dir, err := os.MkdirTemp("", "nui-update-*")
	if err != nil {
		return m.fail(err)
	}

	path, err := Download(ctx, cfg, info, dir, func(received, total int64) {
		m.mu.Lock()
		m.status.BytesReceived = received
		m.status.BytesTotal = total
		if total > 0 {
			m.status.Progress = float64(received) / float64(total)
		}
		m.mu.Unlock()
	})
	if err != nil {
		_ = os.RemoveAll(dir)
		return m.fail(err)
	}

	optionalChecksum := cfg.Kind == KindDesktop
	if err := VerifyChecksum(ctx, cfg, info, path, optionalChecksum); err != nil {
		_ = os.RemoveAll(dir)
		return m.fail(err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.ArchivePath = path
	m.status.State = StateReady
	m.status.Progress = 1
	return m.status, nil
}

// ApplyCLI installs the downloaded CLI archive to target.
func (m *Manager) ApplyCLI(target Target) (Status, error) {
	m.mu.Lock()
	if m.status.State != StateReady {
		st := m.status
		m.mu.Unlock()
		return st, fmt.Errorf("update not ready (state=%s)", st.State)
	}
	archive := m.status.ArchivePath
	m.status.State = StateInstalling
	m.status.Target = target
	m.status.Error = ""
	m.mu.Unlock()

	dest, err := resolveCLITarget(target)
	if err != nil {
		return m.fail(err)
	}
	if err := InstallCLIFromArchive(archive, dest); err != nil {
		return m.fail(err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = StateIdle
	m.status.CurrentVersion = m.status.AvailableVersion
	m.cfg.CurrentVersion = m.status.AvailableVersion
	m.status.ArchivePath = ""
	return m.status, nil
}

// ApplyDesktop installs the downloaded desktop archive over the running install.
func (m *Manager) ApplyDesktop() (Status, string, error) {
	m.mu.Lock()
	if m.status.State != StateReady {
		st := m.status
		m.mu.Unlock()
		return st, "", fmt.Errorf("update not ready (state=%s)", st.State)
	}
	archive := m.status.ArchivePath
	m.status.State = StateInstalling
	m.status.Error = ""
	m.mu.Unlock()

	root, err := DesktopInstallRoot()
	if err != nil {
		st, _ := m.fail(err)
		return st, "", err
	}
	if err := ApplyDesktopFromArchive(archive, root); err != nil {
		st, _ := m.fail(err)
		return st, "", err
	}
	exe := DesktopExecutablePath(root)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = StateIdle
	m.status.CurrentVersion = m.status.AvailableVersion
	m.cfg.CurrentVersion = m.status.AvailableVersion
	m.status.ArchivePath = ""
	return m.status, exe, nil
}

// ClearError resets error state to idle (or available if version still newer).
func (m *Manager) ClearError() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.Error = ""
	if m.status.AvailableVersion != "" && IsNewer(m.status.AvailableVersion, m.status.CurrentVersion) {
		m.status.State = StateAvailable
	} else {
		m.status.State = StateIdle
	}
	return m.status
}

func (m *Manager) fail(err error) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.State = StateError
	m.status.Error = err.Error()
	return m.status, err
}

func resolveCLITarget(target Target) (string, error) {
	switch target {
	case TargetPathCLI:
		return DefaultCLIPath()
	case TargetSelf, "":
		return CurrentExecutable()
	default:
		return "", fmt.Errorf("unknown update target %q", target)
	}
}

// CleanupWindowsOld removes dest.old left after a Windows self-replace.
func CleanupWindowsOld(dest string) {
	_ = os.Remove(dest + ".old")
	_ = os.Remove(filepath.Clean(dest) + ".old")
}
