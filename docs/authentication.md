# Authentication & Authorization

The app authenticates users against an external **OpenID Connect (OIDC)** provider
using the **Backend-for-Frontend (BFF)** pattern, and authorizes them by **group
membership** carried in the IdP token claims. Authentication is **required** — every
endpoint except `/auth/*` and `/health` rejects unauthenticated requests.

- **Identity provider:** Maranics UserManagement (the "nightly" environment).
- **Tokens never reach the browser.** The access/refresh/id tokens are held
  server-side and **encrypted at rest** (AES-256-GCM). The browser only ever holds
  an opaque, HttpOnly session cookie (`mlm_session`).
- **Permissions are group-based.** Members of the configured admin group get full
  read+write; every other authenticated user is **read-only** (GETs succeed,
  mutations get `403`).

Source of truth in the code:

| Concern | File |
|---|---|
| OIDC client (discovery, auth URL, code exchange, id_token verify, userinfo, refresh) | `api/internal/auth/oidc.go` |
| Token-at-rest crypto (AES-256-GCM) + opaque-id generation | `api/internal/auth/crypto.go` |
| Group claim extraction + group→permission mapping | `api/internal/auth/groups.go` |
| HTTP routes `/auth/login`, `/auth/callback`, `/auth/logout`, `/auth/session` | `api/internal/httpapi/auth.go` |
| Auth/authz middleware (401/403 enforcement) | `api/internal/httpapi/authmw.go` |
| Session / user / flow persistence | `api/internal/store/auth.go` |
| Schema (users repurposed for OIDC, `oidc_flow`, `auth_session`) | `api/db/migrations/0008_auth.up.sql` |
| Frontend auth provider, route guard, write guard, hooks | `web/src/app/auth/` |

---

## The BFF login flow

All browser↔API traffic goes through the web origin under the `/api` prefix, which
the dev proxy (and the nginx config in production) strips before forwarding to the
API. So the browser hits `/api/auth/login`, the API sees `/auth/login`.

```
Browser                     API (BFF)                         IdP
   │  GET /api/auth/login       │                               │
   │ ─────────────────────────► │  create oidc_flow row         │
   │                            │  (state, PKCE verifier, nonce,│
   │                            │   return_to)                  │
   │  302 → IdP authorize URL   │                               │
   │ ◄───────────────────────── │  (state, nonce, S256 PKCE     │
   │                            │   challenge, prompt)          │
   │  full-page redirect ───────────────────────────────────► │  user signs in
   │                            │                               │
   │  302 → /api/auth/callback?code&state ◄─────────────────── │
   │ ─────────────────────────► │  take+delete flow by state    │
   │                            │  exchange code (+PKCE verifier)│ ──► tokens
   │                            │  verify id_token (RS512,       │
   │                            │   issuer, audience, nonce)     │
   │                            │  fetch /userinfo, merge claims │
   │                            │  extract groups, upsert user   │
   │                            │  encrypt tokens, create session│
   │  Set-Cookie: mlm_session   │                               │
   │  302 → return_to           │                               │
   │ ◄───────────────────────── │                               │
```

Key properties:

- **Auth code + PKCE (S256).** A fresh PKCE verifier and `state`/`nonce` are minted
  per login and stored in the short-lived `oidc_flow` table. The flow row is
  **consumed (deleted) on callback** and is rejected if older than 10 minutes.
- **id_token verification.** Signature is pinned to **RS512** (the only alg the
  provider offers), and issuer, audience (`OIDC_CLIENT_ID`) and the `nonce` are all
  checked.
- **Identity = id_token claims merged with `/userinfo`.** `sub`, `email`, `name`
  (falling back to `preferred_username`/`given_name`) and group claims are read from
  the union of both.
- **User upsert.** On every login the user is upserted into `app_user` keyed by
  `oidc_sub` (email, name, groups, `is_admin`, `last_login_at` refreshed). `app_user`
  is now an IdP-identity projection — there is **no password** (the legacy
  `password_hash` column was dropped in migration `0008_auth`).
