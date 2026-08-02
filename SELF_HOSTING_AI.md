# Native Linux Self-Hosting (for AI Agents)

Use [SELF_HOSTING.md](SELF_HOSTING.md) as the source of truth. Chimii production services run as native systemd units behind Caddy or nginx.

Before changing a server, inspect the target architecture, Linux distribution, active systemd units, reverse-proxy configuration, PostgreSQL version, pgvector availability, Node.js version, existing environment-file paths, and upload storage path. Never print secret values.

Deployment order:

1. Build target-compatible backend and standalone web artifacts before uploading anything.
2. Upload artifacts to a new immutable release directory and verify checksums.
3. Stop the backend and run the new `migrate up` binary with the production `DATABASE_URL`.
4. Atomically switch the backend and web release symlinks.
5. Restart `chimii-backend` and `chimii-web`.
6. Verify loopback health checks and the public HTTPS endpoint.
7. On application failure, restore the previous symlink; never run automatic down migrations.

Useful checks:

```bash
systemctl is-active postgresql chimii-backend chimii-web caddy
curl -fsS http://127.0.0.1:8080/healthz
curl -fsSI http://127.0.0.1:3000/
journalctl -u chimii-backend -n 100 --no-pager
journalctl -u chimii-web -n 100 --no-pager
```
