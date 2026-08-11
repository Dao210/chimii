package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/ldrawsync"
	"github.com/chimii-ai/chimii/server/internal/logger"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger.Init()
	if err := run(); err != nil {
		slog.Error("ldraw catalog sync failed", "error", err)
		os.Exit(1)
	}
}

type config struct {
	dbURL              string
	archive            string
	lockPath           string
	release            string
	overrideRelease    bool
	sourceURL          string
	overrideSourceURL  bool
	archiveSHA         string
	overrideArchiveSHA bool
	rootOnly           bool
	maxParts           int
	dryRun             bool
	force              bool
}

func run() error {
	cfg := parseConfig()
	lock, err := ldrawsync.ReadLock(cfg.lockPath)
	if err != nil {
		return err
	}
	if cfg.overrideRelease {
		lock.Release = cfg.release
	}
	if cfg.overrideSourceURL {
		lock.SourceURL = cfg.sourceURL
	}
	if cfg.overrideArchiveSHA {
		lock.ArchiveSHA256 = cfg.archiveSHA
	}

	archivePath, downloaded, err := ensureArchive(cfg.archive, lock.SourceURL, lock.ArchiveSHA256)
	if err != nil {
		return err
	}
	if downloaded {
		defer os.Remove(archivePath)
	}

	if err := ldrawsync.VerifyArchive(archivePath, lock.ArchiveSHA256); err != nil {
		return err
	}

	library, err := ldrawsync.OpenLibrary(archivePath)
	if err != nil {
		return err
	}
	defer library.Close()

	parts := append([]string(nil), lock.RootParts...)
	if !cfg.rootOnly {
		all, err := ldrawsync.AssetFilesFromZip(archivePath)
		if err != nil {
			return err
		}
		parts = make([]string, 0, len(all))
		seen := map[string]struct{}{}
		for _, candidate := range all {
			effective := strings.TrimPrefix(candidate, "parts/")
			if candidate == effective {
				effective = strings.TrimPrefix(candidate, "p/")
			}
			if strings.Contains(candidate, "/") && effective != candidate {
				if _, ok := seen[effective]; ok {
					continue
				}
				seen[effective] = struct{}{}
				parts = append(parts, effective)
			}
		}
		if len(parts) == 0 {
			return errors.New("no compatible .dat paths in complete.zip")
		}
		sort.Strings(parts)
	}
	if cfg.maxParts > 0 && cfg.maxParts < len(parts) {
		parts = parts[:cfg.maxParts]
	}

	compiled, err := library.CompileParts(parts, 0)
	if err != nil {
		return err
	}

	catalogVersion := build.ComposeLDrawCatalogVersion(lock.Release, lock.ArchiveSHA256)
	releaseMetadata := map[string]any{
		"release":        lock.Release,
		"source_url":     lock.SourceURL,
		"archive_sha256": lock.ArchiveSHA256,
		"root_parts":     lock.RootParts,
		"part_count":     len(compiled),
	}
	releaseJSON, err := json.Marshal(releaseMetadata)
	if err != nil {
		return fmt.Errorf("marshal release metadata: %w", err)
	}

	summary(compiled, cfg.dryRun, catalogVersion)
	if cfg.dryRun {
		return nil
	}

	if err := persistCatalog(cfg, lock, catalogVersion, string(releaseJSON), compiled); err != nil {
		return err
	}
	return nil
}

func parseConfig() config {
	cfg := config{
		dbURL:      os.Getenv("DATABASE_URL"),
		release:    build.CatalogRelease,
		sourceURL:  build.CatalogSourceURL,
		archiveSHA: build.CatalogArchiveSHA256,
		rootOnly:   true,
		maxParts:   0,
		dryRun:     false,
		force:      false,
	}
	flagDBURL := flag.String("db-url", cfg.dbURL, "PostgreSQL DSN for Chimii DB")
	flagArchive := flag.String("archive", "", "Path to complete.zip; if empty, download from source URL")
	flagLock := flag.String("lock", "", "Path to catalog lock JSON file")
	flagRelease := flag.String("release", cfg.release, "Catalog release tag")
	flagSource := flag.String("source-url", cfg.sourceURL, "LDraw source URL")
	flagArchiveSHA := flag.String("archive-sha256", cfg.archiveSHA, "SHA-256 of complete.zip")
	flagRootOnly := flag.Bool("root-only", cfg.rootOnly, "Compile only locked root parts")
	flagMaxParts := flag.Int("max-parts", cfg.maxParts, "If >0, stop after first N parts")
	flagDryRun := flag.Bool("dry-run", cfg.dryRun, "Compile and validate only; no writes")
	flagForce := flag.Bool("force", cfg.force, "Overwrite an existing release even if hashes differ")
	flag.Parse()

	vis := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { vis[f.Name] = true })

	cfg.dbURL = strings.TrimSpace(*flagDBURL)
	cfg.archive = strings.TrimSpace(*flagArchive)
	cfg.lockPath = strings.TrimSpace(*flagLock)
	cfg.release = strings.TrimSpace(*flagRelease)
	cfg.overrideRelease = vis["release"]
	cfg.sourceURL = strings.TrimSpace(*flagSource)
	cfg.overrideSourceURL = vis["source-url"]
	cfg.archiveSHA = strings.TrimSpace(*flagArchiveSHA)
	cfg.overrideArchiveSHA = vis["archive-sha256"]
	cfg.rootOnly = *flagRootOnly
	cfg.maxParts = *flagMaxParts
	cfg.dryRun = *flagDryRun
	cfg.force = *flagForce
	if cfg.dbURL == "" {
		cfg.dbURL = "postgres://chimii:chimii@localhost:5432/chimii?sslmode=disable"
	}
	if cfg.lockPath == "" {
		cfg.lockPath = mustDefaultCatalogLockPath()
	}
	return cfg
}

