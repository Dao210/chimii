package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/chimii-ai/chimii/server/internal/util"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func buildRequestHash(v any) string {
	b := mustBuildJSON(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type buildMessageResponse struct {
	ID        string          `json:"id"`
	Sequence  int64           `json:"sequence"`
	SessionID string          `json:"session_id"`
	Role      string          `json:"role"`
	Kind      string          `json:"kind"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt string          `json:"created_at"`
}

func buildMessageResponseOf(v db.BuildMessage) buildMessageResponse {
	return buildMessageResponse{uuidToString(v.ID), v.Sequence, uuidToString(v.SessionID), v.Role, v.Kind, v.Content, v.Metadata, v.CreatedAt.Time.UTC().Format(time.RFC3339)}
}
func appendBuildUserMessage(ctx context.Context, q *db.Queries, s db.BuildSession, content, event, hash string) error {
	_, err := q.AppendBuildMessage(ctx, db.AppendBuildMessageParams{ConversationID: s.ConversationID, SessionID: s.ID, Role: "user", Kind: "message", Content: content, EventKey: event, RequestHash: hash, Metadata: mustBuildJSON(map[string]any{"revision": s.Revision})})
	return err
}

// Called inside the same fenced transaction as the outcome. Polling never
// appends messages; one deterministic event represents a committed outcome.
func appendBuildOutcome(ctx context.Context, q *db.Queries, id pgtype.UUID) error {
	s, err := q.GetBuildSessionForWorker(ctx, id)
	if err != nil {
		return err
	}
	if s.Status != "completed" && s.Status != "clarifying" && s.Status != "failed" {
		return nil
	}
	v := toBuildSessionResponse(s)
	kind, content := "message", v.Summary
	if s.Status == "clarifying" && v.Question != nil {
		kind, content = "question", v.Question.Prompt
	}
	if s.Status == "failed" {
		kind = "error"
		content = v.Message
		if content == "" {
			content = v.Error
		}
	}
	if s.CreationID.Valid || s.CircuitCreationID.Valid {
		kind = "result"
	}
	_, err = q.AppendBuildMessage(ctx, db.AppendBuildMessageParams{ConversationID: s.ConversationID, SessionID: s.ID, Role: "assistant", Kind: kind, Content: content, EventKey: fmt.Sprintf("session:%s:%d:%s", uuidToString(s.ID), s.Revision, s.Status), Metadata: mustBuildJSON(v)})
	return err
}

func (h *Handler) GetBuildConversation(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "conversationID"), "conversation")
	if !ok {
		return
	}
	// Reads use the same parent-management policy as existing Build artifacts.
	root, err := h.Queries.GetBuildSessionInWorkspace(r.Context(), db.GetBuildSessionInWorkspaceParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child})
	if err != nil || root.ConversationID != id {
		writeError(w, 404, "conversation not found")
		return
	}
	before := int64(0)
	if value := r.URL.Query().Get("before"); value != "" {
		before, err = strconv.ParseInt(value, 10, 64)
		if err != nil || before <= 0 {
			writeError(w, 400, "invalid cursor")
			return
		}
	}
	latest, err := h.Queries.LatestBuildConversationSession(r.Context(), id)
	if err != nil {
		writeError(w, 500, "could not load conversation")
		return
	}
	rows, err := h.Queries.ListBuildMessages(r.Context(), db.ListBuildMessagesParams{ConversationID: id, BeforeSequence: before, PageSize: 51})
	if err != nil {
		writeError(w, 500, "could not load messages")
		return
	}
	next := ""
	if len(rows) > 50 {
		rows = rows[:50]
		next = strconv.FormatInt(rows[49].Sequence, 10)
	}
	items := make([]buildMessageResponse, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		items = append(items, buildMessageResponseOf(rows[i]))
	}
	var result *buildSessionResponse
	if row, e := h.Queries.LatestBuildConversationResult(r.Context(), id); e == nil {
		v := toBuildSessionResponse(row)
		result = &v
	} else if !errors.Is(e, pgx.ErrNoRows) {
		writeError(w, 500, "could not load result")
		return
	}
	writeJSON(w, 200, map[string]any{"id": uuidToString(id), "session": toBuildSessionResponse(latest), "result": result, "messages": items, "next_cursor": next})
}

// The caller holds the actor advisory lock and membership lock. Continuations
// use exact actor scope, even when a parent may read a child's old conversation.
func (h *Handler) continueBuildRequest(w http.ResponseWriter, r *http.Request, q *db.Queries, req createBuildSessionRequest, ws, user, child pgtype.UUID, hash string) (pgtype.UUID, *db.BuildSession, bool) {
	pathID := chi.URLParam(r, "conversationID")
	if pathID == "" {
		return pgtype.UUID{}, nil, true
	}
	id, err := util.ParseUUID(pathID)
	if err != nil {
		writeError(w, 400, "invalid conversation")
		return id, nil, false
	}
	root, err := q.LockBuildSessionForAnswer(r.Context(), db.LockBuildSessionForAnswerParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child})
	if err != nil || root.ConversationID != id {
		writeError(w, 404, "conversation not found")
		return id, nil, false
	}
	if prior, e := q.GetBuildMessageByEvent(r.Context(), db.GetBuildMessageByEventParams{ConversationID: id, EventKey: "request:" + req.ClientRequestID}); e == nil {
		if prior.RequestHash != hash {
			writeError(w, 409, "request ID already used for different content")
			return id, nil, false
		}
		s, e := q.GetBuildSessionForWorker(r.Context(), prior.SessionID)
		if e != nil {
			writeError(w, 500, "could not replay message")
			return id, nil, false
		}
		writeJSON(w, 202, toBuildSessionResponse(s))
		return id, nil, false
	} else if !errors.Is(e, pgx.ErrNoRows) {
		writeError(w, 500, "could not check message")
		return id, nil, false
	}
	latest, err := q.LatestBuildConversationSession(r.Context(), id)
	if err != nil {
		writeError(w, 500, "could not load conversation")
		return id, nil, false
	}
	if req.ExpectedSessionID != uuidToString(latest.ID) || req.ExpectedRevision != latest.Revision {
		writeError(w, 409, "conversation changed; refresh before sending")
		return id, nil, false
	}
	if latest.ExpiresAt.Time.After(time.Now()) && (latest.Status == "queued" || latest.Status == "generating") {
		writeError(w, 409, "conversation is busy; wait or cancel")
		return id, nil, false
	}
	return id, &latest, true
}

func (h *Handler) ListInventions(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	p := db.ListInventionSummariesParams{WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child, ActorKey: circuitActorKey(child), PageSize: 41}
	if cursor := r.URL.Query().Get("before"); cursor != "" {
		var v struct {
			Time time.Time
			ID   string
			Kind string
		}
		if json.Unmarshal([]byte(cursor), &v) != nil {
			writeError(w, 400, "invalid cursor")
			return
		}
		id, err := util.ParseUUID(v.ID)
		if err != nil || v.Time.IsZero() || (v.Kind != "brick" && v.Kind != "circuit") {
			writeError(w, 400, "invalid cursor")
			return
		}
		p.BeforeTime = pgtype.Timestamptz{Time: v.Time, Valid: true}
		p.BeforeID = id
		p.BeforeKind = v.Kind
	}
	rows, err := h.Queries.ListInventionSummaries(r.Context(), p)
	if err != nil {
		writeError(w, 500, "could not load creations")
		return
	}
	next := ""
	if len(rows) > 40 {
		rows = rows[:40]
		v := rows[39]
		next = string(mustBuildJSON(map[string]any{"Time": v.CreatedAt.Time, "ID": uuidToString(v.ID), "Kind": v.Kind}))
	}
	writeJSON(w, 200, map[string]any{"creations": rows, "next_cursor": next})
}
