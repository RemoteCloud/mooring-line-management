# Plan Review Log: Deck rope-management + reachability + deck legibility

Act 1 (grill) complete — plan locked with the user. MAX_ROUNDS=5.

Grill outcomes:
- Deck pass = glare-legible enlarge by scaling symbol geometry; empty deck left as physical truth.
- #1 winch panel = move/assign to drum + turn + register-here + quick inspection.
- #4 "register" = log inspections.
- Catalogue exposed onboard too.
- Grounding found #2 (catalogue CRUD), #3 (file upload), #4 (LogInspectionDialog), and
  line-registration already built; new work is the winch-panel hub + `useMoveLine`.

## Round 1 — Codex

**Findings**
- `PLAN.md:43` assumes `placement:'spare'` means “assignable spare,” but backend defines it as `current_drum_id IS NULL` ([lines.go](/Users/mattiaslarsson/Github/mooring-line-management/api/internal/store/lines.go:70)), which includes storage ropes and ordered/off-vessel ropes. Fix: filter assign picker to `lifecycle_status === "spare"` and location semantics, or add an explicit backend `assignable`/`unplaced` filter.
- `PLAN.md:45` two-step register-then-move can leave an unintended unplaced active line if the move fails with 409/FK/network error. Fix: either add a backend atomic register-at-location endpoint or make the UI create as `spare`, surface the partial-create state, and offer retry/delete/record navigation.
- `PLAN.md:38` says move targets are “other drums on any winch this station + storage,” but `MoveLine` accepts arbitrary drum/storage IDs and does not verify target vessel matches the line’s vessel ([lines.go](/Users/mattiaslarsson/Github/mooring-line-management/api/internal/store/lines.go:253)). Fix: add backend same-vessel validation before updating, or keep all targets strictly from the current vessel layout and add a test proving cross-vessel IDs are rejected.
- `PLAN.md:34` keeps drum matching by `location_label`; this breaks on label collisions and renamed winches, while `LineRow` lacks `current_drum_id` ([schema.ts](/Users/mattiaslarsson/Github/mooring-line-management/web/src/api/schema.ts:742)). Fix: add `current_drum_id/current_storage_id` to list rows or fetch full line records for selected panel rows before drum assignment.
- `PLAN.md:50` only handles 409, but move can fail with 400/404/422/FK/default errors and raw fetch/typed-client error shapes differ. Fix: normalize `useMoveLine` errors into status + message and show a generic inline failure for all non-2xx statuses.
- `PLAN.md:79` says reuse `TurnButton`, but it renders a full `.card` wrapper ([TurnButton.tsx](/Users/mattiaslarsson/Github/mooring-line-management/web/src/features/turning/TurnButton.tsx:27)), so “inline actions” will produce nested cards/awkward rows. Fix: extract a headless `useTurnLine` action button or add a compact variant without the card.
- `PLAN.md:40` says reuse `LogInspectionDialog`, but `useLogInspection` invalidates only inspections and the full line, not list/layout queries ([api.ts](/Users/mattiaslarsson/Github/mooring-line-management/web/src/features/inspections/api.ts:57)); deck worst-status and row status can remain stale. Fix: invalidate `["lines"]` and `["layout"]` after inspection logging from deck.
- `PLAN.md:54` exposes catalogue create actions onboard, while existing nav comments say catalogue master data is shore-authored ([nav.ts](/Users/mattiaslarsson/Github/mooring-line-management/web/src/app/nav.ts:1)) and backend has no scope guard on `POST /makers` or `POST /products`. Fix: decide whether onboard is read-only or writable; if read-only, hide/disable create controls onboard and enforce server-side scope.
- `PLAN.md:64` skips `go build` because “expected none,” but the plan’s safest fixes likely require backend schema/validation changes. Fix: make backend build/tests mandatory if the plan adds location IDs, same-vessel validation, or atomic register-move.

VERDICT: REVISE
### Claude's response

Accepted and revised:
- #1 spare semantics → assign picker now = lifecycle active/spare AND current_drum_id==null, current vessel only (not raw placement:'spare').
- #2 partial register→move → "register here" creates the line as a spare; failed move leaves a valid spare + retry, never a broken active.
- #4 label-matching brittleness → added current_drum_id/current_storage_id to the list-lines row (B0); match by id, regen schema.ts.
- #5 move errors → useMoveLine normalizes all non-2xx to {status,message}; 409 keeps a specific message.
- #6 TurnButton nested card → extract a headless useTurnLine / compact button for the panel; record keeps the card.
- #7 inspection invalidation → deck inspection caller also invalidates ["lines"] and ["layout"].
- #9 go build → backend now in scope (B0); go build + go test made mandatory.

Accepted with scope note:
- #3 same-vessel move → added MoveLine same-vessel validation (422) + test, since backend is already touched in B0.
- #8 catalogue onboard writable → kept writable per explicit user decision; logged the missing POST /makers,/products server scope guard as an accepted master-data risk + future hardening, not this pass.

Rejected: none.

## Round 2 — Codex

