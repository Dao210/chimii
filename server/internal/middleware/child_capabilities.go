package middleware

import (
	"net/http"
	"strings"
)

// ChildCapabilities is the authoritative product boundary for mch_ sessions.
// Frontend menu hiding is presentational; this middleware prevents a child
// token from calling parent/work-management APIs directly.
func ChildCapabilities(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Actor-Source") != "child_session" {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		allowed := (path == "/api/me" && r.Method == http.MethodGet) ||
			((path == "/api/workspaces" || path == "/api/workspaces/") && r.Method == http.MethodGet) ||
			(path == "/api/child-mode" && r.Method == http.MethodGet) ||
			(path == "/api/child-mode/exit" && r.Method == http.MethodPost) ||
			childBuildCapabilityAllowed(r.Method, path)
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"parent unlock required","code":"child_capability_denied"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func childBuildCapabilityAllowed(method, path string) bool {
	if !strings.HasPrefix(path, "/api/build/") {
		return false
	}
	segments := strings.Split(strings.TrimPrefix(path, "/api/build/"), "/")
	if len(segments) == 0 {
		return false
	}
	switch segments[0] {
	case "sessions":
		return (len(segments) == 1 && method == http.MethodPost) ||
			(len(segments) == 2 && segments[1] != "" && method == http.MethodGet) ||
			(len(segments) == 3 && segments[1] != "" && segments[2] == "answers" && method == http.MethodPost)
	case "creations":
		return (len(segments) == 1 && method == http.MethodGet) ||
			(len(segments) == 2 && segments[1] != "" && method == http.MethodGet) ||
			(len(segments) == 3 && segments[1] != "" && segments[2] == "export.mpd" && method == http.MethodGet)
	default:
		return false
	}
}
