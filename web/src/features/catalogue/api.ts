// Catalogue (master data) API hooks. Raw fetch against the same-origin /api
// prefix; the typed client is intentionally not used here so the catalogue
// feature stays self-contained.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { API_BASE } from "../../config";

export interface Maker {
  id: string;
  name: string;
  notes?: string;
}

export interface LineType {
  id: string;
  name: string;
  description?: string;
}

export interface Product {
  id: string;
  makerId: string;
  makerName: string;
  lineTypeId: string;
  lineTypeName: string;
  productName: string;
  constructionType?: string;
  defaultLength?: number;
  swl?: number;
  breakLoad?: number;
  canBeTurned: boolean;
  manufacturerManualRef?: string;
  notes?: string;
}

export interface CreateMakerBody {
  name: string;
  notes?: string;
}

export interface CreateLineTypeBody {
  name: string;
  description?: string;
}

export interface CreateProductBody {
  makerId: string;
  lineTypeId: string;
  productName: string;
  constructionType?: string;
  defaultLength?: number;
  swl?: number;
  breakLoad?: number;
  canBeTurned: boolean;
  manufacturerManualRef?: string;
  notes?: string;
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) {
    throw new Error(`GET ${path} failed: ${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`POST ${path} failed: ${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

export function useMakers() {
  return useQuery<Maker[]>({
    queryKey: ["catalogue", "makers"],
    queryFn: () => getJSON<Maker[]>("/makers"),
  });
}

export function useLineTypes() {
  return useQuery<LineType[]>({
    queryKey: ["catalogue", "line-types"],
    queryFn: () => getJSON<LineType[]>("/line-types"),
  });
}

export function useProducts() {
  return useQuery<Product[]>({
    queryKey: ["catalogue", "products"],
    queryFn: () => getJSON<Product[]>("/products"),
  });
}

export function useCreateMaker() {
  const qc = useQueryClient();
  return useMutation<Maker, Error, CreateMakerBody>({
    mutationFn: (body: CreateMakerBody) => postJSON<Maker>("/makers", body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["catalogue", "makers"] });
    },
  });
}

export function useCreateLineType() {
  const qc = useQueryClient();
  return useMutation<LineType, Error, CreateLineTypeBody>({
    mutationFn: (body: CreateLineTypeBody) => postJSON<LineType>("/line-types", body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["catalogue", "line-types"] });
    },
  });
}

export function useCreateProduct() {
  const qc = useQueryClient();
  return useMutation<Product, Error, CreateProductBody>({
    mutationFn: (body: CreateProductBody) => postJSON<Product>("/products", body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["catalogue", "products"] });
    },
  });
}
