import { useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useVessel } from "../../app/VesselContext";
import { useVesselLayout, useLines, useSaveLayout, type Layout, type Winch, type Storage } from "../../api/hooks";
import { WinchSymbol, StorageSymbol, Hull, VB_W, VB_H } from "./symbols";
import { StatusDot } from "../../components/ui";
import { WriteGuard } from "../../app/auth/WriteGuard";

type Station = "fwd" | "aft";
const ORIENTATIONS = [0, 45, -45, 90, -90];

const clone = (l: Layout): Layout => JSON.parse(JSON.stringify(l));

export function DeckPage() {
  const { vesselId } = useVessel();
  const navigate = useNavigate();
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
  const storage = (active?.storage ?? []).filter((s) => s.station === station);

  // selected entity, tracked by a stable key so new (id="") items can be selected
  const [selKey, setSelKey] = useState<string | null>(null);

  const enterEdit = () => { if (layout) { setDraft(clone(layout)); setEdit(true); setSelKey(null); } };
  const cancelEdit = () => { setEdit(false); setDraft(null); setSelKey(null); };
  const resetEdit = () => { if (layout) setDraft(clone(layout)); };
  const doSave = async () => {
    if (!draft) return;
    await save.mutateAsync({
      winches: draft.winches.map((w) => ({
        id: w.id || undefined, label: w.label, station: w.station,
        x: w.x, y: w.y, orientation: w.orientation, drum_count: w.drum_count,
      })),
      storage: draft.storage.map((s) => ({
        id: s.id || undefined, label: s.label, station: s.station, x: s.x, y: s.y,
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
      const item = arr.find((i) => keyOf(i) === d.id);
      if (item) { item.x = x; item.y = y; }
      return next;
    });
  };

  const keyOf = (i: Winch | Storage) => i.id || (i as Winch).label + ":" + i.station;

  const addWinch = () => setDraft((p) => p && ({ ...clone(p), winches: [...p.winches, {
    id: "", label: "New winch", station, x: 0.5, y: 0.5, orientation: 0, drum_count: 2, drums: [], worst_status: "",
  }] }));
  const addStorage = () => setDraft((p) => p && ({ ...clone(p), storage: [...p.storage, {
    id: "", label: "New store", station, x: 0.5, y: 0.5, line_count: 0, worst_status: "",
  }] }));

  const updateWinch = (key: string, patch: Partial<Winch>) =>
    setDraft((p) => { if (!p) return p; const n = clone(p); const w = n.winches.find((w) => keyOf(w) === key); if (w) Object.assign(w, patch); return n; });
  const updateStorage = (key: string, patch: Partial<Storage>) =>
    setDraft((p) => { if (!p) return p; const n = clone(p); const s = n.storage.find((s) => keyOf(s) === key); if (s) Object.assign(s, patch); return n; });
  const removeSel = () => setDraft((p) => {
    if (!p || !selKey) return p; const n = clone(p);
    n.winches = n.winches.filter((w) => keyOf(w) !== selKey);
    n.storage = n.storage.filter((s) => keyOf(s) !== selKey);
    return n;
  });

  const editWinch = draft?.winches.find((w) => keyOf(w) === selKey);
  const editStorage = draft?.storage.find((s) => keyOf(s) === selKey);

  // view-mode panel: ropes at the selected symbol
  const viewLines = useMemo(() => {
    if (edit || !selKey || !lines) return [];
    const w = layout?.winches.find((w) => keyOf(w) === selKey);
    const s = layout?.storage.find((s) => keyOf(s) === selKey);
    if (w) return lines.items.filter((l) => l.location_label.startsWith(w.label + " "));
    if (s) return lines.items.filter((l) => l.location_label === s.label);
    return [];
  }, [edit, selKey, lines, layout]);

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
          <WriteGuard>
            <button className="btn ghost" onClick={enterEdit} disabled={!layout}>Edit layout</button>
          </WriteGuard>
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
              onClick={() => setSelKey(keyOf(w))}
              onPointerDown={edit ? (e) => { drag.current = { id: keyOf(w), kind: "winch" }; setSelKey(keyOf(w)); (e.target as Element).setPointerCapture?.(e.pointerId); } : undefined}
            />
          ))}
          {storage.map((s) => (
            <StorageSymbol
              key={keyOf(s)}
              s={s}
              selected={selKey === keyOf(s)}
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
              <p className="muted">Select a symbol to edit, or drag to reposition. Add winches/storage from the toolbar.</p>
            )
          ) : selKey ? (
            <>
              <h3 style={{ marginTop: 0 }}>{(layout?.winches.find((w) => keyOf(w) === selKey)?.label) || (layout?.storage.find((s) => keyOf(s) === selKey)?.label)}</h3>
              {viewLines.length === 0 && <p className="muted">No ropes here.</p>}
              {viewLines.map((l) => (
                <div key={l.id} className="list-item" onClick={() => navigate(`/lines/${l.id}`)}>
                  <StatusDot condition={l.current_condition_status as never} />
                  <div style={{ flex: 1 }}>
                    <div>{l.name}</div>
                    <div className="muted" style={{ fontSize: 12 }}>{l.location_label} · {l.serial_number}</div>
                  </div>
                </div>
              ))}
            </>
          ) : (
            <p className="muted">Click a winch or storage to see its ropes. Worst-case status shown by the dot.</p>
          )}
        </div>
      </div>
    </>
  );
}

function WinchEditor({ w, onChange, onRemove }: { w: Winch; onChange: (p: Partial<Winch>) => void; onRemove: () => void }) {
  return (
    <>
      <h3 style={{ marginTop: 0 }}>Edit winch</h3>
      <div className="field"><label>Label</label><input className="input" value={w.label} onChange={(e) => onChange({ label: e.target.value })} /></div>
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
      <h3 style={{ marginTop: 0 }}>Edit storage</h3>
      <div className="field"><label>Label</label><input className="input" value={s.label} onChange={(e) => onChange({ label: e.target.value })} /></div>
      <div className="field">
        <label>Station</label>
        <select className="input" value={s.station} onChange={(e) => onChange({ station: e.target.value })}>
          <option value="fwd">Forward</option><option value="aft">Aft</option>
        </select>
      </div>
      <button className="btn danger" onClick={onRemove} style={{ marginTop: 10 }}>Remove storage</button>
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
