// Files slice — photos + certificates/documents over S3-backed endpoints.
// These routes are not in the typed OpenAPI client, so we use raw fetch.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { API_BASE } from "../../config";

export interface FilePhoto {
  id: string;
  line_id: string;
  inspection_id?: string;
  file_ref: string;
  taken_at?: string;
  side?: string;
  condition_at_capture?: string;
  created_at: string;
  url?: string;
}

export interface FileDoc {
  id: string;
  line_id?: string;
  product_id?: string;
  vessel_id?: string;
  kind: string;
  file_ref: string;
  file_name: string;
  content_type?: string;
  size_bytes: number;
  created_at: string;
  url?: string;
}

async function jsonOrThrow<T>(resOrPromise: Response | Promise<Response>): Promise<T> {
  const res = await resOrPromise;
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      detail = body?.detail || body?.title || detail;
    } catch {
      /* ignore */
    }
    throw new Error(detail);
  }
  return res.json() as Promise<T>;
}

// fileToBase64 reads a File and returns the raw base64 (data-URL prefix stripped).
export function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      const comma = result.indexOf(",");
      resolve(comma >= 0 ? result.slice(comma + 1) : result);
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}

export function usePhotos(lineId: string) {
  return useQuery({
    queryKey: ["photos", lineId],
    queryFn: () => jsonOrThrow<FilePhoto[]>(fetch(`${API_BASE}/lines/${lineId}/photos`)),
  });
}

export interface UploadPhotoInput {
  file_base64: string;
  content_type?: string;
  taken_at?: string;
  side?: string;
  condition_at_capture?: string;
  inspection_id?: string;
}

// postPhoto / postDocument take the line id as a parameter so callers that only
// learn the id at submit time (uploading right after registering a line) can reuse
// the exact same request as the hooks below.
export function postPhoto(lineId: string, input: UploadPhotoInput): Promise<FilePhoto> {
  return jsonOrThrow<FilePhoto>(
    fetch(`${API_BASE}/lines/${lineId}/photos`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }),
  );
}

export function useUploadPhoto(lineId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UploadPhotoInput) => postPhoto(lineId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["photos", lineId] }),
  });
}

export function useDeletePhoto(lineId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (photoId: string) => {
      const res = await fetch(`${API_BASE}/photos/${photoId}`, { method: "DELETE" });
      if (!res.ok) throw new Error(res.statusText);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["photos", lineId] }),
  });
}

export function useDocuments(lineId: string) {
  return useQuery({
    queryKey: ["documents", lineId],
    queryFn: () => jsonOrThrow<FileDoc[]>(fetch(`${API_BASE}/lines/${lineId}/files`)),
  });
}

export interface UploadCertificateInput {
  file_base64: string;
  file_name: string;
  content_type?: string;
  kind?: string;
}

export function postDocument(lineId: string, input: UploadCertificateInput): Promise<FileDoc> {
  return jsonOrThrow<FileDoc>(
    fetch(`${API_BASE}/lines/${lineId}/certificate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(input),
    }),
  );
}

export function useUploadCertificate(lineId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UploadCertificateInput) => postDocument(lineId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["documents", lineId] }),
  });
}
