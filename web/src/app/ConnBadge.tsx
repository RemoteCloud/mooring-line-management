// Live API/DB connectivity indicator, driven by the real /health endpoint.
import { useHealth } from "../api/hooks";

export function ConnBadge() {
  const { data, isError } = useHealth();
  const ok = !isError && data?.db === "ok";
  return (
    <span className={"pill " + (ok ? "ok" : "down")} title="API / database connectivity">
      {ok ? "● online" : "● offline"}
    </span>
  );
}
