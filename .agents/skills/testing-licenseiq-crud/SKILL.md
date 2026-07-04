---
name: testing-licenseiq-crud
description: Bring up the LicenseIQ self-hosted stack via docker compose and test the Vendors/Products/Licenses CRUD UI end-to-end with API-key auth. Use when verifying M8b (CRUD UI) changes, the compose deployment, or API-key auth wiring.
---

# Testing LicenseIQ (self-hosted compose stack + CRUD UI)

## Stack bring-up
- Stack lives in `deploy/compose/compose.yaml`: services `postgres` (18), `valkey`, `migrate` (one-shot, runs `licenseiq migrate` then exits 0), `api` (Go, :8080), `frontend` (Next standalone, :3000, proxies `/api/v1` → `api:8080`).
- Bring up: `cd deploy/compose && docker compose up -d`. Verify: `docker compose ps` — all of postgres/valkey/api/frontend should be `healthy`; `migrate` should show `Exited (0)`.
- Bootstrap admin API key is injected for testing via `deploy/compose/compose.override.yaml` env (`BOOTSTRAP_ADMIN_EMAIL`, `BOOTSTRAP_ADMIN_API_KEY`). This override is test-only; do not commit real keys.
- API-key format is `liq_<keyId>.<secret>`.

## Known gotchas (may or may not still apply)
- After a Docker daemon / VM restart, `postgres` and `valkey` may not auto-restart, so `api` crash-loops. Re-run `docker compose up -d` to bring the whole stack back.
- `api` may crash-loop for ~30s after restart with `valkey addr: ... server misbehaving` — it resolves the valkey hostname during config validation before valkey DNS is ready. `restart: unless-stopped` recovers it; just wait and re-check `docker compose ps`. If it persists, hardening the cache dial with retry/backoff would fix it.
- `frontend/public/` must exist in git (there's a `.gitkeep`) or the Docker `COPY .../public` step fails on a fresh clone.
- Postgres 18 needs the volume mounted at `/var/lib/postgresql` (NOT `/var/lib/postgresql/data`).
- Gin/httprouter forbids two routes with different wildcard names at the same path position (e.g. `/x/{id}` vs `/x/{key}/...`) — it panics on boot. Use a static segment to disambiguate.

## Quick API smoke test (before browser testing)
```
AUTH="Authorization: Bearer <bootstrap key>"
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/v1/vendors            # expect 401
curl -s -o /dev/null -w "%{http_code}\n" -H "$AUTH" http://localhost:8080/api/v1/vendors   # expect 200
```
To reset test data quickly, DELETE vendors via `DELETE /api/v1/vendors/{id}` (expect 204). Delete fails if the vendor has dependent products/licenses (FK).

## Frontend auth
- Credential is entered at `/settings` ("Credential" input → "Save credential"), stored in `localStorage` under `licenseiq.authToken`, and attached as `Authorization: Bearer` to all requests.
- localStorage persists across browser restarts. For an unauthenticated test, click "Clear credential" first (header should read "Not signed in"). When signed in, header shows "API key <keyId>".

## CRUD flow (golden path)
1. Unauth `/vendors` → "unauthorized" + "No vendors found" (proves auth enforced).
2. `/settings` → paste key → toast "Credential saved".
3. `/vendors/new` → Name → "Save vendor" → toast "Vendor created", redirect, row appears.
4. `/products/new` → Name + Vendor `<select>` (populated from vendors) → "Save product" → toast "Product created".
5. `/licenses/new` → Vendor + Product `<select>`s, Type `<select>` (enum: Subscription/Perpetual/PerUser/PerDevice/PerCore/Concurrent/Enterprise — exact backend values), Seat count / Assigned seats → "Save license" → toast "License created"; list shows type + `assigned/total` seats.
6. Edit vendor (`/vendors` → Edit) → form pre-fills (proves GET-by-id) → change a field → "Save vendor" → toast "Vendor updated".
7. Delete (`/vendors` → Delete) → native `window.confirm("Delete vendor {name}?")` → accept → toast "Vendor deleted", row removed. Delete a throwaway vendor with no dependent product/license (FK).

## Testing tips
- The Type/Vendor/Product controls are native HTML `<select>` — click to open, click the option.
- Watch for dropped keystrokes in the `computer` `type` action on freshly-focused inputs (a leading char can be lost). Verify field contents via the annotated DOM before saving; correct via the Edit form if needed.
- Toasts are transient (Sonner) — screenshot immediately after the action.

## Devin Secrets Needed
- None for local testing: the bootstrap admin API key is defined in `deploy/compose/compose.override.yaml` for the test environment. Real deployments set `BOOTSTRAP_ADMIN_API_KEY` as a secret.
