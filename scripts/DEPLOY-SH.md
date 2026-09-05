# Deploying Chimii to `sh`

From the repository root, run:

```bash
scripts/deploy-sh.sh
```

The default action is `deploy`. It requires a clean `main` checkout. An exact
stable `vX.Y.Z` tag matching the root package version is used when present;
otherwise the deployment version is `v<root-version>-<short-commit>`, for example
`v0.3.0-10f4f20`. The root version must be stable `X.Y.Z`. Prerelease or mismatched
release tags at HEAD are rejected, not silently converted to commit builds.

The script never bumps package versions, creates commits/tags, pushes, or triggers
or waits for GitHub Releases. Uncommitted tracked and untracked changes are
rejected before SSH; commit intended changes yourself. A commit build is not an
uncommitted `-dirty` build. Repeating the command for the same deployed commit
AND version verifies it without rebuilding. Adding a formal tag to that commit
therefore refreshes the deployed version. `FORCE_DEPLOY=true scripts/deploy-sh.sh`
forces another deployment.

## Release version consistency

For formal CLI/Desktop releases, the root `package.json` version and tag must match (`0.3.0` and
`v0.3.0`, respectively). Before creating a tag, update and commit the root
version, then run `bash scripts/check-release-version.sh v0.3.0` with the intended
tag. After tagging, run the same check without arguments to require an exact tag
at HEAD. This is a metadata check only: deployment also requires clean `main`.
The release workflow and tagged deployments share this check, so a tag with stale
package metadata is rejected. Untagged server deployments use a commit suffix
without relaxing the formal release workflow's checks.

Published CLI and Desktop artifact versions are derived from the Git tag. Server
deployments inject the same resolved tag or commit version into the backend, the
bundled CLI, and Web (`NEXT_PUBLIC_APP_VERSION`), and record the full source commit
in deployment state. Commit-suffixed versions are server build identifiers, not
client auto-update releases. Workspace package versions are not the product
release version; Mobile keeps its own release cadence. Do not bulk-rewrite
dependency, catalog or migration versions.

Updating `package.json` in a new commit does not update an existing tag. If a
tag has already been pushed, do not silently move it or replace its assets:
publish a new version, or obtain explicit approval for a same-version rebuild.

Git pushes use the repository's configured SSH remote; local `gh` authentication
is not needed to trigger publication. Actions uses its own `GITHUB_TOKEN`.
Release runs are serialized per tag, and existing releases fail closed. The only
approved replacement exception is the original `v0.3.0` release ID `383132584`
from run `33944152071` / commit `fc2f6538502d9f651118493731f4fd22d2da2a22`.
Recovery requires the old publisher to have stopped and no competing run to be
active. It downloads every original asset, verifies size and SHA-256, retains the
backup as an Actions artifact for 30 days, rechecks the release identity, and only
then deletes that old release (not its tag) before normal publication. It never
overwrites a subsequently rebuilt release or any other version. Rerunning a
completed release is not a general-purpose overwrite command.

## Deployment operations

| Command | Behavior |
| --- | --- |
| `scripts/deploy-sh.sh` or `deploy` | Build and deploy both applications |
| `scripts/deploy-sh.sh plan` | Read-only server, certificate and capacity checks |
| `scripts/deploy-sh.sh verify` | Application, origin TLS, firewall and public checks |
| `scripts/deploy-sh.sh config` | Converge the Caddy configuration; keep credentials |
| `scripts/deploy-sh.sh rollback` | Previous application release, only with the same migration set |
| `scripts/deploy-sh.sh bootstrap` | One-time replacement; refuses an initialized production installation |

The target is SSH alias `sh`, verified as `106.54.235.89` / `VM-0-8-ubuntu`.
Cloudflare proxies `https://chimii.com` to Caddy at `32443`. Its Origin Rule must
rewrite the destination port to `32443`, with proxied DNS and Full (strict) TLS.
The existing UFW policy permits only Cloudflare address ranges on this port.
The script audits that policy; it does not change Cloudflare or firewall rules.

The application uses `/opt/chimii`, the existing `chimii` database, and loopback
ports `8080` (backend) / `3000` (web). Caddy loads
`/etc/caddy/certs/chimii.com.pem` and `/etc/caddy/certs/chimii.com.key`.
Only the Chimii site blocks in `/etc/caddy/Caddyfile` are managed. The old manual
32443 block and managed 443 block converge to one managed 32443 block.

Runtime environment files remain on the server:

```text
/etc/chimii/backend.env
/etc/chimii/web.env
/etc/chimii/secrets/runtime.env
```

All three require mode `0600`. The systemd backend reads the runtime secrets as
an override. Deployments never upload local `.env` files or regenerate production
database, JWT, VCS or provider credentials. `config` updates Caddy only; application
units already reference stable release links, and deploy ensures the secrets
drop-in is present. Certificates and secrets must be provisioned before bootstrap.

Deployment builds before interrupting traffic. It restores a production dump into
a temporary database and runs migration rehearsal without starting any server or
workers on that clone. A separate empty business database tests the candidate
backend on `18080` and web on `13000`; candidate egress is restricted to loopback,
Redis is isolated by using the in-memory implementation, and uploads use a separate
directory. Applied migration files must remain unchanged.

The cutover runs as a server-side systemd job named
`chimii-sh-cutover-<release-id>.service`, surviving an SSH disconnect. It serves a
503 maintenance response, stops application writers, backs up `chimii`, migrates
that same database, switches release links and starts the new applications. Local
checks precede reopening traffic. Public endpoint/version checks precede recording
the successful release. The database OID is compared before and after migration.

Backups and progress are stored under
`/opt/chimii/state/deployments/<release-id>/`. An `exit-code` of `0` records success.
The `chimii.dump` file and checksum are retained. Application artifacts keep the
current and previous releases and any artifacts referenced by legacy snapshots;
eligible older successful releases are pruned according to `KEEP_RELEASES` (2).

If a cutover fails without schema changes, the previous application links and
Caddy configuration are restored. PostgreSQL migrations run outside a transaction
to support concurrent indexes. If a migration may have changed the schema, the
script retains maintenance mode, stopped applications, the backup and deployment
lock, and creates `needs-recovery`. It never automatically restores a database
over potentially newer writes or starts old code on an unknown schema. Inspect
the job log and backup before deciding how to recover. An application-only
`rollback` refuses a different migration set and never rolls the database back.

`bootstrap` retains the old destructive replacement workflow behind
`CONFIRM_REPLACE_LEGACY=replace-chimii`; it is rejected once `state/current.env`
or `state/bootstrap.complete` exists. `replace` is no longer a supported action.
`rollback-legacy` is an explicit disaster-recovery operation requiring
`CONFIRM_ROLLBACK_LEGACY=rollback-chimii`, not a daily rollback command.
