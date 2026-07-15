// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestVendorChartFileServed(t *testing.T) {
	uiFiles := fstest.MapFS{
		"vendor/chart.min.js": &fstest.MapFile{Data: []byte("/* chart */")},
		"assets/index.js":     &fstest.MapFile{Data: []byte("console.log()")},
		"index.html":            &fstest.MapFile{Data: []byte("<html></html>")},
	}

	mux := http.NewServeMux()
	assetsFS, err := fs.Sub(uiFiles, "assets")
	if err != nil {
		t.Fatal(err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))
	vendorFS, err := fs.Sub(uiFiles, "vendor")
	if err != nil {
		t.Fatal(err)
	}
	mux.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))

	req := httptest.NewRequest(http.MethodGet, "/vendor/chart.min.js", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "chart") {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}