- **Server-side session.** A random opaque `sid` is generated; the access, refresh
  and id tokens are AES-256-GCM encrypted and stored in `auth_session`. The browser
  receives only the `sid` as an HttpOnly cookie.
- **Open-redirect safe.** `return_to` is honored only if it is a same-origin
  relative path (`/...`, not `//...`); otherwise it falls back to `/`.

### Session cookie

`mlm_session` is set with `Path=/`, `HttpOnly`, `SameSite=Lax`, and `Secure` **iff
`APP_BASE_URL` is `https://`** (so it works over plain HTTP locally and is secured in
production). It carries only the opaque session id, never a token.

### Other endpoints

- `GET /auth/session` — returns `{ authenticated, user, groups, permissions }` for
  the current cookie; `401` when not signed in. The SPA polls this to know who you
  are and what you may do.
- `POST /auth/logout` — deletes the server-side session and clears the cookie.

---

## Authorization: groups → permissions

Implemented in `api/internal/auth/groups.go` and enforced in
`api/internal/httpapi/authmw.go`.

1. **Extract groups.** Group/role membership is read defensively from several claim
   keys (`roles`, `groups`, `role`, `wids`), accepting either JSON arrays or
   CSV/space-separated strings, then lowercased and de-duplicated.
2. **Resolve permissions.**
   - If the user's groups contain **`OIDC_ADMIN_GROUP`** (default `admin`) →
     `admin: true, canWrite: true` (full read+write).
   - Otherwise → `admin: false, canWrite: false` (**read-only**).
3. **Enforce per request** (after the scope guard):
   - Public paths (`/auth/*`, `/health`) pass through.
   - No valid session cookie → **`401`**.
   - Authenticated but mutating method (`POST/PUT/PATCH/DELETE`) without write
     permission → **`403`** (`"read-only: write access requires the admin group"`).
   - Otherwise the request proceeds with the user + permissions attached to context.

> Authorization is **server-enforced**. The frontend mirrors it for UX only: a
> read-only user sees write actions disabled and a "Read-only" indicator, but even if
> they bypassed the UI the API would still return `403`.

### Frontend behavior (`web/src/app/auth/`)

- `AuthProvider` fetches `GET /api/auth/session` (via React Query, `credentials:
  "include"`) to derive `status`/`user`/`groups`/`permissions`.
