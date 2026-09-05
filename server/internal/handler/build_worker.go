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
	started := time.Now()
	defer func() {
		slog.Info("build worker: execution finished", "job_id", uuidToString(job.ID), "elapsed_ms", time.Since(started).Milliseconds())
	}()
	// Leave headroom for a final fenced transaction before the 60-second lease ends.
	runCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	session, err := w.h.Queries.GetBuildSessionForWorker(runCtx, job.SessionID)
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	if session.WorkspaceID != job.WorkspaceID {
		return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, errors.New("job ownership mismatch"))
	}
	if session.ExpiresAt.Valid && session.ExpiresAt.Time.Before(time.Now()) {
		return true, w.failPermanently(ctx, job, "BUILD_GENERATION_FAILED", errors.New("session expired"))
	}
	err = w.withLease(runCtx, job, func(q *db.Queries) error {
		var markErr error
		session, markErr = q.MarkBuildSessionGenerating(runCtx, db.MarkBuildSessionGeneratingParams{ID: session.ID, Revision: session.Revision, LeaseToken: job.LeaseToken})
		return markErr
	})
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = w.h.Queries.CompleteBuildJob(ctx, db.CompleteBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken})
		if errors.Is(err, pgx.ErrNoRows) {
			err = nil
		}
		return true, err
	}
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	answers := map[string]string{}
	if err = json.Unmarshal(session.Answers, &answers); err != nil {
		return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, err)
	}
	var inventory buildstudio.InventorySnapshot
	if err = json.Unmarshal(session.InventorySnapshot, &inventory); err != nil {
		return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, err)
	}
	catalog, err := loadBuildCatalogParts(runCtx, w.h.Queries, inventory.CatalogVersion)
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	var recipe *buildstudio.AssemblyRecipe
	if len(session.Recipe) > 0 {
		if err = json.Unmarshal(session.Recipe, &recipe); err != nil {
			return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, err)
		}
	}
	if session.Phase != "compiling" || recipe == nil {
		planStarted := time.Now()
		decision, planErr := w.h.planBuildRecipe(runCtx, session.Prompt, answers, recipe, session.Revision, inventory, catalog)
		slog.Info("build worker: planning", "session_id", uuidToString(session.ID), "revision", session.Revision, "elapsed_ms", time.Since(planStarted).Milliseconds(), "outcome", decision.Outcome)
		if planErr != nil {
			if code, ok := buildstudio.BuildErrorCode(planErr); ok {
				return true, w.failPermanently(ctx, job, code, planErr)
			}
			return true, w.retry(ctx, job, planErr)
		}
		switch decision.Outcome {
		case "clarify":
			return true, w.pause(ctx, job, session, decision)
		case "unsupported":
			draft := buildstudio.AssemblyRecipe{Version: 2, Summary: decision.Message}
			return true, w.finish(ctx, job, func(q *db.Queries) error {
				if _, err := q.SaveBuildSessionMessage(ctx, db.SaveBuildSessionMessageParams{ID: session.ID, Revision: session.Revision, Recipe: mustBuildJSON(draft)}); err != nil {
					return err
				}
				return q.FailBuildSession(ctx, db.FailBuildSessionParams{ID: session.ID, Error: pgtype.Text{String: buildstudio.BuildErrorUnsupported, Valid: true}})
			})
		case "ready":
			recipe = decision.Recipe
		default:
			return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, errors.New("invalid decision"))
		}
		recipe.Metadata["module_library_version"] = buildstudio.ModuleLibraryVersion
		if err = w.withLease(runCtx, job, func(q *db.Queries) error {
			_, saveErr := q.SaveBuildSessionRecipe(runCtx, db.SaveBuildSessionRecipeParams{ID: session.ID, Revision: session.Revision, LeaseToken: job.LeaseToken, Recipe: mustBuildJSON(recipe)})
			return saveErr
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return true, nil
			}
			return true, w.retry(ctx, job, err)
		}
	}
	if recipe.Metadata["module_library_version"] != buildstudio.ModuleLibraryVersion {
		return true, w.failPermanently(ctx, job, buildstudio.BuildErrorUnsupported, errors.New("saved module version is unavailable"))
	}
	compileStarted := time.Now()
	result, err := buildstudio.CompileWithCatalog(*recipe, inventory, inventory.CatalogVersion, catalog, time.Now())
	slog.Info("build worker: compile", "session_id", uuidToString(session.ID), "elapsed_ms", time.Since(compileStarted).Milliseconds())
	if err != nil {
		if code, ok := buildstudio.BuildErrorCode(err); ok {
			return true, w.failPermanently(ctx, job, code, err)
		}
		return true, w.retry(ctx, job, err)
	}
	return true, w.finish(ctx, job, func(q *db.Queries) error {
		creation, err := q.CreateBuildCreation(ctx, db.CreateBuildCreationParams{
			WorkspaceID: session.WorkspaceID, CreatorUserID: session.CreatorUserID, ChildProfileID: session.ChildProfileID, SessionID: session.ID,
			Title: recipe.Title, Prompt: session.Prompt, Archetype: recipe.Archetype, Recipe: mustBuildJSON(result.Recipe), BuildPlan: mustBuildJSON(result.Plan), Validation: mustBuildJSON(result.Plan.Validation), LdrawMpd: result.MPD, InventorySnapshot: session.InventorySnapshot,
		})
		if err != nil {
			return err
		}
		_, err = q.CompleteBuildSession(ctx, db.CompleteBuildSessionParams{ID: session.ID, CreationID: creation.ID})
		return err
	})
}

