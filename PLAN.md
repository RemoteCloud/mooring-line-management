# Plan: Deck rope-management + reachability, plus deck-map legibility

_Locked via grill — by Claude + mattias_

## Goal

Make the deck map the operational hub the crew actually works from, not a read-only
diagram. Standing at a winch, a crew member should manage the ropes on it: assign a
spare to an empty drum, register a brand-new rope straight onto a drum, move a rope to
another drum or to storage, turn a rope's side, and log a quick condition check — all
from the winch panel without hunting through the register. Plus: enlarge the deck
symbols so the map reads on a sunlit tablet, and expose the (already-built) catalogue to
onboard crew. A grounding pass found that documents-upload, catalogue CRUD, inspection
logging, and line registration are **already implemented**; the genuinely new work is the
winch-panel action hub (#1) and the move hook it needs. The rest is surfacing and
verification.

## Approach

### A. Deck symbol enlarge (glare legibility) — `web/src/features/deck/symbols.tsx` + styles
1. Scale the drawing constants up: `DRUM_W` 20→28, `DRUM_H` 30→42, `DRUM_GAP` 4→5,
   `PAD` 8→11. Winch bodies and drums grow; the whole `<g>` is the tap target, so hit
   areas grow past the 44px floor at tablet width.
2. Bump label/marker type: `.sym-label` 11→14px, `.sym-drive` 10→13px, `StatusMark`
   `r` 7→9. Counter-rotation math already handles labels; verify after scaling.
3. Keep `viewBox` 1000×600 and the normalized 0..1 model untouched (edit-mode drag needs
   the full deck). Do NOT crop/auto-fit, do NOT spread seed winches — empty deck is
   physical truth (Principle 3).

### B0. Backend: robust drum identity + move safety (small) — `api/internal/store/lines.go`, `httpapi/lines.go`
4. Add `current_drum_id` and `current_storage_id` to the **list-lines** row payload
   (`LineRow`) so the frontend matches ropes to drums by **id**, not by parsing
   `location_label` (Codex #4 — label matching breaks on collisions/renames). Regenerate
   `web/src/api/schema.ts` via `make gen-ts`.
5. Harden `MoveLine` request validation (Codex round-2):
   - **Exactly-one destination (XOR):** reject both-empty (today silently clears location)
     and both-set (today hits the CHECK and 500s). Extract a **pure** func
     `validateMoveRequest(toDrumID, toStorageID) error` → new sentinel
     `store.ErrInvalidMoveTarget`. Pure = unit-testable with no DB (the repo has only
     domain unit tests, no DB harness — so don't promise an integration test).
   - **Same-vessel:** enforce in the `MoveLine` UPDATE itself — the target drum/storage must
     belong to the line's vessel (add `AND <target>.vessel_id = line.vessel_id` to the
     resolve, or check the looked-up target vessel); zero match → `ErrInvalidMoveTarget`.
     Onboard is single-vessel so this is unreachable there; it guards shore.
   - **Error mapping:** add `errors.Is(err, store.ErrInvalidMoveTarget) →
     huma.Error422UnprocessableEntity` in `mapErr` (today it falls through to 500 — Codex #2).
   - Tests: unit-test the pure `validateMoveRequest` (XOR cases). Cover same-vessel +
     occupied-drum (409) via the manual Docker e2e, not a new Go DB harness.

### B. Winch-panel rope management hub (NEW — the core) — `web/src/features/deck/`
View mode only (edit mode stays layout-only). When a winch is selected, render a
**drum-aware** panel instead of the flat rope list:
6. Build the drum list from `layout.winch.drums` (each has `id`, `idx`, `line_count`) and
   match ropes to drums by `current_drum_id` (from B0), not by label. Render one row per
   drum, in `idx` order.
7. **Occupied drum** (rope present): rope name + status mark; actions inline:
   - **Turn** → extract a headless `useTurnLine` / compact button from `TurnButton`, which
     today renders a full `.card` (Codex #6 — reusing it as-is nests cards, an impeccable
     ban). Compact variant only inside the panel; record page keeps the card.
   - **Move** → target picker built strictly from the **current vessel's** layout (other
     drums this station + storage) → `POST /lines/{id}/move` via new `useMoveLine` hook.
   - **Log inspection** → reuse `LogInspectionDialog`, but the deck caller must also
     invalidate `["lines"]` and `["layout"]` (the existing `useLogInspection` invalidates
     only inspections + the single line — Codex #7 — so deck dots/rows would go stale).
   - Rope name still links to the full record.
8. **Empty drum**: two affordances:
   - **Assign rope** → picker of **assignable** lines, defined precisely as:
     `current_drum_id == null AND ( lifecycle_status == 'spare' OR (lifecycle_status ==
     'active' AND current_storage_id != null) )`, current vessel only. I.e. a spare, or an
     active rope currently in storage being redeployed. Excludes off-drum sweep of
     ordered/off-vessel (Codex r1 #1), active-but-nowhere anomalies (r2 #4), **and
     retired-in-storage** (r3 #2 — retired must never return to a drum). Then `move` onto
     `drum.id`.
   - **Register here** → open `AddLineDialog` (extended with optional target `drumId`),
     creating the line as **`spare`** (not active); on success chain `move(newLineId,
     drumId)`. If the move fails, the line persists as a valid spare (not a broken
     half-placed active line — Codex #2); surface the error with retry + a link to the new
     record. Register-line has no drum field, so placement is always register→move.
9. Storage panel keeps its rope list; add the same Move/Turn/Inspect row actions for ropes
   in storage (move target includes drums).
10. `useMoveLine` **normalizes all non-2xx errors** into `{status, message}` and shows a
    generic inline failure; 409 (occupied) gets a specific message. Not just 409 (Codex
    #5 — move can also 400/404/422/FK).

### C. Catalogue reachable onboard — `web/src/app/router.tsx` + nav
11. Remove the shore-only `Navigate` guard on `/catalogue`; add the nav link for onboard.
    CataloguePage already does makers + line-types (list-only; leave as display) + products
    create. **Decision (per user): onboard is writable** — crew can add vendors/models on
    deck. Honest risk statement: the API has **no authentication at all today** — only
    `ScopeMiddleware` (vessel scoping, not auth); `JWTSecret` is loaded but unenforced. So
    "anyone with network access to the onboard API can `POST /makers`/`/products`." This is
    **not new exposure** — register/move/turn/inspect are already unauthenticated writes on
    the same trusted-network onboard deployment; catalogue is no different. Real auth + a
    catalogue write scope guard are a separate, app-wide future hardening, explicitly out of
    this pass.

### D. Verify already-built (little/no code) — confirm in Docker
12. Smoke-test in the running onboard + shore stacks: FilesTab upload (photo + cert +
    document → MinIO), CataloguePage create maker/product, LogInspectionDialog from a
    record, AddLineDialog register. Fix only what's broken. Surface to user that #2/#3/#4
    were already implemented.

### Verification
- `go build ./...` + `go test ./...` clean — backend **is** touched now (B0: list-row drum
  ids + move XOR/same-vessel validation + `ErrInvalidMoveTarget` mapping). New Go test is the
  **pure** `validateMoveRequest` (no DB); same-vessel/409 covered by Docker e2e.
- `tsc -b` + `npm run build` clean; `make gen-ts` regenerates `schema.ts` cleanly.
- Headless-Chrome screenshots of `/deck` (enlarged symbols, winch panel actions),
  `/catalogue` reachable onboard. Grayscale check unaffected (marker shapes unchanged).
- Manual e2e against Docker: assign spare → drum, register-here, move, turn, inspect;
  each reflects after refetch. Move to occupied drum → 409 surfaced.

## Key decisions & tradeoffs

- **Enlarge by scaling symbol geometry, not cropping the canvas.** Deterministic, grows
  hit targets, preserves the full 0..1 deck for dragging. Rejected auto-fit viewBox (breaks
  edit-mode drag, shifts as symbols move) and seed re-placement (masks real layout).
- **Empty lower deck left as-is** — physical truth (Principle 3); the sparseness was tiny
  symbols, not the empty area.
- **Reuse over rebuild.** Turn, inspection, register, file upload, catalogue all exist;
  the winch panel composes them rather than reimplementing. Only new primitive is
  `useMoveLine` + the drum-aware panel + an optional `drumId` on AddLineDialog.
- **Register-on-drum is two-step (register then move)**, and the new line is created as a
  **spare** so a failed move leaves a valid spare, never a broken half-placed active line.
- **Drum↔rope matching by id**, via new `current_drum_id`/`current_storage_id` on the list
  row (small backend add) — chosen over the original `location_label` string parse, which
  breaks on label collisions/renames.
- **Catalogue exposed onboard** per user; line-type creation stays absent (no backend
  POST) — only view + makers/products create.
- **All rope actions are view-mode**, edit mode remains pure layout editing — avoids
  mixing destructive layout edits with operational rope moves.

## Risks / open questions

- **Stale layout after move/register/inspect.** All three change drum occupancy and/or
  worst-status; every such mutation must invalidate **both** `["lines"]` and `["layout"]`
  so deck dots + panel update. (The existing inspection hook does not — fixed in step 7.)
- **Tap-target after scaling** — verify the *measured* rendered size ≥44px at the tablet
  breakpoint, not just the viewBox units (advisor's standing note: assert, don't assume).
- **Catalogue write authority onboard** — accepted per user; makers/products POST have no
  server scope guard. Documented as accepted master-data risk, future hardening if needed.
- **`AddLineDialog` refactor blast radius** — adding optional `drumId` + spare-default must
  not regress the existing register-page flow; keep the default path identical.

## Out of scope

- Backend changes are limited to B0: list-row drum ids; move XOR + same-vessel validation
  with `ErrInvalidMoveTarget`→422 mapping. The **only new Go test is the pure
  `validateMoveRequest` (XOR)** — no same-vessel/DB Go test (covered by Docker e2e). No new
  endpoints; move/turn/inspect/register/files/catalogue all exist.
- No server-side scope guard on catalogue writes (noted accepted risk above).
- No line-type creation UI (no backend endpoint).
- No deck hull/proportion redesign, no bollards/fairleads, no canvas cropping.
- No new inspection ingestion / third-party sync work.
- Not re-reviewing already-shipped FilesTab/Catalogue/Inspection code beyond a smoke test.