- `RequireAuth` wraps the app: while `unauthenticated` it triggers a **full-page**
  redirect to `/api/auth/login` (cross-origin 302s can't be done via fetch).
- `WriteGuard` disables nested write controls for read-only users and adds a
  "Read-only access" tooltip.
- `useAuth()` / `useCanWrite()` expose auth state and the write flag to feature pages.

---

## Environment variables

Authoritative defaults live in `.env.example` (root) and `api/internal/config/config.go`.
Copy `.env.example` → `.env` and fill the real client credentials.

| Var | Required | Default | Meaning |
|---|---|---|---|
| `OIDC_ISSUER` | no | `https://administration.cloud.maranics-nightly.com/nightly` | OIDC issuer / discovery base. The client fetches `.../.well-known/openid-configuration` from here. |
| `OIDC_CLIENT_ID` | **yes** (to serve) | — | OAuth client id registered with the provider; also the expected id_token audience. |
| `OIDC_CLIENT_SECRET` | **yes** (to serve) | — | OAuth client secret. Keep out of source control (it lives in `.env`, which is gitignored). |
| `OIDC_REDIRECT_URI` | **yes** (to serve) | `http://localhost:8091/api/auth/callback` | Provider callback. **Must exactly match** a redirect URI registered with the provider. |
| `OIDC_SCOPES` | no | `openid email profile roles offline_access` | Space-separated scopes. `roles` surfaces group claims; `offline_access` yields a refresh token. |
| `OIDC_PROMPT` | no | `login` | OIDC `prompt` param sent on the authorize request (e.g. force re-auth). |
| `OIDC_ADMIN_GROUP` | no | `admin` | Group name that grants write access. Everyone else is read-only. |
| `SESSION_SECRET` | no* | — | Server secret. In dev, if `TOKEN_ENC_KEY` is unset it is hashed (SHA-256) to derive the token-encryption key. |
| `TOKEN_ENC_KEY` | recommended | — (derived from `SESSION_SECRET` in dev) | Base64 of exactly **32 bytes** for AES-256-GCM token-at-rest encryption. Generate with `openssl rand -base64 32`. Set this explicitly in production. |
| `APP_BASE_URL` | no | `http://localhost:8091` | Public origin the browser reaches the app on. Drives the cookie `Secure` flag (`https` ⇒ `Secure`) and post-login redirects. |

\* `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET` and `OIDC_REDIRECT_URI` are validated only
on the serve path (`ValidateServe`); codegen/migration subcommands don't need them.
Provide either `TOKEN_ENC_KEY` or `SESSION_SECRET` so a token-encryption key can be
resolved.

> **Note on compose defaults.** The shore stack defaults `APP_BASE_URL` and
> `OIDC_REDIRECT_URI` to port **8091**; the onboard stack defaults them to **8090**
> (its web port). Whichever stack you log in through, the registered redirect URI and
> `APP_BASE_URL` must point at that stack's web port. Values come from your `.env` /
> host env, so override them per stack as needed.

---

## Registering the redirect URI (nightly admin dashboard)

The provider only redirects back to **pre-registered** callback URLs. To run locally:

1. Sign in to the Maranics nightly administration dashboard at
   `https://administration.cloud.maranics-nightly.com/nightly`.
2. Open (or create) the OAuth/OIDC **client application** for this app and note its
   **Client ID** and **Client Secret** → put them in `.env` as `OIDC_CLIENT_ID` /
   `OIDC_CLIENT_SECRET`.
3. Add the **redirect / callback URI** exactly as the app will send it. For local
   shore-stack development this is:

   ```
   http://localhost:8091/api/auth/callback
   ```

   (For the onboard stack it is `http://localhost:8090/api/auth/callback`.) The URI
   must match `OIDC_REDIRECT_URI` **character-for-character** — scheme, host, port and
   path all count.
4. Ensure the client is allowed the scopes `openid email profile roles
   offline_access` and that it issues group/role claims (so authorization can resolve
   the admin group).
5. To grant a user write access, put them in the group named by `OIDC_ADMIN_GROUP`
   (default `admin`) in the provider.

---

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| Redirected back to `/?auth_error=invalid_state` | The `oidc_flow` row was missing or already consumed (double callback, or cookies/DB reset mid-flow). Just retry the login. |
| `/?auth_error=state_expired` | More than 10 minutes elapsed between starting login and the callback. Retry. |
| `/?auth_error=exchange_failed` | Code exchange with the IdP failed — usually a wrong `OIDC_CLIENT_SECRET`, or `OIDC_REDIRECT_URI` not matching the registered one. |
| `/?auth_error=invalid_id_token` / `nonce_mismatch` | id_token failed verification (issuer/audience/signature) or nonce didn't match. Check `OIDC_ISSUER` and `OIDC_CLIENT_ID`. |
| Provider shows "redirect_uri mismatch" before returning | The redirect URI isn't registered, or doesn't match `OIDC_REDIRECT_URI` exactly (port/path). |
| `503 authentication unavailable` / `auth not configured` | DB/auth stack not wired at boot (e.g. Postgres down), or missing `OIDC_CLIENT_ID`/`OIDC_CLIENT_SECRET`/`OIDC_REDIRECT_URI` (`ValidateServe` fails). |
| Every request returns `401` | No/expired `mlm_session` cookie. Sign in again. Cross-origin cookie issues: ensure you reach the app via `APP_BASE_URL`. |
| Writes return `403` "read-only" | The user isn't in `OIDC_ADMIN_GROUP`. Add them to the admin group in the provider, then sign out/in to refresh groups. |
| Server fails to start: "TOKEN_ENC_KEY must decode to 32 bytes" | `TOKEN_ENC_KEY` is not base64 of exactly 32 bytes. Regenerate with `openssl rand -base64 32`, or unset it in dev to derive from `SESSION_SECRET`. |
| Cookie not marked `Secure` in production | Set `APP_BASE_URL` to an `https://` origin. |
