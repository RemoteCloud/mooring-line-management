import { useState } from "react";
import { useProducts, useRegisterLine } from "../../api/hooks";

export function AddLineDialog({ vesselId, onClose }: { vesselId: string; onClose: () => void }) {
  const { data: products = [] } = useProducts();
  const register = useRegisterLine(vesselId);

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
    await register.mutateAsync({
      product_id: form.product_id,
      name: form.name,
      serial_number: form.serial_number,
      tag_number: form.tag_number || undefined,
      certificate_number: form.certificate_number || undefined,
      length: form.length ? Number(form.length) : undefined,
      manufacture_date: form.manufacture_date || undefined,
      installation_date: form.installation_date || undefined,
      lifecycle_status: form.lifecycle_status,
    } as never);
    onClose();
  };

  const valid = form.product_id && form.name && form.serial_number;

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>Add mooring line</h3>

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
          <div className="field">
            <label>Lifecycle</label>
            <select className="input" value={form.lifecycle_status} onChange={set("lifecycle_status")}>
              <option value="active">Active</option>
              <option value="ordered">Ordered (not yet aboard)</option>
              <option value="spare">Spare</option>
            </select>
          </div>
        </div>
        <div className="row2">
          <div className="field"><label>Manufacture date</label><input className="input" type="date" value={form.manufacture_date} onChange={set("manufacture_date")} /></div>
          <div className="field"><label>Installation date</label><input className="input" type="date" value={form.installation_date} onChange={set("installation_date")} /></div>
        </div>

        {register.isError && <div className="err">Could not register line (serial may already exist).</div>}

        <div className="dialog-actions">
          <button className="btn ghost" onClick={onClose}>Cancel</button>
          <button className="btn" disabled={!valid || register.isPending} onClick={submit}>
            {register.isPending ? "Saving…" : "Register line"}
          </button>
        </div>
      </div>
    </div>
  );
}
