import { useState, type FormEvent } from "react";
import { dateLabel } from "../../lib/format";
import {
  fileToBase64,
  useDocuments,
  useUploadCertificate,
  useUploadPhoto,
} from "./api";
import { PhotoTimeline } from "./PhotoTimeline";
import "./files.css";

const SIDES = ["n/a", "A", "B"] as const;
const CONDITIONS = ["Good", "Monitor", "Action"] as const;
const KINDS = ["certificate", "manual", "guide"] as const;

function humanSize(bytes: number): string {
  if (!bytes) return "—";
  const units = ["B", "KB", "MB", "GB"];
  let n = bytes;
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function FilesTab({ lineId }: { lineId: string }) {
  return (
    <div className="grid" style={{ gap: 24 }}>
      <PhotoSection lineId={lineId} />
      <DocumentSection lineId={lineId} />
    </div>
  );
}

function PhotoSection({ lineId }: { lineId: string }) {
  const upload = useUploadPhoto(lineId);
  const [file, setFile] = useState<File | null>(null);
  const [side, setSide] = useState<string>("n/a");
  const [condition, setCondition] = useState<string>("Good");
  const [takenAt, setTakenAt] = useState<string>("");

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!file) return;
    const file_base64 = await fileToBase64(file);
    await upload.mutateAsync({
      file_base64,
      content_type: file.type || undefined,
      taken_at: takenAt || undefined,
      side,
      condition_at_capture: condition,
    });
    setFile(null);
    setTakenAt("");
    (e.target as HTMLFormElement).reset();
  }

  return (
    <section className="card">
      <h3 style={{ marginTop: 0 }}>Condition photos</h3>
      <form className="file-form" onSubmit={submit}>
        <input
          className="input"
          type="file"
          accept="image/*"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
        />
        <label className="file-field">
          <span className="muted">Side</span>
          <select className="input" value={side} onChange={(e) => setSide(e.target.value)}>
            {SIDES.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
        </label>
        <label className="file-field">
          <span className="muted">Condition</span>
          <select className="input" value={condition} onChange={(e) => setCondition(e.target.value)}>
            {CONDITIONS.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
        </label>
        <label className="file-field">
          <span className="muted">Taken</span>
          <input className="input" type="date" value={takenAt} onChange={(e) => setTakenAt(e.target.value)} />
        </label>
        <button className="btn" type="submit" disabled={!file || upload.isPending}>
          {upload.isPending ? "Uploading…" : "Upload photo"}
        </button>
      </form>
      {upload.isError && <p className="muted">Upload failed: {(upload.error as Error).message}</p>}
      <div style={{ marginTop: 16 }}>
        <PhotoTimeline lineId={lineId} />
      </div>
    </section>
  );
}

function DocumentSection({ lineId }: { lineId: string }) {
  const { data: docs, isLoading } = useDocuments(lineId);
  const upload = useUploadCertificate(lineId);
  const [file, setFile] = useState<File | null>(null);
  const [kind, setKind] = useState<string>("certificate");

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!file) return;
    const file_base64 = await fileToBase64(file);
    await upload.mutateAsync({
      file_base64,
      file_name: file.name,
      content_type: file.type || undefined,
      kind,
    });
    setFile(null);
    (e.target as HTMLFormElement).reset();
  }

  return (
    <section className="card">
      <h3 style={{ marginTop: 0 }}>Certificates &amp; documents</h3>
      <form className="file-form" onSubmit={submit}>
        <input className="input" type="file" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
        <label className="file-field">
          <span className="muted">Kind</span>
          <select className="input" value={kind} onChange={(e) => setKind(e.target.value)}>
            {KINDS.map((k) => (
              <option key={k} value={k}>{k}</option>
            ))}
          </select>
        </label>
        <button className="btn" type="submit" disabled={!file || upload.isPending}>
          {upload.isPending ? "Uploading…" : "Upload document"}
        </button>
      </form>
      {upload.isError && <p className="muted">Upload failed: {(upload.error as Error).message}</p>}

      {isLoading ? (
        <p className="muted">Loading documents…</p>
      ) : !docs || docs.length === 0 ? (
        <div className="stub"><h3>No documents yet</h3>Upload a certificate, manual or guide above.</div>
      ) : (
        <div className="table-wrap" style={{ marginTop: 16 }}>
          <table className="grid">
            <thead>
              <tr><th>File</th><th>Kind</th><th>Size</th><th>Added</th><th></th></tr>
            </thead>
            <tbody>
              {docs.map((d) => (
                <tr key={d.id}>
                  <td>{d.file_name}</td>
                  <td>{d.kind}</td>
                  <td>{humanSize(d.size_bytes)}</td>
                  <td>{dateLabel(d.created_at)}</td>
                  <td>
                    {d.url ? (
                      <a className="btn" href={d.url} target="_blank" rel="noreferrer">Download</a>
                    ) : (
                      <span className="muted">—</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
