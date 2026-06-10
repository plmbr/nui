// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"fmt"
	"io/fs"
	"net/http"
)

func Start(port int, uiFiles fs.FS) error {
	mux := http.NewServeMux()

	assetsFS, err := fs.Sub(uiFiles, "assets")
	if err != nil {
		return fmt.Errorf("reading embedded assets: %w", err)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	if err := initStore(); err != nil {
		return fmt.Errorf("loading store: %w", err)
	}

	registerAPIRoutes(mux)

	mux.HandleFunc("/health", handleHealth)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		file, err := fs.ReadFile(uiFiles, "index.html")
		if err != nil {
			http.Error(w, "error reading index.html", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(file)
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Listening on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}
