import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useVessel } from "../../app/VesselContext";
import { useVesselLayout, useLines, useSaveLayout, type Layout, type Winch, type Storage } from "../../api/hooks";
import { WinchSymbol, StorageSymbol, Hull, VB_W, VB_H } from "./symbols";
import { WinchPanel, StoragePanel } from "./WinchPanel";
import { StatusDot } from "../../components/ui";

type Station = "fwd" | "aft";
const ORIENTATIONS = [0, 45, -45, 90, -90];

const clone = (l: Layout): Layout => JSON.parse(JSON.stringify(l));

// A new, unsaved symbol carries a "tmp:" id so its key stays stable while we
// relabel it; doSave strips the prefix back to undefined for the API.
// crypto.randomUUID is unavailable in insecure contexts (http over a LAN IP,
// which is exactly how onboard tablets reach the PWA), so fall back.
let tmpSeq = 0;
const tmpID = () => {
  const rand = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${++tmpSeq}`;
  return "tmp:" + rand;
};
const isTmp = (id: string) => id.startsWith("tmp:");

// Deck side from the normalized x: the map is a top-down plan with the bow up,
// so port is to the left and starboard to the right. A center band covers
// keel-line winches.
type Side = "P" | "S" | "C";
function sideOf(x: number): Side {
  if (x < 0.45) return "P";
  if (x > 0.55) return "S";
  return "C";
}
const SIDE_NAME: Record<Side, string> = { P: "Port", S: "Starboard", C: "Center" };
const STATION_NAME: Record<Station, string> = { fwd: "Forward", aft: "Aft" };

// positionLabel describes a winch's derived placement, e.g. "Forward · Starboard".
export function positionLabel(station: string, x: number): string {
  return `${STATION_NAME[station as Station] ?? station} · ${SIDE_NAME[sideOf(x)]}`;
}

// renumberAuto rewrites the label of every auto-named winch to
// <STATION>-<SIDE><n>, numbered per station+side in array order. Hand-named
// winches (label_auto=false) keep their label and are skipped.
function renumberAuto(winches: Winch[]) {
  const counters: Record<string, number> = {};
  for (const w of winches) {
    if (!w.label_auto) continue;
    const side = sideOf(w.x);
    const k = w.station + side;
    counters[k] = (counters[k] ?? 0) + 1;
    w.label = `${w.station.toUpperCase()}-${side}${counters[k]}`;
  }
}

export function DeckPage() {
  const { vesselId } = useVessel();
  const { data: layout } = useVesselLayout(vesselId);
  const { data: lines } = useLines(vesselId, {});
  const save = useSaveLayout(vesselId ?? "");

  const [station, setStation] = useState<Station>("fwd");
  const [edit, setEdit] = useState(false);
  const [draft, setDraft] = useState<Layout | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const drag = useRef<{ id: string; kind: "winch" | "storage" } | null>(null);

  const active = edit ? draft : layout;
  const winches = (active?.winches ?? []).filter((w) => w.station === station);
  // On-map storage is drawn on the SVG for the current station; off-map "areas" are
  // vessel-wide text locations listed under the map regardless of station.
  const storage = (active?.storage ?? []).filter((s) => s.on_map && s.station === station);
  const areas = (active?.storage ?? []).filter((s) => !s.on_map);

  const [selKey, setSelKey] = useState<string | null>(null);
  // Drum picked in the side panel; glows green on the selected winch's symbol. Only
  // meaningful paired with selKey, so it clears whenever the selection/station/mode changes.
  const [selDrumIdx, setSelDrumIdx] = useState<number | null>(null);
  useEffect(() => setSelDrumIdx(null), [selKey, station, edit]);

  // A line can be deep-linked from its rope record (/deck?line=<id>); resolve where
  // it physically sits so we can jump to that station and glow the exact drum/store.
  const [sp] = useSearchParams();
  const focusLineId = sp.get("line");
  const focus = useMemo(() => {
    if (!focusLineId || !layout) return null;
    const line = lines?.items.find((l) => l.id === focusLineId);
    if (!line) return null;
    if (line.current_drum_id) {
      for (const w of layout.winches) {
        const d = w.drums.find((d) => d.id === line.current_drum_id);
        if (d) return { kind: "winch" as const, id: w.id, drumIdx: d.idx, station: w.station as Station };
      }
    }
    if (line.current_storage_id) {
      const s = layout.storage.find((s) => s.id === line.current_storage_id);
      // Off-map areas are vessel-wide, so don't force a station for them.
      if (s) return { kind: "storage" as const, id: s.id, station: s.on_map ? (s.station as Station) : undefined };
    }
    return null;
  }, [focusLineId, layout, lines]);

  // When a line is focused, jump to its station and select it (view mode only).
  useEffect(() => {
    if (!focus || edit) return;
    if (focus.station) setStation(focus.station);
    setSelKey(focus.id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focus]);

  const enterEdit = () => { if (layout) { setDraft(clone(layout)); setEdit(true); setSelKey(null); } };
  const cancelEdit = () => { setEdit(false); setDraft(null); setSelKey(null); };
  const resetEdit = () => { if (layout) setDraft(clone(layout)); };
  const doSave = async () => {
    if (!draft) return;
    await save.mutateAsync({
      winches: draft.winches.map((w) => ({
        id: isTmp(w.id) ? undefined : w.id, label: w.label, station: w.station,
        x: w.x, y: w.y, orientation: w.orientation, drum_count: w.drum_count,
        drive_type: w.drive_type, label_auto: w.label_auto,
      })),
      storage: draft.storage.map((s) => ({
        id: isTmp(s.id) ? undefined : s.id, label: s.label, station: s.on_map ? s.station : "",
        on_map: s.on_map, x: s.x, y: s.y,
      })),
    });
    cancelEdit();
  };

  // pointer drag to reposition in edit mode
  const toNorm = (e: React.PointerEvent) => {
    const svg = svgRef.current!;
    const ctm = svg.getScreenCTM()!;
    const p = new DOMPoint(e.clientX, e.clientY).matrixTransform(ctm.inverse());
    return { x: Math.min(1, Math.max(0, p.x / VB_W)), y: Math.min(1, Math.max(0, p.y / VB_H)) };
  };
  const onMove = (e: React.PointerEvent) => {
    if (!edit || !drag.current || !draft) return;
    const { x, y } = toNorm(e);
    const d = drag.current;
    setDraft((prev) => {
      if (!prev) return prev;
      const next = clone(prev);
      const arr = d.kind === "winch" ? next.winches : next.storage;
      const item = arr.find((i) => i.id === d.id);
      if (item) { item.x = x; item.y = y; }
      // dragging across the centerline can change a winch's port/starboard side,
      // so re-derive auto names live.
      if (d.kind === "winch") renumberAuto(next.winches);
      return next;
    });
  };

  const keyOf = (i: Winch | Storage) => i.id;

  const addWinch = () => setDraft((p) => {
    if (!p) return p;
    const n = clone(p);
    const w: Winch = {
      id: tmpID(), label: "", station, x: 0.5, y: 0.5, orientation: 0,
      drum_count: 2, drive_type: "electric", label_auto: true, drums: [], worst_status: "",
    };
    n.winches.push(w);
    renumberAuto(n.winches);
    setSelKey(w.id);
    return n;
  });
  const addStorage = () => setDraft((p) => {
    if (!p) return p;
    const n = clone(p);
    const s: Storage = { id: tmpID(), label: "New store", station, on_map: true, x: 0.5, y: 0.5, line_count: 0, worst_status: "" };
    n.storage.push(s);
    setSelKey(s.id);
    return n;
  });
  // An off-map area is a text-only storage location (no deck position, no station).
  const addStorageArea = () => setDraft((p) => {
    if (!p) return p;
    const n = clone(p);
    const s: Storage = { id: tmpID(), label: "New area", station: "", on_map: false, x: 0.5, y: 0.5, line_count: 0, worst_status: "" };
    n.storage.push(s);
    setSelKey(s.id);
    return n;
  });

  const updateWinch = (key: string, patch: Partial<Winch>) =>
    setDraft((p) => {
      if (!p) return p;
      const n = clone(p);
      const w = n.winches.find((w) => w.id === key);
      if (w) {
        Object.assign(w, patch);
        // station/side or auto-toggle changes can shift the derived numbering
        if ("station" in patch || "label_auto" in patch) renumberAuto(n.winches);
      }
      return n;
    });
  const updateStorage = (key: string, patch: Partial<Storage>) =>
    setDraft((p) => { if (!p) return p; const n = clone(p); const s = n.storage.find((s) => s.id === key); if (s) Object.assign(s, patch); return n; });
  const removeSel = () => setDraft((p) => {
    if (!p || !selKey) return p; const n = clone(p);
    n.winches = n.winches.filter((w) => w.id !== selKey);
    n.storage = n.storage.filter((s) => s.id !== selKey);
    renumberAuto(n.winches);
    return n;
  });

  const editWinch = draft?.winches.find((w) => w.id === selKey);
  const editStorage = draft?.storage.find((s) => s.id === selKey);

  const selWinch = !edit ? layout?.winches.find((w) => w.id === selKey) : undefined;
  const selStorage = !edit ? layout?.storage.find((s) => s.id === selKey) : undefined;

  return (
    <>
      <div className="deck-toolbar">
        <h1 className="page-title" style={{ margin: 0 }}>Deck map</h1>
        <div className="seg">
          {(["fwd", "aft"] as Station[]).map((s) => (
            <button key={s} className={station === s ? "active" : ""} onClick={() => { setStation(s); setSelKey(null); }}>
              {s === "fwd" ? "Forward" : "Aft"}
            </button>
          ))}
        </div>
        <div className="grow" style={{ flex: 1 }} />
        {!edit ? (
          <button className="btn ghost" onClick={enterEdit} disabled={!layout}>Edit layout</button>
        ) : (
          <>
            <button className="btn ghost" onClick={addWinch}>+ Winch</button>
            <button className="btn ghost" onClick={addStorage}>+ Storage</button>
            <button className="btn ghost" onClick={resetEdit}>Reset</button>
            <button className="btn ghost" onClick={cancelEdit}>Cancel</button>
            <button className="btn" onClick={doSave} disabled={save.isPending}>{save.isPending ? "Saving…" : "Save"}</button>
          </>
        )}
      </div>

      {save.isError && <div className="err" style={{ marginBottom: 10 }}>Save failed — a removed winch/drum may still hold a line.</div>}

      <div className="deck-wrap">
        <svg
          ref={svgRef}
          className="deck-svg"
          viewBox={`0 0 ${VB_W} ${VB_H}`}
          preserveAspectRatio="xMidYMid meet"
          onPointerMove={onMove}
          onPointerUp={() => (drag.current = null)}
          onPointerLeave={() => (drag.current = null)}
        >
          <Hull station={station} />
          {winches.map((w) => (
            <WinchSymbol
              key={keyOf(w)}
              w={w}
              selected={selKey === keyOf(w)}
              highlightDrumIdx={!edit && focus?.kind === "winch" && focus.id === w.id ? focus.drumIdx : undefined}
              selectedDrumIdx={!edit && selKey === w.id ? selDrumIdx ?? undefined : undefined}
              onClick={() => setSelKey(keyOf(w))}
              onPointerDown={edit ? (e) => { drag.current = { id: keyOf(w), kind: "winch" }; setSelKey(keyOf(w)); (e.target as Element).setPointerCapture?.(e.pointerId); } : undefined}
            />
          ))}
          {storage.map((s) => (
            <StorageSymbol
              key={keyOf(s)}
              s={s}
              selected={selKey === keyOf(s)}
              highlighted={!edit && focus?.kind === "storage" && focus.id === s.id}
              onClick={() => setSelKey(keyOf(s))}
              onPointerDown={edit ? (e) => { drag.current = { id: keyOf(s), kind: "storage" }; setSelKey(keyOf(s)); (e.target as Element).setPointerCapture?.(e.pointerId); } : undefined}
            />
          ))}
        </svg>

        <div className="deck-side">
          {edit ? (
            editWinch ? (
              <WinchEditor
                w={editWinch}
                onChange={(patch) => updateWinch(selKey!, patch)}
                onRemove={removeSel}
              />
            ) : editStorage ? (
              <StorageEditor s={editStorage} onChange={(patch) => updateStorage(selKey!, patch)} onRemove={removeSel} />
            ) : (
              <p className="muted">Select a symbol to edit, or drag to reposition. Add winches/storage from the toolbar — new winches are auto-named from their deck position.</p>
            )
          ) : selWinch && layout && vesselId ? (
            <WinchPanel winch={selWinch} layout={layout} lines={lines?.items ?? []} vesselId={vesselId} selectedDrumIdx={selDrumIdx} onSelectDrum={setSelDrumIdx} />
          ) : selStorage && layout && vesselId ? (
            <StoragePanel storage={selStorage} layout={layout} lines={lines?.items ?? []} vesselId={vesselId} />
          ) : (
            <p className="muted">Click a winch or storage to manage its ropes. Worst-case status shown by the corner marker: ● Good, ◆ Monitor, ▲ Action.</p>
          )}
        </div>
      </div>

      {(edit || areas.length > 0) && (
        <div className="other-storage">
          <div className="other-storage-head">Other storage{edit ? " — text areas (off the deck plan)" : ""}</div>
          <div className="store-chips">
            {areas.map((a) => (
              <button
                key={a.id}
                className={"store-chip" + (selKey === a.id ? " sel" : "")}
                onClick={() => setSelKey(a.id)}
              >
                <StatusDot condition={a.worst_status as never} />
                <span className="store-chip-label">{a.label || "Untitled area"}</span>
                <span className="store-chip-count">{a.line_count}</span>
              </button>
            ))}
            {edit && (
              <button className="store-chip add" onClick={addStorageArea}>+ Add area</button>
            )}
            {!edit && areas.length === 0 && <span className="muted">No other storage areas.</span>}
          </div>
        </div>
      )}
    </>
  );
}

function WinchEditor({ w, onChange, onRemove }: { w: Winch; onChange: (p: Partial<Winch>) => void; onRemove: () => void }) {
  return (
    <>
      <h3 style={{ marginTop: 0 }}>Edit winch</h3>
      <p className="muted" style={{ marginTop: -6 }}>Position: <b>{positionLabel(w.station, w.x)}</b></p>
      <div className="field">
        <label>
          Name
          {w.label_auto
            ? <span className="muted" style={{ fontWeight: 400 }}> · auto from position</span>
            : <button className="linkbtn" style={{ marginLeft: 8 }} onClick={() => onChange({ label_auto: true })}>use auto</button>}
        </label>
        <input
          className="input"
          value={w.label}
          placeholder="auto-named"
          onChange={(e) => onChange({ label: e.target.value, label_auto: false })}
        />
      </div>
      <div className="field">
        <label>Drive</label>
        <div className="seg">
          {(["electric", "hydraulic"] as const).map((d) => (
            <button key={d} className={w.drive_type === d ? "active" : ""} onClick={() => onChange({ drive_type: d })}>
              {d === "electric" ? "Electric" : "Hydraulic"}
            </button>
          ))}
        </div>
      </div>
      <div className="field">
        <label>Station</label>
        <select className="input" value={w.station} onChange={(e) => onChange({ station: e.target.value })}>
          <option value="fwd">Forward</option><option value="aft">Aft</option>
        </select>
      </div>
      <div className="field">
        <label>Drums ({w.drum_count})</label>
        <div className="stepper">
          <button onClick={() => onChange({ drum_count: Math.max(1, w.drum_count - 1) })}>−</button>
          <b>{w.drum_count}</b>
          <button onClick={() => onChange({ drum_count: Math.min(6, w.drum_count + 1) })}>+</button>
        </div>
      </div>
      <div className="field">
        <label>Rotation</label>
        <div className="preset-row">
          {ORIENTATIONS.map((o) => (
            <button key={o} className={w.orientation === o ? "active" : ""} onClick={() => onChange({ orientation: o })}>
              {o > 0 ? `+${o}` : o}°
            </button>
          ))}
        </div>
        <Compass deg={w.orientation} />
      </div>
      <button className="btn danger" onClick={onRemove} style={{ marginTop: 10 }}>Remove winch</button>
    </>
  );
}

function StorageEditor({ s, onChange, onRemove }: { s: Storage; onChange: (p: Partial<Storage>) => void; onRemove: () => void }) {
  return (
    <>
      <h3 style={{ marginTop: 0 }}>{s.on_map ? "Edit storage" : "Edit storage area"}</h3>
      <div className="field"><label>Name</label><input className="input" value={s.label} onChange={(e) => onChange({ label: e.target.value })} placeholder={s.on_map ? "" : "e.g. Under mooring deck"} /></div>
      {s.on_map ? (
        <div className="field">
          <label>Station</label>
          <select className="input" value={s.station} onChange={(e) => onChange({ station: e.target.value })}>
            <option value="fwd">Forward</option><option value="aft">Aft</option>
          </select>
        </div>
      ) : (
        <p className="muted" style={{ marginTop: -2 }}>Vessel-wide · not drawn on the deck map.</p>
      )}
      <button className="btn danger" onClick={onRemove} style={{ marginTop: 10 }}>{s.on_map ? "Remove storage" : "Remove area"}</button>
    </>
  );
}

function Compass({ deg }: { deg: number }) {
  return (
    <svg width="60" height="60" viewBox="-30 -30 60 60" style={{ marginTop: 8 }}>
      <circle r="26" fill="var(--bg)" stroke="var(--border)" />
      <g transform={`rotate(${deg})`}>
        <line x1="0" y1="14" x2="0" y2="-18" stroke="var(--accent)" strokeWidth="3" />
        <polygon points="0,-22 -5,-12 5,-12" fill="var(--accent)" />
      </g>
    </svg>
  );
}
