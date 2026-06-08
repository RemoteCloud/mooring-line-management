import { useRef, useState } from "react";
import { useProducts, useRegisterLine, useMoveLine, type MoveError } from "../../api/hooks";
import { postPhoto, postDocument, fileToBase64 } from "../files/api";

type Attachment = { file: File; kind: "photo" | "delivery" };

// AddLineDialog registers a line. When `targetDrumId` is given (the deck
// "register here" flow), the line is created as a spare and then moved onto that
// drum — so a failed move leaves a valid spare, never a half-placed active line.
export function AddLineDialog({
  vesselId, onClose, targetDrumId, targetLabel,
}: {
  vesselId: string;
  onClose: () => void;
  targetDrumId?: string;
  targetLabel?: string;
}) {
  const { data: products = [] } = useProducts();
  const register = useRegisterLine(vesselId);
  const move = useMoveLine(vesselId);
  const placing = !!targetDrumId;
  const [placeErr, setPlaceErr] = useState<string | null>(null);

  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadErr, setUploadErr] = useState<string | null>(null);
  const docInput = useRef<HTMLInputElement>(null);
  const photoInput = useRef<HTMLInputElement>(null);

  const addFiles = (kind: Attachment["kind"]) => (e: React.ChangeEvent<HTMLInputElement>) => {
    const picked = Array.from(e.target.files ?? []).map((file) => ({ file, kind }));
    if (picked.length) setAttachments((a) => [...a, ...picked]);
    e.target.value = ""; // allow re-picking the same file
  };
  const removeAttachment = (i: number) => setAttachments((a) => a.filter((_, idx) => idx !== i));

  // Upload staged files to a freshly-created line. Never throws: the line already
  // exists, so a failed upload is surfaced, not fatal. Returns the failure count.
  const uploadAttachments = async (lineId: string): Promise<number> => {
    let failed = 0;
    for (const { file, kind } of attachments) {
      try {
        const file_base64 = await fileToBase64(file);
        if (kind === "photo") {
          await postPhoto(lineId, { file_base64, content_type: file.type || undefined });
        } else {
          await postDocument(lineId, {
            file_base64, file_name: file.name,
            content_type: file.type || undefined, kind: "delivery",
          });
        }
      } catch {
        failed += 1;
      }
    }
    return failed;
  };

  const [form, setForm] = useState({
    product_id: "",
    name: "",
    serial_number: "",
    tag_number: "",
    certificate_number: "",
    length: "",
    manufacture_date: "",
    installation_date: "",
    lifecycle_status: "active" as "active" | "ordered" | "spare",
  });
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm({ ...form, [k]: e.target.value });

  const selected = products.find((p) => p.id === form.product_id);

  const submit = async () => {
    setPlaceErr(null);
    setUploadErr(null);
    const line = await register.mutateAsync({
      product_id: form.product_id,
      name: form.name,
      serial_number: form.serial_number,
      tag_number: form.tag_number || undefined,
      certificate_number: form.certificate_number || undefined,
      length: form.length ? Number(form.length) : undefined,
      manufacture_date: form.manufacture_date || undefined,
      installation_date: form.installation_date || undefined,
      // when landing on a drum, create as spare; the move flips it to active.
      lifecycle_status: placing ? "spare" : form.lifecycle_status,
    } as never);
    if (placing && targetDrumId) {
      try {
        await move.mutateAsync({ lineId: line.id, toDrumId: targetDrumId });
      } catch (e) {
        // line exists as a valid spare; surface the placement failure and stop.
        setPlaceErr((e as MoveError)?.message ?? "Registered as spare, but placing on the drum failed.");
        return;
      }
    }
    if (attachments.length) {
      setUploading(true);
      const failed = await uploadAttachments(line.id);
      setUploading(false);
      if (failed > 0) {
        // line is saved; let the user retry the rest from the rope record.
        setUploadErr(`${failed} of ${attachments.length} attachment${attachments.length === 1 ? "" : "s"} failed to upload.`);
        setAttachments([]);
        return;
      }
    }
    onClose();
  };

  const valid = form.product_id && form.name && form.serial_number;

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>{placing ? "Register line onto drum" : "Add mooring line"}</h3>
        {placing && (
          <p className="muted" style={{ marginTop: -6 }}>
            New line lands on <b>{targetLabel ?? "the selected drum"}</b>.
          </p>
        )}

        <div className="field">
          <label>Product</label>
          <select className="input" value={form.product_id} onChange={set("product_id")}>
            <option value="">Select product…</option>
            {products.map((p) => (
              <option key={p.id} value={p.id}>
                {p.product_name} — {p.maker_name} ({p.line_type_name})
              </option>
            ))}
          </select>
          {selected && (
            <span className="muted" style={{ fontSize: 12 }}>
              Maker {selected.maker_name} · type {selected.line_type_name}
              {selected.construction_type ? ` · ${selected.construction_type}` : ""}
            </span>
          )}
        </div>

        <div className="row2">
          <div className="field"><label>Name / identification</label><input className="input" value={form.name} onChange={set("name")} /></div>
          <div className="field"><label>Serial number</label><input className="input" value={form.serial_number} onChange={set("serial_number")} /></div>
        </div>
        <div className="row2">
          <div className="field"><label>Tag number</label><input className="input" value={form.tag_number} onChange={set("tag_number")} /></div>
          <div className="field"><label>Certificate number</label><input className="input" value={form.certificate_number} onChange={set("certificate_number")} /></div>
        </div>
        <div className="row2">
          <div className="field"><label>Length (m)</label><input className="input" type="number" value={form.length} onChange={set("length")} placeholder={selected?.default_length?.toString() ?? ""} /></div>
          {placing ? (
            <div className="field">
              <label>Lifecycle</label>
              <input className="input" value="Active (placed on drum)" disabled />
            </div>
          ) : (
            <div className="field">
              <label>Lifecycle</label>
              <select className="input" value={form.lifecycle_status} onChange={set("lifecycle_status")}>
                <option value="active">Active</option>
                <option value="ordered">Ordered (not yet aboard)</option>
                <option value="spare">Spare</option>
              </select>
            </div>
          )}
        </div>
        <div className="row2">
          <div className="field"><label>Manufacture date</label><input className="input" type="date" value={form.manufacture_date} onChange={set("manufacture_date")} /></div>
          <div className="field"><label>Installation date</label><input className="input" type="date" value={form.installation_date} onChange={set("installation_date")} /></div>
        </div>

        <div className="field">
          <label>Attachments</label>
          <span className="muted" style={{ fontSize: 12 }}>
            Delivery document or photos — captured now or added later on the rope record.
          </span>
          <div className="attach-actions">
            <button type="button" className="btn ghost" onClick={() => docInput.current?.click()}>+ Delivery document</button>
            <button type="button" className="btn ghost" onClick={() => photoInput.current?.click()}>+ Photo</button>
          </div>
          <input ref={docInput} type="file" accept="application/pdf,image/*" multiple hidden onChange={addFiles("delivery")} />
          <input ref={photoInput} type="file" accept="image/*" capture="environment" multiple hidden onChange={addFiles("photo")} />
          {attachments.length > 0 && (
            <ul className="attach-list">
              {attachments.map((a, i) => (
                <li key={i}>
                  <span className="attach-kind">{a.kind === "photo" ? "Photo" : "Document"}</span>
                  <span className="attach-name">{a.file.name}</span>
                  <button type="button" className="attach-remove" onClick={() => removeAttachment(i)} aria-label="Remove">×</button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {register.isError && <div className="err">Could not register line (serial may already exist).</div>}
        {placeErr && <div className="err">{placeErr} The line was saved as a spare — assign it from the register or deck.</div>}
        {uploadErr && <div className="err">{uploadErr} The line was saved — add the missing files from its record.</div>}

        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>{placeErr || uploadErr ? "Close" : "Cancel"}</button>
          <button className="btn" disabled={!valid || register.isPending || move.isPending || uploading} onClick={submit}>
            {uploading ? "Uploading…" : register.isPending || move.isPending ? "Saving…" : placing ? "Register & place" : "Register line"}
          </button>
        </div>
      </div>
    </div>
  );
}
