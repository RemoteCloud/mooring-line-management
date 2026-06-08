// SVG symbols for the deck map. Coordinates are normalized 0..1 and scaled to the
// viewBox by the parent, so everything stays responsive.
import type { Winch, Storage } from "../../api/hooks";
import { condClass } from "../../lib/format";

export const VB_W = 1000;
export const VB_H = 600;

function dotColor(status: string | undefined): string {
  switch (condClass(status as never)) {
    case "good": return "var(--good)";
    case "monitor": return "var(--monitor)";
    case "action": return "var(--action)";
    default: return "var(--border)";
  }
}

const DRUM_W = 20;
const DRUM_H = 30;
const DRUM_GAP = 4;
const PAD = 8;

export function winchBox(w: Winch) {
  const inner = w.drum_count * DRUM_W + (w.drum_count - 1) * DRUM_GAP;
  return { bw: inner + PAD * 2, bh: DRUM_H + PAD * 2, inner };
}

export function WinchSymbol({
  w, selected, onPointerDown, onClick,
}: {
  w: Winch;
  selected?: boolean;
  onPointerDown?: (e: React.PointerEvent) => void;
  onClick?: () => void;
}) {
  const cx = w.x * VB_W;
  const cy = w.y * VB_H;
  const { bw, bh, inner } = winchBox(w);
  const byIdx = new Map(w.drums.map((d) => [d.idx, d]));

  return (
    <g
      transform={`translate(${cx} ${cy}) rotate(${w.orientation})`}
      onPointerDown={onPointerDown}
      onClick={onClick}
      style={{ cursor: onPointerDown ? "grab" : "pointer" }}
    >
      <rect className={"winch-body" + (selected ? " sel" : "")} x={-bw / 2} y={-bh / 2} width={bw} height={bh} rx={8} />
      {Array.from({ length: w.drum_count }).map((_, i) => {
        const filled = (byIdx.get(i + 1)?.line_count ?? 0) > 0;
        return (
          <rect
            key={i}
            className={"drum-cell" + (filled ? " filled" : "")}
            x={-inner / 2 + i * (DRUM_W + DRUM_GAP)}
            y={-DRUM_H / 2}
            width={DRUM_W}
            height={DRUM_H}
            rx={3}
          />
        );
      })}
      {/* worst-case status dot (counter-rotated so it stays top-right visually) */}
      <circle cx={bw / 2 - 4} cy={-bh / 2 + 4} r={6} fill={dotColor(w.worst_status)} stroke="var(--bg)" strokeWidth={1.5} />
      {/* drive-type marker, top-left: E electric / H hydraulic */}
      <text
        className="sym-drive"
        x={-bw / 2 + 6}
        y={-bh / 2 + 11}
        transform={`rotate(${-w.orientation} ${-bw / 2 + 6} ${-bh / 2 + 7})`}
      >
        {w.drive_type === "hydraulic" ? "H" : "E"}
      </text>
      <text className="sym-label" x={0} y={bh / 2 + 16} transform={`rotate(${-w.orientation})`}>{w.label}</text>
    </g>
  );
}

export function StorageSymbol({
  s, selected, onPointerDown, onClick,
}: {
  s: Storage;
  selected?: boolean;
  onPointerDown?: (e: React.PointerEvent) => void;
  onClick?: () => void;
}) {
  const cx = s.x * VB_W;
  const cy = s.y * VB_H;
  const w = 70, h = 46;
  return (
    <g transform={`translate(${cx} ${cy})`} onPointerDown={onPointerDown} onClick={onClick}
       style={{ cursor: onPointerDown ? "grab" : "pointer" }}>
      <rect className={"winch-body" + (selected ? " sel" : "")} x={-w / 2} y={-h / 2} width={w} height={h} rx={6}
            strokeDasharray="5 4" />
      <text className="sym-label" x={0} y={4}>▤ {s.line_count}</text>
      <circle cx={w / 2 - 4} cy={-h / 2 + 4} r={6} fill={dotColor(s.worst_status)} stroke="var(--bg)" strokeWidth={1.5} />
      <text className="sym-label" x={0} y={h / 2 + 16}>{s.label}</text>
    </g>
  );
}

export function Hull({ station }: { station: "fwd" | "aft" }) {
  // Forward deck has a pointed bow at the top; aft deck a flat stern.
  const pts =
    station === "fwd"
      ? `${VB_W / 2},20 850,170 850,560 150,560 150,170`
      : `150,40 850,40 850,470 640,580 360,580 150,470`;
  return <polygon className="hull" points={pts} />;
}
