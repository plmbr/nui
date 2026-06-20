package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSuggestDirectories(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"Alpha", "alpine", "beta", ".hidden"} {
		if err := os.Mkdir(filepath.Join(home, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, "also-a-file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := suggestDirectories(home, "~/al")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(home, "Alpha"), filepath.Join(home, "alpine")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestDirectories() = %v, want %v", got, want)
	}

	got, err = suggestDirectories(home, "~/.")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{filepath.Join(home, ".hidden")}) {
		t.Fatalf("hidden suggestions = %v", got)
	}
}

func TestSuggestDirectoriesLimitsResults(t *testing.T) {
	home := t.TempDir()
	for i := 0; i < 25; i++ {
		if err := os.Mkdir(filepath.Join(home, "project-"+string(rune('a'+i))), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := suggestDirectories(home, "~/project-")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 20 {
		t.Fatalf("got %d results, want 20", len(got))
	}
}

func TestSuggestDirectoriesGuidesAbsolutePathTowardHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "Users", "person")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := suggestDirectories(home, string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(string(filepath.Separator), home)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(string(filepath.Separator), strings.Split(rel, string(filepath.Separator))[0])}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("root suggestions = %v, want %v", got, want)
	}

	got, err = suggestDirectories(home, filepath.Dir(home)+string(filepath.Separator))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{home}) {
		t.Fatalf("home parent suggestions = %v", got)
	}
}

func TestSuggestDirectoriesRejectsOutsideHomeAndEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := suggestDirectories(home, outside+string(filepath.Separator)); err != errDirectoryOutsideHome {
		t.Fatalf("outside path error = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(home, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, err := suggestDirectories(home, "~/esc")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("escaping symlink was suggested: %v", got)
	}
}

func TestDirectoriesHandler(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "work"), 0o755); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/directories?path=~/wo", nil)
	recorder := httptest.NewRecorder()
	handleDirectoriesForHome(recorder, req, home)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Directories []string `json:"directories"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response.Directories, []string{filepath.Join(home, "work")}) {
		t.Fatalf("directories = %v", response.Directories)
	}

	outside := filepath.Join(filepath.Dir(home), "outside") + string(filepath.Separator)
	outsideReq := httptest.NewRequest(http.MethodGet, "/api/directories?path="+outside, nil)
	outsideRecorder := httptest.NewRecorder()
	handleDirectoriesForHome(outsideRecorder, outsideReq, home)
	if outsideRecorder.Code != http.StatusBadRequest || !strings.Contains(outsideRecorder.Body.String(), "inside the home") {
		t.Fatalf("outside response = %d %q", outsideRecorder.Code, outsideRecorder.Body.String())
	}
}
