// middleware.go implements cross-cutting concerns like logging and session-less flash messages.
package web

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// HandlerE is an enhanced handler signature that returns routing metadata.
type HandlerE = func(w http.ResponseWriter, r *http.Request) (string, error)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// handlerWrapper injects security, compression, and error recovery into the request lifecycle.
func handlerWrapper(handler HandlerE) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Recovery: Ensure a single handler panic doesn't crash the server.
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "recover", r)
				// On panic, send user home with a generic error toast.
				http.SetCookie(w, &http.Cookie{
					Name: "flash_msg", Value: "error_unexpected", Path: "/", MaxAge: 30, HttpOnly: true,
				})
				http.Redirect(w, req, "/", http.StatusSeeOther)
			}
		}()

		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Restrict request body size to mitigate resource exhaustion.
		req.Body = http.MaxBytesReader(w, req.Body, 2048)

		if req.Method == http.MethodPost {
			origin := req.Header.Get("Origin")
			if origin != "" && !strings.Contains(req.Host, strings.TrimPrefix(strings.TrimPrefix(origin, "http://"), "https://")) {
				http.Error(w, "Cross-origin request denied", http.StatusForbidden)
				return
			}
		}

		writer := w
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			writer = gzipResponseWriter{Writer: gz, ResponseWriter: w}
		}

		totID, err := handler(writer, req)
		slog.Debug("request handled", "method", req.Method, "path", req.URL.Path, "totID", totID)

		if err != nil {
			slog.Warn("request error", "method", req.Method, "path", req.URL.Path, "err", err)

			flashValue := "error_unexpected"
			if err.Error() == "tot does not exist" || !isValidID(totID) {
				flashValue = "error_not_found"
			}

			http.SetCookie(w, &http.Cookie{
				Name: "flash_msg", Value: flashValue, Path: "/", MaxAge: 30, HttpOnly: true,
			})
			http.Redirect(w, req, "/", http.StatusSeeOther)
		}
	}
}

// isValidID validates that the string is a valid standard UUID.
func isValidID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