// All terminal writes first consume a live lease in the same transaction. A
// stale worker can neither overwrite a new question nor create a second artifact.
func (w *BuildWorker) finish(ctx context.Context, job db.BuildJob, apply func(*db.Queries) error) error {
	tx, err := w.h.TxStarter.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := w.h.Queries.WithTx(tx)
	session, err := q.GetBuildSessionForWorker(ctx, job.SessionID)
	if err != nil {
		return err
	}
	if err = lockWorkspaceMemberForScopedWrite(ctx, q, session.WorkspaceID, session.CreatorUserID); err != nil {
		return err
	}
	if _, err = q.LockBuildSessionForWorker(ctx, job.SessionID); err != nil {
		return err
	}
	if _, err = q.CompleteBuildJob(ctx, db.CompleteBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken}); errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if err = apply(q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (w *BuildWorker) pause(ctx context.Context, job db.BuildJob, session db.BuildSession, d buildPlanningDecision) error {
	question := buildstudio.ClarifyingQuestion{ID: fmt.Sprintf("q%d", session.Revision), Prompt: d.Question.Prompt, Choices: d.Question.Choices, Options: []string{}, AllowFreeText: true}
	for _, choice := range question.Choices {
		question.Options = append(question.Options, choice.Label)
	}
	return w.finish(ctx, job, func(q *db.Queries) error {
		_, err := q.PauseBuildSession(ctx, db.PauseBuildSessionParams{ID: session.ID, Revision: session.Revision, Question: mustBuildJSON(question), Recipe: mustBuildJSON(d.Recipe)})
		return err
	})
}
func (w *BuildWorker) failPermanently(ctx context.Context, job db.BuildJob, code string, cause error) error {
	tx, err := w.h.TxStarter.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := w.h.Queries.WithTx(tx)
	if _, err = q.LockBuildSessionForWorker(ctx, job.SessionID); err != nil {
		return err
	}
	if _, err = q.FailBuildJob(ctx, db.FailBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken, LastError: pgtype.Text{String: cause.Error(), Valid: true}}); errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if err = q.FailBuildSession(ctx, db.FailBuildSessionParams{ID: job.SessionID, Error: pgtype.Text{String: code, Valid: true}}); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	slog.Warn("build worker: permanent failure", "job_id", uuidToString(job.ID), "code", code)
	return nil
}
func (w *BuildWorker) retry(ctx context.Context, job db.BuildJob, cause error) error {
	tx, err := w.h.TxStarter.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := w.h.Queries.WithTx(tx)
	if _, err = q.LockBuildSessionForWorker(ctx, job.SessionID); err != nil {
		return err
	}
	updated, err := q.RetryBuildJob(ctx, db.RetryBuildJobParams{ID: job.ID, LeaseToken: job.LeaseToken, AvailableAt: pgtype.Timestamptz{Time: time.Now().Add(time.Duration(1<<min(job.Attempts, 5)) * time.Second), Valid: true}, LastError: pgtype.Text{String: cause.Error(), Valid: true}})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if updated.Status == "failed" {
		if err = q.FailBuildSession(ctx, db.FailBuildSessionParams{ID: job.SessionID, Error: pgtype.Text{String: "BUILD_GENERATION_FAILED", Valid: true}}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func mustBuildJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

// Always lock session before job, matching answer/cancel and terminal writes.
func (w *BuildWorker) withLease(ctx context.Context, job db.BuildJob, apply func(*db.Queries) error) error {
	tx, err := w.h.TxStarter.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := w.h.Queries.WithTx(tx)
	if _, err = q.LockBuildSessionForWorker(ctx, job.SessionID); err != nil {
		return err
	}
	if _, err = q.LockBuildJobLease(ctx, db.LockBuildJobLeaseParams{ID: job.ID, LeaseToken: job.LeaseToken}); err != nil {
		return err
	}
	if err = apply(q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