func persistCatalog(cfg config, lock ldrawsync.LockFile, catalogVersion string, releaseJSON string, compiled []ldrawsync.CompiledPart) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.dbURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	queries := db.New(pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)

	existing, err := qtx.GetLDrawCatalogReleaseByVersion(ctx, catalogVersion)
	if err == nil {
		if !cfg.force && existing.ArchiveSha256 != "" && !strings.EqualFold(existing.ArchiveSha256, lock.ArchiveSHA256) {
			return fmt.Errorf("existing release %s has different archive sha256 (%s != %s); use --force to overwrite", catalogVersion, existing.ArchiveSha256, lock.ArchiveSHA256)
		}
	} else if err != pgx.ErrNoRows {
		return fmt.Errorf("load existing release: %w", err)
	}

	if err := qtx.DeprecateLDrawCatalogReleasesExcept(ctx, db.DeprecateLDrawCatalogReleasesExceptParams{
		Status:         "active",
		CatalogVersion: catalogVersion,
	}); err != nil {
		return fmt.Errorf("deprecate prior releases: %w", err)
	}

	if _, err := qtx.UpsertLDrawCatalogRelease(ctx, db.UpsertLDrawCatalogReleaseParams{
		CatalogVersion: catalogVersion,
		Release:        lock.Release,
		SourceUrl:      lock.SourceURL,
		ArchiveSha256:  lock.ArchiveSHA256,
		Status:         "active",
		ReleaseJson:    pgtype.Text{String: releaseJSON, Valid: true},
		PartCount:      int32(len(compiled)),
	}); err != nil {
		return fmt.Errorf("upsert catalog release: %w", err)
	}

	for _, part := range compiled {
		storageKey := filepath.ToSlash(filepath.Join("ldraw", catalogVersion, part.PartID+".glb"))
		if _, err := qtx.UpsertLDrawPartRevision(ctx, db.UpsertLDrawPartRevisionParams{
			CatalogVersion:   catalogVersion,
			PartID:           part.PartID,
			Revision:         part.Revision,
			Kind:             part.Kind,
			LdrawSha256:      part.LDrawSHA256,
			StorageBackend:   "db",
			ContentSha256:    pgtype.Text{String: part.ContentSHA256, Valid: true},
			ContentType:      part.ContentType,
			StorageKey:       storageKey,
			Payload:          part.Content,
			PayloadSizeBytes: part.PayloadSize,
			PayloadFormat:    2,
		}); err != nil {
			return fmt.Errorf("upsert part %s: %w", part.PartID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit catalog sync: %w", err)
	}

	slog.Info("ldraw catalog synced", "catalog_version", catalogVersion, "parts", len(compiled))
	return nil
}

func ensureArchive(archivePath, sourceURL, expectedSHA string) (string, bool, error) {
	if archivePath != "" {
		if _, err := os.Stat(archivePath); err != nil {
			return "", false, err
		}
		return archivePath, false, nil
	}
	if sourceURL == "" {
		return "", false, errors.New("archive not provided and source URL is empty")
	}
	if expectedSHA == "" {
		return "", false, errors.New("archive SHA-256 is required for download verification")
	}

	file, err := os.CreateTemp("", "ldraw-complete-*.zip")
	if err != nil {
		return "", false, err
	}

	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", false, err
	}
	res, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		_ = os.Remove(file.Name())
		return "", false, fmt.Errorf("download %q failed: %s", sourceURL, res.Status)
	}
	if _, err := io.Copy(file, res.Body); err != nil {
		_ = os.Remove(file.Name())
		return "", false, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return "", false, err
	}
	if err := ldrawsync.VerifyArchive(file.Name(), expectedSHA); err != nil {
		_ = os.Remove(file.Name())
		return "", false, err
	}
	return file.Name(), true, nil
}

func mustDefaultCatalogLockPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return "tools/ldraw-catalog/catalog.lock.json"
	}
	current := wd
	for i := 0; i < 4; i++ {
		candidate := filepath.Join(current, "tools", "ldraw-catalog", "catalog.lock.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	return "tools/ldraw-catalog/catalog.lock.json"
}

func summary(parts []ldrawsync.CompiledPart, dryRun bool, catalogVersion string) {
	if len(parts) == 0 {
		slog.Info("nothing to sync", "catalog_version", catalogVersion)
		return
	}
	for i, part := range parts {
		if i >= 5 {
			break
		}
		slog.Info("compiled",
			"catalog_version", catalogVersion,
			"part", part.PartID,
			"ldraw_sha256", part.LDrawSHA256,
			"content_sha256", part.ContentSHA256,
			"triangles", part.TriangleCount,
			"size_bytes", strconv.FormatInt(part.PayloadSize, 10),
		)
	}
	if dryRun {
		slog.Info("dry-run complete", "catalog_version", catalogVersion, "parts", len(parts))
	}
}
