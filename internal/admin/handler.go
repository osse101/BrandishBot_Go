package admin

import (
	"io/fs"
	"net/http"
	"strings"
)

// Handler returns an http.Handler that serves the embedded admin SPA.
// It expects to be mounted behind http.StripPrefix so paths arrive without the mount prefix.
// For any path that doesn't match a static file, it serves index.html (SPA fallback).
func Handler() http.Handler {
	// Get the dist subdirectory from the embedded FS
	distRoot, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("admin: failed to get dist sub-filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distRoot))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path == "/" {
			// Serve index.html for root
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try to open the file to check if it exists
		cleanPath := strings.TrimPrefix(path, "/")
		if f, err := distRoot.Open(cleanPath); err == nil {
			f.Close()
			// File exists — serve it with appropriate caching
			if strings.HasPrefix(cleanPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// File doesn't exist — serve index.html (SPA fallback)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
