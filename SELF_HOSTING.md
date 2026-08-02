# Self-Hosting Guide

Chimii production deployments run as native Linux processes managed by systemd. PostgreSQL stores durable data, Redis provides public-auth rate limiting and realtime relay, nginx terminates TLS, and the CLI daemon runs on each user's machine rather than on the application server.

## Architecture

| Component | Default address | Process manager |
|---|---|---|
| PostgreSQL 17 + pgvector | `127.0.0.1:5432` | systemd or a managed database service |
| Redis 7 | `127.0.0.1:6379` | systemd or a managed Redis service |
| Chimii backend | `127.0.0.1:18080` | systemd |
| Next.js web server | `127.0.0.1:13000` | systemd |
| nginx | `:80`, `:443` | systemd |

Only SSH, HTTP, and HTTPS should be reachable from the public internet. Keep ports `5432`, `6379`, `13000`, and `18080` private.

## Automated Native Deployment

The repository deployment script cross-compiles Go locally, builds only the Web app on the Linux host, installs PostgreSQL/pgvector and Redis when needed, generates the production environment, runs migrations, and activates versioned systemd releases. It never packages desktop or mobile applications.

```bash
./scripts/deploy.sh deploy
./scripts/deploy.sh verify
```

The first command intentionally configures HTTP before DNS is moved. After the domain's A record points to the server, issue and install the certificate separately:

```bash
TLS_EMAIL=ops@example.com ./scripts/deploy.sh tls
```

## Prerequisites

Build machine:

- Go 1.26.1
- Node.js 22
- pnpm 10.28.2

Linux server:

- A systemd-based distribution
- PostgreSQL 17 with pgvector, or a compatible managed PostgreSQL service
- Redis 7, or a compatible managed Redis service
- Node.js 22
- nginx
- A dedicated unprivileged `chimii` service account

Build artifacts must match the target Linux architecture. A web bundle created on macOS can contain platform-specific dependencies and must not be copied directly to Linux.

## Build the Backend

Set the target architecture to `amd64` or `arm64`:

```bash
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/server ./cmd/server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/migrate ./cmd/migrate
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/backfill_task_usage_hourly ./cmd/backfill_task_usage_hourly
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/backfill_codex_usage_cache ./cmd/backfill_codex_usage_cache
```

Deploy the binaries together with `server/migrations/`. The migration command discovers that directory from its working directory or executable location.

## Build the Web App

Build on a Linux environment matching the target architecture:

```bash
pnpm install --frozen-lockfile
STANDALONE=true CHIMII_LOW_MEMORY_BUILD=true pnpm --filter @chimii/web build
```

Deploy these paths from `apps/web/.next/`:

- `standalone/`
- `static/`, copied to `standalone/apps/web/.next/static/`
- `apps/web/public/`, copied to `standalone/apps/web/public/`

The web process starts from the standalone root:

```bash
NODE_ENV=production \
HOSTNAME=127.0.0.1 \
PORT=13000 \
REMOTE_API_URL=http://127.0.0.1:18080 \
node apps/web/server.js
```

`REMOTE_API_URL` is read at runtime, so changing the backend address does not require rebuilding the web app.

## Production Files

Keep releases immutable and durable data outside release directories:

```text
/opt/chimii/releases/backend/<release-id>/
/opt/chimii/releases/web/<release-id>/
/opt/chimii/current-backend -> releases/backend/<release-id>
/opt/chimii/current-web     -> releases/web/<release-id>
/etc/chimii/backend.env
/etc/chimii/web.env
/var/lib/chimii/uploads/
```

Recommended backend settings for a single-domain deployment:

```env
APP_ENV=production
DATABASE_URL=postgres://chimii:replace-me@127.0.0.1:5432/chimii?sslmode=disable
REDIS_URL=redis://127.0.0.1:6379/0
JWT_SECRET=replace-with-a-long-random-value
PORT=18080
CHIMII_BIND_HOST=127.0.0.1
FRONTEND_ORIGIN=https://chimii.example.com
CORS_ALLOWED_ORIGINS=https://chimii.example.com
CHIMII_APP_URL=https://chimii.example.com
CHIMII_PUBLIC_URL=https://chimii.example.com
CHIMII_TRUSTED_PROXIES=127.0.0.1/32
LOCAL_UPLOAD_DIR=/var/lib/chimii/uploads
ALLOW_SIGNUP=true
```

Recommended web settings:

```env
NODE_ENV=production
HOSTNAME=127.0.0.1
PORT=13000
REMOTE_API_URL=http://127.0.0.1:18080
```

Store these files with mode `0600`. The deployment script may use a local `.env` as the optional-integration base, but always replaces database credentials, generated secrets, bind addresses, origins, and other production-critical values on the server.

## Database Setup and Migrations

Create the required extension once:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

Before activating a new backend release, run its migration binary against the production database:

```bash
DATABASE_URL="postgres://..." /opt/chimii/releases/backend/<release-id>/migrate up
```

Stop the backend while applying migrations unless a migration has explicitly been verified as compatible with the currently running version. If application rollback is needed after a failed health check, switch the application symlink back; do not automatically run down migrations.

## systemd

Backend unit:

```ini
[Unit]
Description=Chimii Backend
After=network-online.target postgresql.service redis-server.service
Wants=network-online.target
Requires=postgresql.service redis-server.service

[Service]
Type=simple
User=chimii
Group=chimii
WorkingDirectory=/opt/chimii/current-backend
EnvironmentFile=/etc/chimii/backend.env
ExecStart=/opt/chimii/current-backend/server
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

Web unit:

```ini
[Unit]
Description=Chimii Web
After=network-online.target chimii-backend.service
Wants=network-online.target

[Service]
Type=simple
User=chimii
Group=chimii
WorkingDirectory=/opt/chimii/current-web
EnvironmentFile=/etc/chimii/web.env
ExecStart=/usr/bin/node apps/web/server.js
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

After changing units:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now chimii-backend chimii-web
```

## Reverse Proxy

Single-domain Caddy example:

```caddyfile
chimii.example.com {
	encode zstd gzip

	@backend path /api/* /uploads/* /ws /healthz
	reverse_proxy @backend 127.0.0.1:18080 {
		flush_interval -1
	}

	reverse_proxy 127.0.0.1:13000
}
```

Leave `/auth/*` on the web route. The Next.js runtime selectively proxies backend authentication endpoints while retaining frontend callback routes.

## Verification

```bash
systemctl is-active postgresql redis-server nginx chimii-backend chimii-web
curl -fsS http://127.0.0.1:18080/healthz
curl -fsSI http://127.0.0.1:13000/
curl -fsS https://chimii.example.com/healthz
```

Check service logs without printing environment files:

```bash
journalctl -u chimii-backend -n 100 --no-pager
journalctl -u chimii-web -n 100 --no-pager
```

## Connect the CLI

Run this on each machine that executes agents:

```bash
chimii setup self-host \
  --server-url https://chimii.example.com \
  --app-url https://chimii.example.com
```

The daemon stays on the user's machine and connects to the backend over HTTPS/WebSocket.

## Kubernetes

The source Helm chart remains available at [`deploy/helm/chimii/`](deploy/helm/chimii/). Configure `images.backend.repository`, `images.backend.tag`, `images.frontend.repository`, and `images.frontend.tag` explicitly for artifacts available to your cluster, then install the local chart with Helm.

## Advanced Configuration

See [SELF_HOSTING_ADVANCED.md](SELF_HOSTING_ADVANCED.md) for email, storage, cookies, signup controls, metrics, database rollups, and split-domain proxy configuration.
