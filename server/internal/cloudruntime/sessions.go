package cloudruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/redact"
	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrInvalidSessionID = errors.New("invalid cloud runtime session id")
	ErrSessionScope     = errors.New("cloud runtime session belongs to a different runtime or agent")
	ErrSessionCorrupt   = errors.New("cloud runtime session is corrupt")
	ErrSessionNotFound  = errors.New("cloud runtime session not found")
)

type SessionSpec struct {
	WorkspaceID pgtype.UUID
	RuntimeID   pgtype.UUID
	AgentID     pgtype.UUID
	Provider    string
	Model       string
	ContextType string
	ContextID   string
}

type DurableSession struct {
	Record   db.CloudRuntimeSession
	Messages []sdktypes.Message
}

type sessionQueries interface {
	CreateCloudRuntimeSession(context.Context, db.CreateCloudRuntimeSessionParams) (db.CloudRuntimeSession, error)
	GetCloudRuntimeSessionForRuntime(context.Context, db.GetCloudRuntimeSessionForRuntimeParams) (db.CloudRuntimeSession, error)
	ListCloudRuntimeSessionMessages(context.Context, string) ([]db.CloudRuntimeSessionMessage, error)
	AppendCloudRuntimeSessionMessage(context.Context, db.AppendCloudRuntimeSessionMessageParams) (db.AppendCloudRuntimeSessionMessageRow, error)
	SetCloudRuntimeSessionStatus(context.Context, db.SetCloudRuntimeSessionStatusParams) (db.CloudRuntimeSession, error)
}

type SessionStore struct {
	queries sessionQueries
}

func NewSessionStore(queries *db.Queries) *SessionStore {
	return &SessionStore{queries: queries}
}

func newSessionID() string {
	return "crs_" + uuid.NewString()
}

func validSessionID(id string) bool {
	if !strings.HasPrefix(id, "crs_") {
		return false
	}
	_, err := uuid.Parse(strings.TrimPrefix(id, "crs_"))
	return err == nil
}

func (s *SessionStore) Open(ctx context.Context, resumeID string, spec SessionSpec) (*DurableSession, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("session store is unavailable")
	}
	if resumeID == "" {
		record, err := s.queries.CreateCloudRuntimeSession(ctx, db.CreateCloudRuntimeSessionParams{
			ID:          newSessionID(),
			WorkspaceID: spec.WorkspaceID,
			RuntimeID:   spec.RuntimeID,
			AgentID:     spec.AgentID,
			Provider:    spec.Provider,
			Model:       spec.Model,
			ContextType: spec.ContextType,
			ContextID:   spec.ContextID,
		})
		if err != nil {
			return nil, fmt.Errorf("create cloud runtime session: %w", err)
		}
		return &DurableSession{Record: record, Messages: []sdktypes.Message{}}, nil
	}
	if !validSessionID(resumeID) {
		return nil, ErrInvalidSessionID
	}
	record, err := s.queries.GetCloudRuntimeSessionForRuntime(ctx, db.GetCloudRuntimeSessionForRuntimeParams{
		ID:        resumeID,
		RuntimeID: spec.RuntimeID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("load cloud runtime session: %w", err)
	}
	if record.AgentID != spec.AgentID || record.WorkspaceID != spec.WorkspaceID ||
		record.Provider != spec.Provider || record.ContextType != spec.ContextType || record.ContextID != spec.ContextID {
		return nil, ErrSessionScope
	}
	rows, err := s.queries.ListCloudRuntimeSessionMessages(ctx, record.ID)
	if err != nil {
		return nil, fmt.Errorf("list cloud runtime session messages: %w", err)
	}
	messages := make([]sdktypes.Message, 0, len(rows))
	for _, row := range rows {
		var message sdktypes.Message
		if err := json.Unmarshal(row.Payload, &message); err != nil {
			return nil, fmt.Errorf("%w at seq %d: %v", ErrSessionCorrupt, row.Seq, err)
		}
		messages = append(messages, message)
	}
	return &DurableSession{Record: record, Messages: messages}, nil
}

func (s *SessionStore) Append(ctx context.Context, sessionID string, message sdktypes.Message) error {
	if !validSessionID(sessionID) {
		return ErrInvalidSessionID
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal cloud runtime session message: %w", err)
	}
	if _, err := s.queries.AppendCloudRuntimeSessionMessage(ctx, db.AppendCloudRuntimeSessionMessageParams{
		SessionID: sessionID,
		Role:      message.Role,
		Payload:   payload,
	}); err != nil {
		return fmt.Errorf("append cloud runtime session message: %w", err)
	}
	return nil
}

func (s *SessionStore) MarkFailed(ctx context.Context, sessionID string, cause error) error {
	if !validSessionID(sessionID) {
		return ErrInvalidSessionID
	}
	message := ""
	if cause != nil {
		message = redact.Text(cause.Error())
		if len(message) > 1000 {
			message = message[:1000]
		}
	}
	_, err := s.queries.SetCloudRuntimeSessionStatus(ctx, db.SetCloudRuntimeSessionStatusParams{
		ID:        sessionID,
		Status:    "failed",
		LastError: pgtype.Text{String: message, Valid: message != ""},
	})
	return err
}

func (s *SessionStore) MarkCompleted(ctx context.Context, sessionID string) error {
	if !validSessionID(sessionID) {
		return ErrInvalidSessionID
	}
	_, err := s.queries.SetCloudRuntimeSessionStatus(ctx, db.SetCloudRuntimeSessionStatusParams{
		ID:     sessionID,
		Status: "completed",
	})
	return err
}

func (s *SessionStore) MarkCancelled(ctx context.Context, sessionID string) error {
	if !validSessionID(sessionID) {
		return ErrInvalidSessionID
	}
	_, err := s.queries.SetCloudRuntimeSessionStatus(ctx, db.SetCloudRuntimeSessionStatusParams{
		ID:     sessionID,
		Status: "cancelled",
	})
	return err
}
