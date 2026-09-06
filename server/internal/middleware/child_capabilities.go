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
		path := r.URL.EscapedPath()
		if path == "" {
			path = r.URL.Path
		}
		allowed := (path == "/api/me" && r.Method == http.MethodGet) ||
			((path == "/api/workspaces" || path == "/api/workspaces/") && r.Method == http.MethodGet) ||
			(path == "/api/child-mode" && r.Method == http.MethodGet) ||
			(path == "/api/child-mode/exit" && r.Method == http.MethodPost) ||
			childBuildCapabilityAllowed(r.Method, path) || childCircuitCapabilityAllowed(r.Method, path)
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"parent unlock required","code":"child_capability_denied"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func childCircuitCapabilityAllowed(method, path string) bool {
	if path == "/api/circuit/catalog" || path == "/api/circuit/kits" {
		return method == http.MethodGet
	}
	if strings.HasPrefix(path, "/api/circuit/inventory/") {
		kitID := strings.TrimPrefix(path, "/api/circuit/inventory/")
		return method == http.MethodGet && kitID != "" && !strings.Contains(kitID, "/")
	}
	if path == "/api/circuit/creations" {
		return method == http.MethodGet || method == http.MethodPost
	}
	if !strings.HasPrefix(path, "/api/circuit/creations/") {
		return false
	}
	segments := strings.Split(strings.TrimPrefix(path, "/api/circuit/creations/"), "/")
	return segments[0] != "" && ((len(segments) == 1 && method == http.MethodGet) || (len(segments) == 2 && segments[1] == "progress" && method == http.MethodPut) || (len(segments) == 2 && segments[1] == "trials" && (method == http.MethodGet || method == http.MethodPost)))
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
	case "catalog":
		if method == http.MethodGet && (len(segments) == 1 || (len(segments) == 2 && segments[1] == "parts")) {
			return true
		}
		return len(segments) == 4 && segments[1] != "" && segments[2] == "parts" && method == http.MethodGet
	case "inventory":
		return len(segments) == 1 && (method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete)
	case "inventions":
		return len(segments) == 1 && method == http.MethodGet
	case "conversations":
		return (len(segments) == 2 && segments[1] != "" && method == http.MethodGet) || (len(segments) == 3 && segments[1] != "" && segments[2] == "messages" && method == http.MethodPost)
	case "sessions":
		return (len(segments) == 1 && method == http.MethodPost) ||
			(len(segments) == 2 && segments[1] != "" && method == http.MethodGet) ||
			(len(segments) == 3 && segments[1] != "" && (segments[2] == "answers" || segments[2] == "cancel") && method == http.MethodPost)
	case "creations":
		return (len(segments) == 1 && method == http.MethodGet) ||
			(len(segments) == 2 && segments[1] != "" && method == http.MethodGet) ||
			(len(segments) == 3 && segments[1] != "" && segments[2] == "export.mpd" && method == http.MethodGet) ||
			(len(segments) == 3 && segments[1] != "" && segments[2] == "progress" && (method == http.MethodGet || method == http.MethodPut))
	default:
		return false
	}
}
