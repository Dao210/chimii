package ldrawsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	catalogSyncBatchSize    = 20
	catalogSyncPollInterval = 2 * time.Second
)

type CatalogWorker struct {
	pool     *pgxpool.Pool
	queries  *db.Queries
	lock     LockFile
	manifest StarterKitManifest
	client   *http.Client
	notify   chan struct{}
	done     chan struct{}
	doneOnce sync.Once
}

func NewCatalogWorker(pool *pgxpool.Pool, lock LockFile, manifest StarterKitManifest) *CatalogWorker {
	return &CatalogWorker{
		pool: pool, queries: db.New(pool), lock: lock, manifest: manifest,
		client: &http.Client{Timeout: 15 * time.Minute},
		notify: make(chan struct{}, 1), done: make(chan struct{}),
	}
}

func (w *CatalogWorker) Notify() {
	if w == nil {
		return
	}
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func (w *CatalogWorker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	defer w.doneOnce.Do(func() { close(w.done) })
	ticker := time.NewTicker(catalogSyncPollInterval)
	defer ticker.Stop()
	for {
		worked, err := w.ProcessNext(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("ldraw catalog worker: process job", "error", err)
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

func (w *CatalogWorker) WaitWithTimeout(timeout time.Duration) bool {
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

func (w *CatalogWorker) ProcessNext(ctx context.Context) (bool, error) {
	job, err := w.queries.ClaimLDrawCatalogSyncJob(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim LDraw catalog sync job: %w", err)
	}

	archivePath, cleanup, err := DownloadVerifiedArchive(ctx, w.client, w.lock.SourceURL, w.lock.ArchiveSHA256)
	if err != nil {
		return true, w.retry(ctx, job, err)
	}
	defer cleanup()

	progress := func(completed, _ int) error {
		_, err := w.queries.UpdateLDrawCatalogSyncProgress(ctx, db.UpdateLDrawCatalogSyncProgressParams{
			ProgressPartCount: int32(completed), ID: job.ID, LeaseToken: job.LeaseToken,
		})
		return err
	}
	if err := SyncCatalog(ctx, w.pool, w.lock, w.manifest, archivePath, progress); err != nil {
		return true, w.retry(ctx, job, err)
	}
	if _, err := w.queries.CompleteLDrawCatalogSyncJob(ctx, db.CompleteLDrawCatalogSyncJobParams{
		ID: job.ID, LeaseToken: job.LeaseToken,
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return true, fmt.Errorf("complete LDraw catalog sync job: %w", err)
	}
	return true, nil
}

func (w *CatalogWorker) retry(ctx context.Context, job db.LdrawCatalogSyncJob, cause error) error {
	message := cause.Error()
	if len(message) > 2000 {
		message = message[:2000]
	}
	next := time.Now().Add(time.Duration(1<<min(job.Attempts, 5)) * time.Minute)
	_, err := w.queries.RetryLDrawCatalogSyncJob(ctx, db.RetryLDrawCatalogSyncJobParams{
		AvailableAt: pgtype.Timestamptz{Time: next, Valid: true}, Error: pgtype.Text{String: message, Valid: true},
		ID: job.ID, LeaseToken: job.LeaseToken,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("retry LDraw catalog sync after %v: %w", cause, err)
	}
	return cause
}

func DownloadVerifiedArchive(ctx context.Context, client *http.Client, sourceURL, expectedSHA string) (string, func(), error) {
	if strings.TrimSpace(sourceURL) == "" || len(expectedSHA) != 64 {
		return "", func() {}, errors.New("valid LDraw source URL and SHA-256 are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Minute}
	}
	file, err := os.CreateTemp("", "chimii-ldraw-complete-*.zip")
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	cleanup := func() { _ = os.Remove(path) }
	fail := func(err error) (string, func(), error) {
		_ = file.Close()
		cleanup()
		return "", func() {}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fail(err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fail(fmt.Errorf("download LDraw archive: %w", err))
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fail(fmt.Errorf("download LDraw archive: %s", response.Status))
	}
	if _, err := io.Copy(file, response.Body); err != nil {
		return fail(fmt.Errorf("write LDraw archive: %w", err))
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := VerifyArchive(path, expectedSHA); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

func SyncCatalog(
	ctx context.Context,
	pool *pgxpool.Pool,
	lock LockFile,
	manifest StarterKitManifest,
	archivePath string,
	progress func(completed, total int) error,
) error {
	if pool == nil {
		return errors.New("database pool is required")
	}
	if err := VerifyArchive(archivePath, lock.ArchiveSHA256); err != nil {
		return err
	}
	library, err := OpenLibrary(archivePath)
	if err != nil {
		return err
	}
	defer library.Close()

	catalogVersion := composeCatalogVersion(lock.Release, lock.ArchiveSHA256)
	manifestByID := make(map[string]StarterKitPart, len(manifest.Parts))
	for _, part := range manifest.Parts {
		manifestByID[normalizeName(part.LDrawID)] = part
	}
	batch := make([]CompiledPart, 0, catalogSyncBatchSize)
	completed := 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := persistPartBatch(ctx, pool, catalogVersion, batch, manifestByID); err != nil {
			return err
		}
		completed += len(batch)
		batch = batch[:0]
		if progress != nil {
			return progress(completed, manifest.PartCount)
		}
		return nil
	}
	if err := library.CompilePartsStream(manifest.PartIDs(), 0, func(_ int, _ int, part CompiledPart) error {
		batch = append(batch, part)
		if len(batch) >= catalogSyncBatchSize {
			return flush()
		}
		return nil
	}); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}

	releaseJSON, err := json.Marshal(map[string]any{
		"release": lock.Release, "source_url": lock.SourceURL, "archive_sha256": lock.ArchiveSHA256,
		"kit_id": manifest.KitID, "part_count": manifest.PartCount, "selection": json.RawMessage(manifest.Selection),
	})
	if err != nil {
		return fmt.Errorf("marshal LDraw release metadata: %w", err)
	}
	return activateCatalogRelease(ctx, pool, lock, manifest, catalogVersion, string(releaseJSON))
}

func persistPartBatch(
	ctx context.Context,
	pool *pgxpool.Pool,
	catalogVersion string,
	parts []CompiledPart,
	manifestByID map[string]StarterKitPart,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	for _, part := range parts {
		manifestPart, ok := manifestByID[normalizeName(part.PartID)]
		if !ok {
			return fmt.Errorf("part %s is missing from Starter Kit manifest", part.PartID)
		}
		semantics := DerivePartSemantics(manifestPart)
		if err := semantics.Validate(); err != nil {
			return fmt.Errorf("derive semantics for %s: %w", part.PartID, err)
		}
		storageKey := filepath.ToSlash(filepath.Join("ldraw", catalogVersion, part.PartID+".glb"))
		if _, err := queries.UpsertLDrawPartRevision(ctx, db.UpsertLDrawPartRevisionParams{
			CatalogVersion: catalogVersion, PartID: part.PartID, Revision: part.Revision, Kind: part.Kind,
			LdrawSha256: part.LDrawSHA256, StorageBackend: "db",
			ContentSha256: pgtype.Text{String: part.ContentSHA256, Valid: true}, ContentType: part.ContentType,
			StorageKey: storageKey, Payload: part.Content, PayloadSizeBytes: part.PayloadSize, PayloadFormat: 2,
		}); err != nil {
			return fmt.Errorf("upsert LDraw part %s: %w", part.PartID, err)
		}
		if err := queries.UpsertPartDefinition(ctx, db.UpsertPartDefinitionParams{
			PartKey: semantics.PartKey, Name: semantics.Name, Category: semantics.Category,
			PopularityRank: int32(semantics.PopularityRank), CertificationLevel: semantics.CertificationLevel,
			AutoBuildEligible: semantics.AutoBuildEligible, GeometryProfile: semantics.GeometryProfile,
			StudsX: int32(semantics.StudsX), StudsZ: int32(semantics.StudsZ), PlatesY: int32(semantics.PlatesY),
			DefaultQuantity: int32(semantics.DefaultQuantity), HasTopStuds: semantics.HasTopStuds,
			HasBottomReceptors: semantics.HasBottomReceptors,
		}); err != nil {
			return fmt.Errorf("upsert part definition %s: %w", semantics.PartKey, err)
		}
		if err := queries.UpsertPartCatalogRevision(ctx, db.UpsertPartCatalogRevisionParams{
			CatalogVersion: catalogVersion, PartKey: semantics.PartKey, LdrawPartID: part.PartID,
			LdrawSha256:     part.LDrawSHA256,
			ContentSha256:   pgtype.Text{String: part.ContentSHA256, Valid: part.ContentSHA256 != ""},
			SemanticVersion: int32(PartSemanticVersion), OriginYOffsetLdu: int32(semantics.OriginYOffsetLDU),
			OriginCenterZOffsetLdu: int32(semantics.OriginCenterZOffsetLDU), BoundsJson: BoundsJSON(part.Bounds),
			ConnectionsJson: semantics.ConnectionsJSON(), OccupancyJson: semantics.OccupancyJSON(),
		}); err != nil {
			return fmt.Errorf("upsert part catalog revision %s: %w", semantics.PartKey, err)
		}
	}
	return tx.Commit(ctx)
}

func activateCatalogRelease(ctx context.Context, pool *pgxpool.Pool, lock LockFile, manifest StarterKitManifest, catalogVersion, releaseJSON string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	if err := queries.DeprecateLDrawCatalogReleasesExcept(ctx, db.DeprecateLDrawCatalogReleasesExceptParams{
		Status: "active", CatalogVersion: catalogVersion,
	}); err != nil {
		return err
	}
	if _, err := queries.UpsertLDrawCatalogRelease(ctx, db.UpsertLDrawCatalogReleaseParams{
		CatalogVersion: catalogVersion, Release: lock.Release, SourceUrl: lock.SourceURL, ArchiveSha256: lock.ArchiveSHA256,
		Status: "active", ReleaseJson: pgtype.Text{String: releaseJSON, Valid: true}, PartCount: int32(manifest.PartCount),
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func composeCatalogVersion(release, archiveSHA string) string {
	if archiveSHA == "" {
		return "ldraw-official-" + release + "-unknown"
	}
	return fmt.Sprintf("ldraw-official-%s-%s", release, archiveSHA[:min(12, len(archiveSHA))])
}
