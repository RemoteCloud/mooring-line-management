import { useTurnLine } from "./useTurnLine";

type TurnableLine = {
  id: string;
  can_be_turned: boolean;
  current_side?: string;
  current_condition_status?: string;
};

// TurnButton flips a line to its other side. Disabled unless the line is
// turnable and currently installed on a definite side (A or B).
export function TurnButton({ line }: { line: TurnableLine }) {
  const turn = useTurnLine(line.id);

  const disabled =
    !line.can_be_turned ||
    line.current_side === "n/a" ||
    !line.current_side;

  function onClick() {
    if (turn.isPending) return;
    if (!window.confirm("Turn this line to its other side?")) return;
    turn.mutate({});
  }

  return (
    <div className="card">
      <button
        className="btn"
        disabled={disabled || turn.isPending}
        onClick={onClick}
      >
        {turn.isPending ? "Turning…" : "Turn to other side"}
      </button>
      {disabled && (
        <p className="muted">This line cannot be turned.</p>
      )}
      {turn.isError && (
        <p className="muted">
          {turn.error instanceof Error ? turn.error.message : "Turn failed."}
        </p>
      )}
    </div>
  );
}
