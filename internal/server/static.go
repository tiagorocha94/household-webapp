package server

import (
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/tiagorocha94/household-webapp/internal/api"
)

const (
	// staticIndex is the single page of the frontend.
	staticIndex = "index.html"
	// hashedAssetsPrefix is where the frontend build puts files whose names
	// contain a hash of their content, so they can be cached for good.
	hashedAssetsPrefix = "assets/"
	// contentSecurityPolicy lets the page load only its own files. The
	// frontend has no inline scripts or styles and talks only to this server.
	contentSecurityPolicy = "default-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
)

// staticHandler serves the built frontend in files. A path that is a file is
// served as it is; a path with no file extension that matches nothing gets
// the index page, so the app can open at any address; everything else that is
// missing, anything under /api and any method other than GET and HEAD is left
// to fallback, which answers with the JSON errors of the API.
func staticHandler(files fs.FS, fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || isAPIPath(r.URL.Path) {
			fallback.ServeHTTP(w, r)
			return
		}

		// Cleaning the rooted path removes any ".." before it reaches the
		// file system.
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = staticIndex
		}

		file, info, err := openFile(files, name)
		if errors.Is(err, fs.ErrNotExist) && path.Ext(name) == "" {
			name = staticIndex
			file, info, err = openFile(files, name)
		}
		if errors.Is(err, fs.ErrNotExist) {
			fallback.ServeHTTP(w, r)
			return
		}
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		defer func() { _ = file.Close() }()

		content, ok := file.(io.ReadSeeker)
		if !ok {
			api.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		h := w.Header()
		// Set explicitly: the timeout middleware has already marked every
		// response as JSON, and ServeContent only guesses when there is none.
		h.Set("Content-Type", contentType(name))
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		if strings.HasPrefix(name, hashedAssetsPrefix) {
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			h.Set("Cache-Control", "no-cache")
		}
		http.ServeContent(w, r, name, info.ModTime(), content)
	})
}

// isAPIPath reports whether p belongs to the JSON API.
func isAPIPath(p string) bool {
	return p == "/api" || strings.HasPrefix(p, "/api/")
}

// openFile opens a regular file. A directory counts as not found, so the
// server never lists one.
func openFile(files fs.FS, name string) (fs.File, fs.FileInfo, error) {
	file, err := files.Open(name)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, nil, fs.ErrNotExist
	}
	return file, info, nil
}

func contentType(name string) string {
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}
