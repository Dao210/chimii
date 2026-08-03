package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	buildWorkerPollInterval = time.Second
	// The durable SKIP LOCKED queue is safe across goroutines and server
	// replicas. A small local pool prevents one slow LLM request from consuming
	// the entire child-facing latency budget for unrelated families.
	buildWorkerConcurrency = 4
)

type BuildWorker struct {
	h      *Handler
	notify chan struct{}
	done   chan struct{}
}

func NewBuildWorker(h *Handler) *BuildWorker {
	return &BuildWorker{h: h, notify: make(chan struct{}, 1), done: make(chan struct{})}
}

func (w *BuildWorker) Notify() {
	if w == nil {
		return
	}
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func (w *BuildWorker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	defer close(w.done)
	var workers sync.WaitGroup
	workers.Add(buildWorkerConcurrency)
	for range buildWorkerConcurrency {
		go func() {
			defer workers.Done()
			w.runLoop(ctx)
		}()
	}
	workers.Wait()
}

func (w *BuildWorker) runLoop(ctx context.Context) {
	ticker := time.NewTicker(buildWorkerPollInterval)
	defer ticker.Stop()
	for {
		worked, err := w.ProcessNext(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("build worker: process job", "error", err)
		}
		if worked {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-w.notify:
		case <-ticker.C:
		}
	}
}

func (w *BuildWorker) WaitWithTimeout(timeout time.Duration) bool {
	if w == nil {
		return true
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-w.done:
		return true
	case <-timer.C:
		return false
	}
}

func (w *BuildWorker) ProcessNext(ctx context.Context) (bool, error) {
	job, err := w.h.Queries.ClaimBuildJob(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim build job: %w", err)
	}
	session, err := w.h.Queries.GetBuildSessionForWorker(ctx, job.SessionID)
	if err != nil {
		return true, w.retry(ctx, job, fmt.Errorf("load session: %w", err))
	}
	if uuidToString(session.WorkspaceID) != uuidToString(job.WorkspaceID) {
		return true, w.retry(ctx, job, errors.New("build job ownership mismatch"))
	}
	if session.ExpiresAt.Valid && session.ExpiresAt.Time.Before(time.Now()) {
		_ = w.h.Queries.FailBuildSession(ctx, db.FailBuildSessionParams{ID: session.ID, Error: pgtype.Text{String: "build session expired before processing", Valid: true}})
		_, err := w.h.Queries.CompleteBuildJob(ctx, db.CompleteBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return true, fmt.Errorf("complete expired build job: %w", err)
		}
		return true, nil
	}
	if err := w.h.Queries.MarkBuildSessionGenerating(ctx, session.ID); err != nil {
		return true, w.retry(ctx, job, fmt.Errorf("mark generating: %w", err))
	}
	answers := map[string]string{}
	if len(session.Answers) > 0 {
		_ = json.Unmarshal(session.Answers, &answers)
	}
	recipe, err := w.h.planBuildRecipe(ctx, session.Prompt, answers)
	if err != nil {
		return true, w.retry(ctx, job, fmt.Errorf("plan build intent: %w", err))
	}
	result, err := buildstudio.Compile(recipe, time.Now())
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	recipeJSON, _ := json.Marshal(result.Recipe)
	planJSON, _ := json.Marshal(result.Plan)
	validationJSON, _ := json.Marshal(result.Plan.Validation)

	tx, err := w.h.TxStarter.Begin(ctx)
	if err != nil {
		return true, w.retry(ctx, job, fmt.Errorf("begin completion: %w", err))
	}
	defer tx.Rollback(ctx)
	qtx := w.h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(ctx, qtx, session.WorkspaceID, session.CreatorUserID); err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			// Workspace/member cleanup removes the session and job in the same
			// transaction. These best-effort updates also converge any orphan
			// left by a deployment from before the explicit lock protocol.
			_ = w.h.Queries.FailBuildSession(ctx, db.FailBuildSessionParams{
				ID: session.ID, Error: pgtype.Text{String: "build owner scope no longer exists", Valid: true},
			})
			_, completeErr := w.h.Queries.CompleteBuildJob(ctx, db.CompleteBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken})
			if completeErr != nil && !errors.Is(completeErr, pgx.ErrNoRows) {
				return true, fmt.Errorf("complete orphaned build job: %w", completeErr)
			}
			return true, nil
		}
		return true, w.retry(ctx, job, fmt.Errorf("lock build owner scope: %w", err))
	}
	creation, err := qtx.CreateBuildCreation(ctx, db.CreateBuildCreationParams{
		WorkspaceID: session.WorkspaceID, CreatorUserID: session.CreatorUserID, ChildProfileID: session.ChildProfileID, SessionID: session.ID,
		Title: recipe.Title, Prompt: session.Prompt, Archetype: recipe.Archetype,
		Recipe: recipeJSON, BuildPlan: planJSON, Validation: validationJSON, LdrawMpd: result.MPD,
	})
	if err != nil {
		_ = tx.Rollback(ctx)
		return true, w.retry(ctx, job, fmt.Errorf("save creation: %w", err))
	}
	if _, err := qtx.CompleteBuildSession(ctx, db.CompleteBuildSessionParams{CreationID: creation.ID, ID: session.ID}); err != nil {
		_ = tx.Rollback(ctx)
		return true, w.retry(ctx, job, fmt.Errorf("complete session: %w", err))
	}
	if _, err := qtx.CompleteBuildJob(ctx, db.CompleteBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken}); err != nil {
		_ = tx.Rollback(ctx)
		return true, w.retry(ctx, job, fmt.Errorf("complete job: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		return true, fmt.Errorf("commit build completion: %w", err)
	}
	return true, nil
}

func (w *BuildWorker) retry(ctx context.Context, job db.BuildJob, cause error) error {
	next := time.Now().Add(time.Duration(1<<min(job.Attempts, 5)) * time.Second)
	updated, err := w.h.Queries.RetryBuildJob(ctx, db.RetryBuildJobParams{
		AvailableAt: pgtype.Timestamptz{Time: next, Valid: true}, LastError: pgtype.Text{String: cause.Error(), Valid: true},
		ID: job.ID, LeaseToken: job.LeaseToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("retry build job after %v: %w", cause, err)
	}
	if updated.Status == "failed" {
		_ = w.h.Queries.FailBuildSession(ctx, db.FailBuildSessionParams{ID: job.SessionID, Error: pgtype.Text{String: cause.Error(), Valid: true}})
	}
	return nil
}
