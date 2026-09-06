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
	"github.com/chimii-ai/chimii/server/internal/util"
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
	history, historyErr := w.conversationHistory(runCtx, session)
	if historyErr != nil {
		return true, w.retry(ctx, job, historyErr)
	}
	if session.Kind == "auto" {
		decision, routeErr := w.routeBuild(runCtx, session, answers, history)
		if routeErr != nil {
			return true, w.retry(ctx, job, routeErr)
		}
		if decision.Kind == "clarify" {
			if session.Revision > maxBuildClarifications {
				return true, w.finishReply(ctx, job, session, decision.Question, true)
			}
			var plan circuitBuildPlan
			_ = json.Unmarshal(session.Recipe, &plan)
			return true, w.circuitQuestion(ctx, job, session, plan, decision.Question)
		}
		recipe, resolveErr := routedBuildRecipe(runCtx, w.h.Queries, session, decision.Kind)
		if resolveErr != nil {
			return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, resolveErr)
		}
		if err = w.withLease(runCtx, job, func(q *db.Queries) error {
			var e error
			session, e = q.SetBuildSessionKind(runCtx, db.SetBuildSessionKindParams{ID: session.ID, Kind: decision.Kind, Recipe: recipe})
			return e
		}); err != nil {
			return true, w.retry(ctx, job, err)
		}
	}
	if session.Kind == "circuit" {
		return true, w.processCircuit(runCtx, job, session, answers, history)
	}
	// Availability is read for this execution, not frozen at request creation.
	// No inventory row lock, reservation or consumption is part of generation.
	inventory, catalog, err := w.readAvailability(runCtx, session.WorkspaceID)
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	var recipe *buildstudio.AssemblyRecipe
	if len(session.Recipe) > 0 {
		if err = json.Unmarshal(session.Recipe, &recipe); err != nil {
			return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, err)
		}
	}
	planned := session.Phase != "compiling" || recipe == nil
	if planned {
		// A clarification draft deliberately has no executable design. Restore
		// the immutable source as planning context when continuing an edit.
		if recipe != nil && recipe.Design == nil && recipe.Metadata["parent_creation_id"] != "" {
			parentID, parseErr := util.ParseUUID(recipe.Metadata["parent_creation_id"])
			if parseErr != nil {
				return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, parseErr)
			}
			parent, loadErr := w.h.Queries.GetBuildCreationInWorkspace(runCtx, db.GetBuildCreationInWorkspaceParams{ID: parentID, WorkspaceID: session.WorkspaceID, CreatorUserID: session.CreatorUserID, ChildProfileID: session.ChildProfileID})
			if loadErr != nil {
				return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, loadErr)
			}
			var base buildstudio.AssemblyRecipe
			if decodeErr := json.Unmarshal(parent.Recipe, &base); decodeErr != nil {
				return true, w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, decodeErr)
			}
			recipe.Design = base.Design
		}
		planStarted := time.Now()
		decision, planErr := w.h.planBuildRecipe(runCtx, session.Prompt, answers, recipe, session.Revision, inventory, catalog, history)
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
		case "reply":
			return true, w.finishReply(ctx, job, session, decision.Message, false)
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
		if recipe.Design != nil {
			recipe.Metadata["shape_generator_version"] = buildstudio.ShapeGeneratorVersion
		}
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
	if recipe.Design != nil && recipe.Metadata["shape_generator_version"] != buildstudio.ShapeGeneratorVersion {
		return true, w.failPermanently(ctx, job, buildstudio.BuildErrorUnsupported, errors.New("saved shape generator version is unavailable"))
	}
	if planned {
		// Parts may have been edited during the model call. Compilation always
		// uses a fresh availability read, with no transaction held across planning.
		inventory, catalog, err = w.readAvailability(runCtx, session.WorkspaceID)
		if err != nil {
			return true, w.retry(ctx, job, err)
		}
	}
	compileStarted := time.Now()
	var result buildstudio.CompileResult
	if recipe.Design != nil {
		result, err = buildstudio.CompileDesign(runCtx, *recipe, inventory, inventory.CatalogVersion, catalog, time.Now())
	} else {
		result, err = buildstudio.CompileWithCatalog(*recipe, inventory, inventory.CatalogVersion, catalog, time.Now())
	}
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
			Title: recipe.Title, Prompt: session.Prompt, Archetype: recipe.Archetype, Recipe: mustBuildJSON(result.Recipe), BuildPlan: mustBuildJSON(result.Plan), Validation: mustBuildJSON(result.Plan.Validation), LdrawMpd: result.MPD, InventorySnapshot: []byte(`{}`),
		})
		if err != nil {
			return err
		}
		_, err = q.CompleteBuildSession(ctx, db.CompleteBuildSessionParams{ID: session.ID, CreationID: creation.ID})
		return err
	})
}

// A short read-only transaction keeps the availability rows consistent without
// taking the inventory writer lock or persisting a per-session stock snapshot.
func (w *BuildWorker) readAvailability(ctx context.Context, workspaceID pgtype.UUID) (buildstudio.InventorySnapshot, buildstudio.PartCatalog, error) {
	tx, err := w.h.TxStarter.Begin(ctx)
	if err != nil {
		return buildstudio.InventorySnapshot{}, nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY"); err != nil {
		return buildstudio.InventorySnapshot{}, nil, err
	}
	q := w.h.Queries.WithTx(tx)
	availability, err := loadBrickInventorySnapshot(ctx, q, workspaceID)
	if err != nil {
		return availability, nil, err
	}
	catalog, err := loadBuildCatalogParts(ctx, q, availability.CatalogVersion)
	if err != nil {
		return availability, nil, err
	}
	return availability, catalog, tx.Commit(ctx)
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
	if err = appendBuildOutcome(ctx, q, job.SessionID); err != nil {
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
	session, loadErr := q.GetBuildSessionForWorker(ctx, job.SessionID)
	if loadErr != nil {
		return loadErr
	}
	if err = lockWorkspaceMemberForScopedWrite(ctx, q, session.WorkspaceID, session.CreatorUserID); err != nil {
		return err
	}
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
	if err = appendBuildOutcome(ctx, q, job.SessionID); err != nil {
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
	session, loadErr := q.GetBuildSessionForWorker(ctx, job.SessionID)
	if loadErr != nil {
		return loadErr
	}
	if err = lockWorkspaceMemberForScopedWrite(ctx, q, session.WorkspaceID, session.CreatorUserID); err != nil {
		return err
	}
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
	if updated.Status == "failed" {
		if err = appendBuildOutcome(ctx, q, job.SessionID); err != nil {
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
