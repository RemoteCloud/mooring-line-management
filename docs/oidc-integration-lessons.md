# OIDC integration — pitfalls & lessons

We ported a **working** OIDC + group-permissions implementation from a reference
repo (`Color-Line-Vessel-Dashboard`) and still needed several debugging round
trips before login and authorization worked end-to-end.

This document records why, so the next integration can skip the detours.

## The one-line reason

> The reference encoded **provider behaviour** and **product decisions**, but not
> our **stack**. Nearly every bug lived at a seam where our environment differed
> from the reference's — a different language, a PWA service worker, a different
> build/deploy, and a freshly-configured OpenIddict client whose claims/scopes
> were not identical to the reference tenant's.

The reference was Python/FastAPI with a plain SPA; we are Go + a PWA (vite-plugin-pwa)
+ Docker Compose. "Copy the working flow" carried the *concepts* but none of the
*environment-specific* behaviour, and that's where time went.

---

## Category A — Stack divergence (cost the most time)

### A1. The PWA service worker hijacked `/api/auth/*` navigations  ← root cause of the redirect loop

- **Symptom:** endless redirect loop on login; the browser bounced between
  `/api/auth/login` and `/` with an ever-nesting `return_to`. The backend logged
  **zero** login attempts and created **zero** flow rows — the request never
  reached the API.
- **Root cause:** `vite-plugin-pwa`'s generated service worker uses
  `navigateFallback` to serve `index.html` for navigations. A full-page navigation
  to `/api/auth/login` (required, because cross-origin 302s can't go through
  `fetch`) was intercepted by the SW and answered with the SPA shell instead of
  reaching the backend. So the OIDC redirect never started.
- **Fix:** exclude the API from the SW.
  - `workbox.navigateFallbackDenylist: [/^\/api\//]`
  - and exclude `/api/auth/*` (and later `/api/access/*`) from `runtimeCaching` so
    auth/admin responses are never served from cache.
- **Lesson:** **The reference had no service worker, so this class of bug could
  not exist there.** Any redirect-based auth flow behind a PWA *must* deny the SW
  from touching the auth paths. Verify by checking the backend logs for a login
  attempt — if there are none, the request isn't reaching you, and it's almost
  always the SW or a proxy, not your auth code.

### A2. `/auth/session` was treated as a public path → always `401` → loop

- **Symptom:** the SPA always looked unauthenticated and bounced straight back to
  `/api/auth/login`.
- **Root cause:** the middleware short-circuited *all* public paths (`/auth/*`)
  before resolving the session, so `/auth/session` never reported the logged-in
  user — even with a valid cookie.
- **Fix:** for public paths, resolve the session **best-effort** (attach the user
  if present, never reject) instead of skipping resolution entirely
  (`attachUserBestEffort`).
- **Lesson:** "public" (never 401) is not the same as "anonymous" (never look at
  the cookie). The session/whoami endpoint must be public *and* identity-aware.

### A3. Self-inflicted detour: `SameSite=None` on the state cookie

- **Symptom:** while chasing the loop, the OAuth `state` cookie was switched to
  `SameSite=None` on a hunch that `Lax` was being dropped on the IdP→callback
  redirect.
- **Root cause of the detour:** wrong mental model. `SameSite=Lax` **is** sent on
  top-level cross-site GET navigations (which the callback is); `SameSite=None`
  is treated as third-party and **blocked by default in incognito/strict** — so
  the "fix" made it worse.
- **Fix:** revert to `SameSite=Lax` (correct for the callback round-trip).
- **Lesson:** don't guess cookie semantics under pressure. The callback is a
  top-level navigation → `Lax` is correct. Add logging *before* changing cookie
  attributes (we eventually added entry logs at login/callback that made the real
  cause — A1 — obvious).

---

## Category B — Provider / tenant configuration (not visible in reference code)

These are the ones the reference *couldn't* teach us, because they live in the
IdP's client config and claim shape, not in the code.

### B1. Admin comes from the `position_id` claim in `/userinfo`, NOT id_token roles/groups

- **Symptom:** real users (including a tenant admin) resolved as non-admin /
  read-only.
- **Root cause:** we matched admin against the id_token's role/group lists, but
  Maranics UserManagement conveys admin via the user's single **`position_id`**,
  and that field is **not in the JWT** — it is only returned by **`/userinfo`**.
  We were already *fetching* userinfo but reading the wrong claim.
- **Fix:** read `position_id` from the merged userinfo claims; admin = that id is
  in the configured allowlist (`OIDC_ADMIN_GROUP`, mirroring the reference's
  `UM_ADMIN_POSITION_IDS`). Per-team access uses the `position_team_ids` claim.
- **Lesson:** **read the reference's auth doc, not just its code.** This was
  documented in the reference's `AUTHENTICATION.md` ("`position_id` is not present
  in the JWT — it is read from `/userinfo`") and would have saved a full round
  trip. Always confirm *which claim* and *which endpoint* carries the
  authorization signal for the specific provider.

### B2. The admin "position" GUID is provider data — you must read it off a real login

- **Symptom:** `OIDC_ADMIN_GROUP=admin` (a guessed name) matched nobody.
- **Root cause:** admin is keyed on an opaque **GUID** (the seeded "Admin"
  position), not a friendly name. We didn't know the value.
- **Fix:** log the resolved identity (`sub`, `email`, `position_id`, `is_admin`)
  at callback, do one real login, copy the `position_id` into `OIDC_ADMIN_GROUP`.
  (For nightly it happened to equal the reference default
  `0ee848c0-3469-4561-99ad-623d8eb87a7d`.)
