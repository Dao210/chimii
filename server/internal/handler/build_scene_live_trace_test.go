package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Opt-in provider diagnostics retain timing/status only, never request headers,
// credentials, response bodies or private thinking content.
func traceSceneLiveRequest(request *http.Request, sample string, id int32) (*http.Request, func(int, error)) {
	type milestone struct {
		Stage        string `json:"stage"`
		Milliseconds int64  `json:"milliseconds"`
	}
	started := time.Now()
	var mu sync.Mutex
	stages := []milestone{}
	mark := func(stage string) {
		mu.Lock()
		defer mu.Unlock()
		if len(stages) < 32 {
			stages = append(stages, milestone{stage, time.Since(started).Milliseconds()})
		}
	}
	trace := &httptrace.ClientTrace{
		GotConn:              func(httptrace.GotConnInfo) { mark("connection_ready") },
		WroteRequest:         func(httptrace.WroteRequestInfo) { mark("request_written") },
		GotFirstResponseByte: func() { mark("first_response_byte") },
	}
	return request.WithContext(httptrace.WithClientTrace(request.Context(), trace)), func(status int, err error) {
		output := os.Getenv("CHIMII_SCENE_LIVE_OUTPUT")
		if output == "" {
			return
		}
		mu.Lock()
		timeline := append([]milestone{}, stages...)
		mu.Unlock()
		data, _ := json.MarshalIndent(struct {
			Sample           string      `json:"sample"`
			RequestID        int32       `json:"request_id"`
			Host             string      `json:"host"`
			HTTPStatus       int         `json:"http_status"`
			Elapsed          int64       `json:"elapsed_ms"`
			DeadlineExceeded bool        `json:"deadline_exceeded"`
			ErrorType        string      `json:"error_type,omitempty"`
			Stages           []milestone `json:"stages"`
		}{sample, id, request.URL.Hostname(), status, time.Since(started).Milliseconds(), errors.Is(err, context.DeadlineExceeded), fmt.Sprintf("%T", err), timeline}, "", "  ")
		_ = os.WriteFile(filepath.Join(output, fmt.Sprintf("%s-transport-%d.json", sample, id)), data, 0600)
	}
}
