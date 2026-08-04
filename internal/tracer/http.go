package tracer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxRequestBody = 64 << 10

func Handler(session *Session, webDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/workspace", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, session.Snapshot())
	})
	mux.HandleFunc("POST /api/guidance", func(w http.ResponseWriter, r *http.Request) {
		body := http.MaxBytesReader(w, r.Body, maxRequestBody)
		defer body.Close()
		var request struct {
			Guidance string `json:"guidance"`
		}
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeAPIError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))
			return
		}
		if err := ensureJSONEOF(decoder); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		snapshot, err := session.ChangeGuidance(request.Guidance)
		if err != nil {
			writeAPIError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshot)
	})
	mux.Handle("/", staticHandler(webDir))
	return securityHeaders(mux)
}

func staticHandler(webDir string) http.Handler {
	index := filepath.Join(webDir, "index.html")
	if info, err := os.Stat(index); err != nil || info.IsDir() {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, "<!doctype html><title>standard2shape tracer</title><main><h1>Web assets are not built</h1><p>Run <code>npm run build</code>, then restart the tracer.</p></main>")
		})
	}
	files := http.FileServer(http.Dir(webDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath != "." {
			candidate := filepath.Join(webDir, cleanPath)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, index)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("request must contain exactly one JSON value")
	}
	return fmt.Errorf("decode trailing request data: %w", err)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
