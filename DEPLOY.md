# Deploy — Milestone 1 (VPS · Docker Compose + Caddy)

A self-hosted-identity + workspace API behind automatic HTTPS. No chat yet.

## Topology
- `caddy` — the only internet-facing service; terminates HTTPS on `:80`/`:443`,
  reverse-proxies to `app` over the internal network.
- `app` — Go API (HTTP), bound to `127.0.0.1` only (not internet-facing).
- `postgres`, `redis` — bound to `127.0.0.1` only (never internet-facing).

## Prerequisites
- A VPS with Docker + Docker Compose.
- A domain with an `A` record → the VPS public IP.
  (No domain yet? Use a free one from DuckDNS and set `DOMAIN` to it.)

## No domain yet? (deploy with just the server IP)
Let's Encrypt does **not** issue certificates for bare IPs. Easiest fix — use a
magic-DNS hostname that maps to your IP, so Caddy still gets real HTTPS:
- **nip.io / sslip.io** (no signup): set `DOMAIN=<VPS_IP>.nip.io`
  (e.g. `203.0.113.10.nip.io`) plus matching `APP_BASE_URL`/`ALLOWED_ORIGINS`.
  Nothing else changes — keep port 80 open for the ACME challenge.
- **DuckDNS** (free, 1-min signup): `name.duckdns.org` → your IP; survives IP changes.
- Throwaway private test only: a self-signed cert (`tls internal` in the Caddyfile).
  Browsers warn "not secure" — never use this for real users.

## Steps
1. **Point DNS:** create an `A` record `app.example.com → <VPS_IP>`
   (or use a `*.nip.io` hostname as above — no DNS setup needed).
   Verify: `dig +short app.example.com`.
2. **Firewall — web + SSH only:**
   ```bash
   ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw enable
   ```
   Postgres/Redis bind to `127.0.0.1`, so they stay private regardless of ufw.
   (Caddy needs `:80` open for the Let's Encrypt HTTP challenge.)
3. **Create `.env` on the VPS** (never committed — it is gitignored):
   ```env
   DOMAIN=app.example.com
   JWT_SECRET=<openssl rand -base64 48>
   DB_PASSWORD=<strong-password>
   APP_BASE_URL=https://app.example.com
   ALLOWED_ORIGINS=https://app.example.com
   # First deploy without a mail server — verification links go to the app logs:
   EMAIL_TRANSPORT=log
   # For real email instead, set EMAIL_TRANSPORT=smtp and:
   # SMTP_HOST= SMTP_PORT= SMTP_USER= SMTP_PASSWORD= SMTP_FROM=
   ```
4. **Build & start:**
   ```bash
   docker compose up -d --build
   ```
5. **Verify** (Caddy may take ~30s to issue the certificate on first run):
   ```bash
   curl https://app.example.com/healthz   # {"status":"ok"}
   curl https://app.example.com/readyz    # {"status":"ready"} once Postgres is up
   ```

## Operating notes
- **Email in `log` mode:** registration/verification links are printed in
  `docker compose logs app` — no SMTP needed for a demo.
- **Logs:** `docker compose logs -f app` (or `caddy`).
- **DB access from your laptop:** SSH-tunnel to `127.0.0.1:5432` (it is not public).
- **Migrations** run automatically on container start (`./migrate && ./main`).
- **Local dev:** set `DOMAIN=localhost` (Caddy uses a local CA), or skip Caddy and
  hit the app directly at `http://127.0.0.1:8080`.
