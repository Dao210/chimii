package cloudruntime

import (
	"context"
	"errors"
	"testing"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	sdktypes "github.com/codeany-ai/open-agent-sdk-go/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeSessionQueries struct {
	record   db.CloudRuntimeSession
	rows     []db.CloudRuntimeSessionMessage
	getErr   error
	appended []db.AppendCloudRuntimeSessionMessageParams
	statuses []db.SetCloudRuntimeSessionStatusParams
}

func (f *fakeSessionQueries) CreateCloudRuntimeSession(_ context.Context, p db.CreateCloudRuntimeSessionParams) (db.CloudRuntimeSession, error) {
	f.record = db.CloudRuntimeSession{ID: p.ID, WorkspaceID: p.WorkspaceID, RuntimeID: p.RuntimeID, AgentID: p.AgentID, Provider: p.Provider, Model: p.Model, ContextType: p.ContextType, ContextID: p.ContextID, Status: "active"}
	return f.record, nil
}
func (f *fakeSessionQueries) GetCloudRuntimeSessionForRuntime(context.Context, db.GetCloudRuntimeSessionForRuntimeParams) (db.CloudRuntimeSession, error) {
	return f.record, f.getErr
}
func (f *fakeSessionQueries) ListCloudRuntimeSessionMessages(context.Context, string) ([]db.CloudRuntimeSessionMessage, error) {
	return f.rows, nil
}
func (f *fakeSessionQueries) AppendCloudRuntimeSessionMessage(_ context.Context, p db.AppendCloudRuntimeSessionMessageParams) (db.AppendCloudRuntimeSessionMessageRow, error) {
	f.appended = append(f.appended, p)
	return db.AppendCloudRuntimeSessionMessageRow{SessionID: p.SessionID, Payload: p.Payload}, nil
}
func (f *fakeSessionQueries) SetCloudRuntimeSessionStatus(_ context.Context, params db.SetCloudRuntimeSessionStatusParams) (db.CloudRuntimeSession, error) {
	f.statuses = append(f.statuses, params)
	return f.record, nil
}

func testUUID(last byte) pgtype.UUID {
	var value [16]byte
	value[15] = last
	return pgtype.UUID{Bytes: value, Valid: true}
}

func TestSessionStoreCreatesPrefixedSessionAndAppends(t *testing.T) {
	q := &fakeSessionQueries{}
	store := &SessionStore{queries: q}
	session, err := store.Open(context.Background(), "", SessionSpec{
		WorkspaceID: testUUID(1), RuntimeID: testUUID(2), AgentID: testUUID(3),
		Provider: "anthropic", ContextType: "issue", ContextID: "issue-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !validSessionID(session.Record.ID) {
		t.Fatalf("session id = %q", session.Record.ID)
	}
	if err := store.Append(context.Background(), session.Record.ID, sdktypes.Message{Role: "user"}); err != nil {
		t.Fatal(err)
	}
	if len(q.appended) != 1 || q.appended[0].Role != "user" {
		t.Fatalf("appended = %+v", q.appended)
	}
}

func TestSessionStoreRejectsCLIAndCrossRuntimeSessions(t *testing.T) {
	store := &SessionStore{queries: &fakeSessionQueries{getErr: pgx.ErrNoRows}}
	spec := SessionSpec{WorkspaceID: testUUID(1), RuntimeID: testUUID(2), AgentID: testUUID(3), Provider: "openai", ContextType: "chat", ContextID: "chat-1"}
	if _, err := store.Open(context.Background(), "claude-session", spec); !errors.Is(err, ErrInvalidSessionID) {
		t.Fatalf("CLI session error = %v", err)
	}
	if _, err := store.Open(context.Background(), "crs_00000000-0000-0000-0000-000000000001", spec); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cross-runtime error = %v", err)
	}
}

func TestSessionStoreRejectsCorruptMessage(t *testing.T) {
	record := db.CloudRuntimeSession{ID: "crs_00000000-0000-0000-0000-000000000001", WorkspaceID: testUUID(1), RuntimeID: testUUID(2), AgentID: testUUID(3), Provider: "openai", ContextType: "chat", ContextID: "chat-1"}
	store := &SessionStore{queries: &fakeSessionQueries{record: record, rows: []db.CloudRuntimeSessionMessage{{Seq: 1, Payload: []byte("{")}}}}
	_, err := store.Open(context.Background(), record.ID, SessionSpec{WorkspaceID: record.WorkspaceID, RuntimeID: record.RuntimeID, AgentID: record.AgentID, Provider: record.Provider, ContextType: record.ContextType, ContextID: record.ContextID})
	if !errors.Is(err, ErrSessionCorrupt) {
		t.Fatalf("error = %v", err)
	}
}

func TestSessionStoreRedactsFailureAndMarksCancellation(t *testing.T) {
	q := &fakeSessionQueries{}
	store := &SessionStore{queries: q}
	sessionID := "crs_00000000-0000-0000-0000-000000000001"
	secret := "sk-proj-abc123def456ghi789jkl012mno345"
	if err := store.MarkFailed(context.Background(), sessionID, errors.New("provider rejected "+secret)); err != nil {
		t.Fatal(err)
	}
	if len(q.statuses) != 1 || q.statuses[0].Status != "failed" || q.statuses[0].LastError.String == "" {
		t.Fatalf("failure status = %+v", q.statuses)
	}
	if q.statuses[0].LastError.String == "provider rejected "+secret {
		t.Fatal("session failure persisted provider secret")
	}
	if err := store.MarkCancelled(context.Background(), sessionID); err != nil {
		t.Fatal(err)
	}
	if len(q.statuses) != 2 || q.statuses[1].Status != "cancelled" {
		t.Fatalf("cancellation status = %+v", q.statuses)
	}
}
