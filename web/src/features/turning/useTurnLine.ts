import { useMutation, useQueryClient } from "@tanstack/react-query";

import { API_BASE } from "../../config";

// useTurnLine posts a turn for a line. The endpoint is not yet in the generated
// typed schema, so this uses a raw fetch. On success it invalidates the line's
// record and the register list so derived ages/sides refresh.
export function useTurnLine(lineId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { note?: string }) => {
      const res = await fetch(`${API_BASE}/lines/${lineId}/turn`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      });
      if (!res.ok) {
        throw new Error(`Turn failed (${res.status})`);
      }
      return res.json();
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["line", lineId] });
      qc.invalidateQueries({ queryKey: ["lines"] });
    },
  });
}
