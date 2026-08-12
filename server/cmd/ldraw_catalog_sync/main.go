package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/ldrawsync"
	"github.com/chimii-ai/chimii/server/internal/logger"
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
	manifestPath       string
	release            string
	overrideRelease    bool
	sourceURL          string
	overrideSourceURL  bool
	archiveSHA         string
	overrideArchiveSHA bool
	rootOnly           bool
	maxParts           int
	dryRun             bool
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := parseConfig()
	lock, manifest, err := ldrawsync.EmbeddedCatalog()
	if err != nil {
		return err
	}
	if cfg.lockPath != "" {
		lock, err = ldrawsync.ReadLock(cfg.lockPath)
		if err != nil {
			return err
		}
	}
	if cfg.manifestPath != "" {
		manifest, err = ldrawsync.ReadStarterKit(cfg.manifestPath)
		if err != nil {
			return err
		}
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

	archivePath, cleanup, err := ensureArchive(ctx, cfg.archive, lock.SourceURL, lock.ArchiveSHA256)
	if err != nil {
		return err
	}
	defer cleanup()

	if cfg.rootOnly {
		manifest = manifestForIDs("ldraw-embedded-fallback-10", lock.RootParts)
	}
	if cfg.maxParts > 0 && cfg.maxParts < manifest.PartCount {
		manifest.Parts = manifest.Parts[:cfg.maxParts]
		manifest.PartCount = len(manifest.Parts)
	}

	catalogVersion := build.ComposeLDrawCatalogVersion(lock.Release, lock.ArchiveSHA256)
	if cfg.dryRun {
		return dryRun(archivePath, catalogVersion, manifest)
	}

	pool, err := pgxpool.New(ctx, cfg.dbURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	lastReported := 0
	err = ldrawsync.SyncCatalog(ctx, pool, lock, manifest, archivePath, func(completed, total int) error {
		if completed == total || completed-lastReported >= 100 {
			slog.Info("ldraw catalog sync progress", "catalog_version", catalogVersion, "completed", completed, "total", total)
			lastReported = completed
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("ldraw catalog synced", "catalog_version", catalogVersion, "parts", manifest.PartCount)
	return nil
}

func parseConfig() config {
	cfg := config{
		dbURL:      os.Getenv("DATABASE_URL"),
		release:    build.CatalogRelease,
		sourceURL:  build.CatalogSourceURL,
		archiveSHA: build.CatalogArchiveSHA256,
		rootOnly:   false,
		maxParts:   0,
		dryRun:     false,
	}
	flagDBURL := flag.String("db-url", cfg.dbURL, "PostgreSQL DSN for Chimii DB")
	flagArchive := flag.String("archive", "", "Path to complete.zip; if empty, download from source URL")
	flagLock := flag.String("lock", "", "Optional catalog lock JSON override")
	flagManifest := flag.String("manifest", "", "Optional Starter Kit manifest JSON override")
	flagRelease := flag.String("release", cfg.release, "Catalog release tag")
	flagSource := flag.String("source-url", cfg.sourceURL, "LDraw source URL")
	flagArchiveSHA := flag.String("archive-sha256", cfg.archiveSHA, "SHA-256 of complete.zip")
	flagRootOnly := flag.Bool("root-only", cfg.rootOnly, "Compile only the 10 embedded fallback parts instead of Starter Kit 1000")
	flagMaxParts := flag.Int("max-parts", cfg.maxParts, "If >0, stop after first N parts")
	flagDryRun := flag.Bool("dry-run", cfg.dryRun, "Compile and validate only; no writes")
	flag.Parse()

	vis := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { vis[f.Name] = true })

	cfg.dbURL = strings.TrimSpace(*flagDBURL)
	cfg.archive = strings.TrimSpace(*flagArchive)
	cfg.lockPath = strings.TrimSpace(*flagLock)
	cfg.manifestPath = strings.TrimSpace(*flagManifest)
	cfg.release = strings.TrimSpace(*flagRelease)
	cfg.overrideRelease = vis["release"]
	cfg.sourceURL = strings.TrimSpace(*flagSource)
	cfg.overrideSourceURL = vis["source-url"]
	cfg.archiveSHA = strings.TrimSpace(*flagArchiveSHA)
	cfg.overrideArchiveSHA = vis["archive-sha256"]
	cfg.rootOnly = *flagRootOnly
	cfg.maxParts = *flagMaxParts
	cfg.dryRun = *flagDryRun
	if cfg.dbURL == "" {
		cfg.dbURL = "postgres://chimii:chimii@localhost:5432/chimii?sslmode=disable"
	}
	return cfg
}

func ensureArchive(ctx context.Context, archivePath, sourceURL, expectedSHA string) (string, func(), error) {
	if archivePath != "" {
		if _, err := os.Stat(archivePath); err != nil {
			return "", func() {}, err
		}
		if err := ldrawsync.VerifyArchive(archivePath, expectedSHA); err != nil {
			return "", func() {}, err
		}
		return archivePath, func() {}, nil
	}
	return ldrawsync.DownloadVerifiedArchive(ctx, nil, sourceURL, expectedSHA)
}

func manifestForIDs(kitID string, ids []string) ldrawsync.StarterKitManifest {
	manifest := ldrawsync.StarterKitManifest{SchemaVersion: 1, KitID: kitID, PartCount: len(ids)}
	manifest.Parts = make([]ldrawsync.StarterKitPart, 0, len(ids))
	for i, id := range ids {
		manifest.Parts = append(manifest.Parts, ldrawsync.StarterKitPart{Rank: i + 1, LDrawID: id, Name: id})
	}
	return manifest
}

func dryRun(archivePath, catalogVersion string, manifest ldrawsync.StarterKitManifest) error {
	library, err := ldrawsync.OpenLibrary(archivePath)
	if err != nil {
		return err
	}
	defer library.Close()
	compiled := 0
	var payloadBytes int64
	if err := library.CompilePartsStream(manifest.PartIDs(), 0, func(_ int, _ int, part ldrawsync.CompiledPart) error {
		compiled++
		payloadBytes += part.PayloadSize
		if compiled <= 5 {
			slog.Info("compiled", "catalog_version", catalogVersion, "part", part.PartID,
				"ldraw_sha256", part.LDrawSHA256, "content_sha256", part.ContentSHA256,
				"triangles", part.TriangleCount, "size_bytes", part.PayloadSize)
		}
		return nil
	}); err != nil {
		return err
	}
	slog.Info("dry-run complete", "catalog_version", catalogVersion, "parts", compiled, "payload_bytes", payloadBytes)
	return nil
}