- **Lesson:** bootstrap admin from an **observed** login, never a guessed value.
  Keep a one-line identity log in the callback for exactly this.

### B3. The `roles` scope was rejected by the client

- **Symptom:** login failed; OpenIddict error **ID2051** (the client is not
  allowed the `roles` scope).
- **Fix:** drop `roles` from `OIDC_SCOPES` → `openid email profile offline_access`.
- **Lesson:** a scope that works for the reference's client is **not** granted to
  yours by default. Reconcile the requested scopes with what the *new* client is
  actually permitted (the IdP admin dashboard lists allowed scopes).

### B4. Friendly group names need a *separate* UM API call

- **Symptom:** the permissions UI showed a wall of GUIDs.
- **Root cause:** the token/userinfo carry team **ids**, not names. Names come
  from the UM external API `{umBase}/external/api/positionTeams?onlyActive=true`
  (with a `Tenant` header), called with the admin's own access token — exactly as
  the reference does.
- **Fix:** resolve id→name live, best-effort (degrade to GUIDs on failure).
- **Lesson:** if the reference shows a names list, find *where* the names come
  from — often a second, tenant-scoped API, not the token.

### B5. We were scraping the wrong claims into "groups"

- **Symptom:** the access list was polluted with un-nameable, un-grantable GUIDs
  (roles/`wids`/the admin position id).
- **Root cause:** defensive claim extraction read `roles, groups, role, wids` on
  top of `position_team_ids`. Only `position_team_ids` are the grantable units.
- **Fix:** extract **only** `position_team_ids`; derive admin from the persisted
  `position_id` column; list groups from the live positionTeams API.
- **Lesson:** "read every plausible claim defensively" is the wrong default for an
  authorization model — it conflates unrelated id namespaces. Key on the **one**
  claim the provider actually uses for the access unit.

---

## Category C — Local build / deploy / process (ours)

### C1. Compose didn't load `.env` → empty OIDC creds → boot failure

- **Symptom:** the API refused to start (missing `OIDC_CLIENT_ID/SECRET`).
- **Root cause:** `docker compose -f deploy/docker-compose.*.yml` looks for `.env`
  **next to the compose file** (`deploy/`), not at the repo root where ours lives.
- **Fix:** pass `--env-file .env` explicitly in every Makefile compose target.
- **Lesson:** with `-f <subdir>/compose.yml`, always set `--env-file` to the real
  location. A silent empty-env boot failure reads like a code bug but isn't.

### C2. Migration number collision with the default branch

- **Symptom:** migrations applied inconsistently; the auth migration was at the
  same number (`0008`) as an unrelated migration on `main`.
- **Fix:** renumber the auth migration (`0008`→`0009`) and recreate the dev DB
  volume (`down -v`) since the old number had been recorded.
- **Lesson:** when developing a feature on a branch, rebase migration numbers
  against the latest default branch before merging; never reuse a number.

### C3. Redirect URI must be pre-registered and match character-for-character

- **Lesson (standard, but bit us during port juggling):** the IdP only redirects
  to registered callback URLs, and scheme/host/**port**/path must match
  `OIDC_REDIRECT_URI` exactly. Our shore stack is `:8091`, onboard `:8090` — the
  registered URI must match whichever stack you log in through.

---

## Category D — Product / data-model mismatches found late

### D1. The "Users" count was meaningless

- It counted users who had signed into **this app** (≈1 each), not real UM team
  membership. UM exposes no member count, so the column was removed rather than
  faked.
- **Lesson:** don't surface a number you can't source correctly. Either get the
  authoritative count from the provider or omit it.

### D2. Default access posture changed (read-only-default → denied-default)

- The first cut made every authenticated non-admin **read-only**; the reference
  model is **denied by default**, access granted per team by an admin.
- **Lesson:** decide the default-deny vs default-allow posture up front — it
  changes the middleware, the "no access" UX, and the bootstrap story.

---

## Checklist for the next OIDC integration

Pre-flight (before writing matching code):

- [ ] Read the reference's **auth doc**, not just its code — note *which claim*
      and *which endpoint* (`id_token` vs `/userinfo` vs a tenant API) carries
      identity, admin, and group membership.
- [ ] Confirm the **new client's** allowed **scopes** and **released claims** in
      the IdP dashboard — they are not the reference client's.
- [ ] Register the **exact** redirect URI (scheme/host/port/path) for every stack
      you'll log in through.

Wiring:

- [ ] If there's a **service worker / PWA**, deny it from `/api/*` (navigation
      fallback **and** runtime cache), especially the auth + admin-config paths.
- [ ] Make the **session/whoami** endpoint public **and** identity-aware (resolve
      the cookie best-effort; never 401 it).
- [ ] Keep the OAuth `state` cookie `SameSite=Lax` (correct for the top-level
      callback navigation); do not reach for `None`.
- [ ] Pass `--env-file` to compose if `.env` isn't beside the compose file.
- [ ] Renumber migrations against the latest default branch.

Bootstrap & verify:

- [ ] Log the resolved identity (`sub/email/position_id/is_admin`) at callback;
      do one real login and copy the **observed** admin id into config — never
      guess a friendly name.
- [ ] Check the **backend logs** first when a redirect loops: no login attempt
      logged ⇒ the request isn't reaching you (SW/proxy), not an auth-code bug.
- [ ] Resolve group **names** from the provider's group API; degrade to ids on
      failure rather than blocking.

---

See [`authentication.md`](./authentication.md) for the current architecture and
config reference.
