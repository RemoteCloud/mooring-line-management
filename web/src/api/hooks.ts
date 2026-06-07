// React Query hooks over the typed client. Types come from the generated schema.
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type { components } from "./schema";

export type Maker = components["schemas"]["Maker"];
export type LineType = components["schemas"]["LineType"];
export type Product = components["schemas"]["Product"];
export type Vessel = components["schemas"]["Vessel"];
export type Layout = components["schemas"]["Layout"];
export type Winch = components["schemas"]["Winch"];
export type Storage = components["schemas"]["Storage"];
export type LineRow = components["schemas"]["LineRow"];
export type Line = components["schemas"]["Line"];

export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const { data, error } = await api.GET("/health");
      if (error) throw error;
      return data;
    },
    refetchInterval: 15_000,
  });
}

export function useVessels() {
  return useQuery({
    queryKey: ["vessels"],
    queryFn: async () => {
      const { data, error } = await api.GET("/vessels");
      if (error) throw error;
      return data as Vessel[];
    },
  });
}

export function useProducts() {
  return useQuery({
    queryKey: ["products"],
    queryFn: async () => {
      const { data, error } = await api.GET("/products");
      if (error) throw error;
      return data as Product[];
    },
  });
}

export function useLineTypes() {
  return useQuery({
    queryKey: ["line-types"],
    queryFn: async () => {
      const { data, error } = await api.GET("/line-types");
      if (error) throw error;
      return data as LineType[];
    },
  });
}

export function useVesselLayout(vesselId: string | undefined) {
  return useQuery({
    enabled: !!vesselId,
    queryKey: ["layout", vesselId],
    queryFn: async () => {
      const { data, error } = await api.GET("/vessels/{vessel_id}/layout", {
        params: { path: { vessel_id: vesselId! } },
      });
      if (error) throw error;
      return data as Layout;
    },
  });
}

export type LineFilters = {
  line_type_id?: string;
  condition?: "Good" | "Monitor" | "Action";
  placement?: "installed" | "spare";
  q?: string;
};

export function useLines(vesselId: string | undefined, filters: LineFilters) {
  return useQuery({
    enabled: !!vesselId,
    queryKey: ["lines", vesselId, filters],
    queryFn: async () => {
      const { data, error } = await api.GET("/vessels/{vessel_id}/lines", {
        params: {
          path: { vessel_id: vesselId! },
          query: { ...filters, limit: 500 },
        },
      });
      if (error) throw error;
      return data as { items: LineRow[]; total: number };
    },
  });
}

export function useLine(id: string | undefined) {
  return useQuery({
    enabled: !!id,
    queryKey: ["line", id],
    queryFn: async () => {
      const { data, error } = await api.GET("/lines/{id}", {
        params: { path: { id: id! } },
      });
      if (error) throw error;
      return data as Line;
    },
  });
}

export function useRegisterLine(vesselId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: components["schemas"]["LineBody"]) => {
      const { data, error } = await api.POST("/vessels/{vessel_id}/lines", {
        params: { path: { vessel_id: vesselId } },
        body,
      });
      if (error) throw error;
      return data as Line;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["lines"] }),
  });
}

export function useSaveLayout(vesselId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { winches: unknown[]; storage: unknown[] }) => {
      const { data, error } = await api.PUT("/vessels/{vessel_id}/layout", {
        params: { path: { vessel_id: vesselId } },
        body: body as never,
      });
      if (error) throw error;
      return data as Layout;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["layout", vesselId] }),
  });
}
