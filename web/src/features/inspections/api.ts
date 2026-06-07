// Inspections data layer. These endpoints are not yet in the generated typed schema,
// so we hit them with raw fetch wrapped in react-query hooks.
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { API_BASE } from "../../config";

export interface Inspection {
  id: string;
  line_id: string;
  vessel_id: string;
  inspected_at: string;
  inspected_by?: string;
  source: string;
  external_id?: string;
  condition_status: string;
  notes?: string;
  created_at: string;
}

export interface InspLogbookEntry extends Inspection {
  line_name: string;
  serial_number: string;
}

export interface LogInspectionInput {
  condition_status: string;
  inspected_by?: string;
  notes?: string;
  inspected_at?: string;
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`request failed: ${res.status}`);
  return (await res.json()) as T;
}

export function useInspections(lineId: string | undefined) {
  return useQuery({
    enabled: !!lineId,
    queryKey: ["inspections", lineId],
    queryFn: () => getJSON<Inspection[]>(`${API_BASE}/lines/${lineId}/inspections`),
  });
}

export function useLogInspection(lineId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (input: LogInspectionInput) => {
      const res = await fetch(`${API_BASE}/lines/${lineId}/inspections`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      });
      if (!res.ok) throw new Error(`request failed: ${res.status}`);
      return (await res.json()) as Inspection;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inspections", lineId] });
      qc.invalidateQueries({ queryKey: ["line", lineId] });
    },
  });
}

export function useLogbook(vesselId: string | undefined) {
  return useQuery({
    enabled: !!vesselId,
    queryKey: ["logbook", vesselId],
    queryFn: () => getJSON<InspLogbookEntry[]>(`${API_BASE}/inspections/logbook?vessel_id=${vesselId}`),
  });
}
