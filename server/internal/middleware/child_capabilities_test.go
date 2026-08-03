package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChildCapabilitiesAllowsOnlyChildProductSurface(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := ChildCapabilities(next)
	for _, test := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodPost, "/api/build/sessions", http.StatusNoContent},
		{http.MethodGet, "/api/build/creations/one", http.StatusNoContent},
		{http.MethodGet, "/api/build/creations/one/export.mpd", http.StatusNoContent},
		{http.MethodPost, "/api/build/sessions/one/answers", http.StatusNoContent},
		{http.MethodPost, "/api/child-mode/exit", http.StatusNoContent},
		{http.MethodDelete, "/api/build/creations/one", http.StatusForbidden},
		{http.MethodPost, "/api/build/creations/one/archive", http.StatusForbidden},
		{http.MethodGet, "/api/build/internal/catalog", http.StatusForbidden},
		{http.MethodGet, "/api/issues", http.StatusForbidden},
		{http.MethodGet, "/api/agents", http.StatusForbidden},
		{http.MethodGet, "/api/builder-secrets", http.StatusForbidden},
		{http.MethodPatch, "/api/me", http.StatusForbidden},
	} {
		req := httptest.NewRequest(test.method, test.path, nil)
		req.Header.Set("X-Actor-Source", "child_session")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != test.want {
			t.Errorf("%s %s = %d, want %d", test.method, test.path, recorder.Code, test.want)
		}
	}
}

func TestChildCapabilitiesDoesNotRestrictParent(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodDelete, "/api/issues/one", nil)
	recorder := httptest.NewRecorder()
	ChildCapabilities(next).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("parent request = %d", recorder.Code)
	}
}