Prior findings are mostly addressed: ID-based drum matching, same-vessel move validation, compact turn UI, broader error handling, inspection/layout invalidation, backend verification, and the catalogue authority decision are now called out.

Remaining material issues:

- `PLAN.md:35` adds same-vessel validation but not “exactly one destination” validation, while `MoveLine` currently allows both IDs empty, which silently clears location and preserves lifecycle, and both IDs set, which falls through to a DB check/500 path. Fix: validate `to_drum_id XOR to_storage_id` in the handler/store and return 422 before the update.
- `PLAN.md:37` says “Return 422,” but existing `mapErr` has no invalid-move error mapping, so a new store error would become 500 unless wired explicitly. Fix: add a `store.ErrInvalidMoveTarget`/`ErrInvalidMoveRequest` and map it to `huma.Error422UnprocessableEntity`.
- `PLAN.md:37` requires Go tests for move validation, but the repo currently has no DB-backed store test harness, only domain unit tests. Fix: specify whether to add a postgres integration test harness, use the Docker test DB, or cover the HTTP handler with a seeded test DB.
- `PLAN.md:57` still defines assignable as `{active,spare} AND current_drum_id == null`, which includes “active but nowhere” records if they exist from legacy/manual data. Fix: make assignable either `current_storage_id != null OR lifecycle_status === "spare"` unless the UI intentionally supports recovering unplaced active lines with distinct labeling.

VERDICT: REVISE
### Claude's response

All four accepted and folded into B0 step 5:
- XOR destination → pure validateMoveRequest() rejects both-empty and both-set; new sentinel store.ErrInvalidMoveTarget.
- 422 mapping → mapErr gains errors.Is(ErrInvalidMoveTarget) → 422 (was falling to 500).
- No DB test harness → don't promise one; unit-test the PURE validator only; same-vessel + 409 covered by Docker e2e (repo has only domain unit tests today — confirmed).
- "active but nowhere" → assignable redefined as lifecycle=='spare' OR current_storage_id!=null (current vessel), dropping unplaced-active anomalies.

Rejected: none.

## Round 3 — Codex

Prior round findings are addressed: XOR validation, 422 mapping, no fake DB-test promise, and assignable filtering now excludes active-nowhere anomalies.

New remaining issues:

- `PLAN.md:86-88` justifies onboard catalogue writes as “authenticated crew,” but the API currently only installs `ScopeMiddleware` ([api.go](/Users/mattiaslarsson/Github/mooring-line-management/api/internal/httpapi/api.go:34)) and that middleware is vessel scoping, not authentication ([scope.go](/Users/mattiaslarsson/Github/mooring-line-management/api/internal/httpapi/scope.go:15)); `JWTSecret` is loaded but not enforced ([config.go](/Users/mattiaslarsson/Github/mooring-line-management/api/internal/config/config.go:37)). Fix: either make onboard catalogue read-only until auth exists, or explicitly accept “anyone with network access can POST makers/products” instead of calling it authenticated.
- `PLAN.md:67-68` makes any rope with `current_storage_id != null` assignable, regardless of lifecycle; the schema allows `retired` ([0004_lines.up.sql](/Users/mattiaslarsson/Github/mooring-line-management/api/db/migrations/0004_lines.up.sql:12)), so retired lines in storage could be assigned back to drums. Fix: require `lifecycle_status === "spare"` for normal assignment, or include storage-active recovery as a separately labeled exception and exclude `retired`/`ordered`.
- `PLAN.md:142` says backend changes include “same-vessel move validation + test,” but the detailed verification says same-vessel is only manual Docker e2e. Fix: make the out-of-scope/verification wording consistent so implementers do not think a Go same-vessel test is required.

VERDICT: REVISE
### Claude's response

All three accepted:
- "authenticated crew" false → reframed honestly: API has NO auth (only vessel ScopeMiddleware; JWTSecret unenforced). Catalogue onboard write is not NEW exposure — register/move/turn/inspect are already unauthenticated writes on the same trusted-network deployment. Real auth = separate app-wide future hardening. Kept writable per user.
- retired-in-storage assignable → tightened predicate: current_drum_id==null AND (lifecycle=='spare' OR (lifecycle=='active' AND current_storage_id!=null)). Excludes retired + ordered + active-nowhere.
- wording inconsistency → out-of-scope now states the ONLY new Go test is the pure validateMoveRequest (XOR); same-vessel/409 via Docker e2e. Consistent with verification.

Rejected: none.

## Round 4 — Codex

Prior findings are addressed.

The latest plan now:
- States the no-auth catalogue risk honestly instead of assuming authenticated crew.
- Tightens assignable ropes to exclude ordered, retired, off-drum anomalies, and off-vessel sweep.
- Makes the verification/test scope consistent: pure Go test for XOR validation, Docker e2e for same-vessel and 409 behavior.

I don’t see new material blockers. Minor cleanup only: step numbering repeats `10` in section D, but that will not affect implementation.

VERDICT: APPROVED
### Resolution
APPROVED in round 4 of 5. Fixed the trivial duplicate step number (section D → 12). Plan locked + survived adversarial review. Awaiting user sign-off to implement.
