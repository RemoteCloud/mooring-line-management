import { useState } from "react";
import {
  useMakers,
  useLineTypes,
  useProducts,
  useCreateMaker,
  useCreateLineType,
  useCreateProduct,
  type CreateProductBody,
} from "./api";
import "./catalogue.css";

export function CataloguePage() {
  return (
    <>
      <h1 className="page-title">Catalogue</h1>
      <p className="page-sub">Makers, line types and products — master data.</p>

      <div className="catalogue-sections">
        <MakersSection />
        <LineTypesSection />
        <ProductsSection />
      </div>
    </>
  );
}

/* ------------------------------------------------------------------ Makers */

function MakersSection() {
  const { data: makers, isLoading, isError } = useMakers();
  const createMaker = useCreateMaker();

  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");

  const valid = name.trim().length > 0;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    await createMaker.mutateAsync({
      name: name.trim(),
      notes: notes.trim() || undefined,
    });
    setName("");
    setNotes("");
  };

  return (
    <section className="catalogue-section">
      <div className="toolbar">
        <h2>Makers</h2>
      </div>

      <div className="card">
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr>
                <th>Name</th>
                <th>Notes</th>
              </tr>
            </thead>
            <tbody>
              {makers?.map((m) => (
                <tr key={m.id}>
                  <td>{m.name}</td>
                  <td className="muted">{m.notes || "—"}</td>
                </tr>
              ))}
              {isLoading && (
                <tr>
                  <td colSpan={2} className="muted catalogue-empty">
                    Loading…
                  </td>
                </tr>
              )}
              {isError && (
                <tr>
                  <td colSpan={2} className="err catalogue-empty">
                    Could not load makers.
                  </td>
                </tr>
              )}
              {!isLoading && !isError && makers?.length === 0 && (
                <tr>
                  <td colSpan={2} className="muted catalogue-empty">
                    No makers yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <form className="catalogue-inline-form" onSubmit={submit}>
          <div className="field">
            <label htmlFor="maker-name">Name</label>
            <input
              id="maker-name"
              className="input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Samson"
            />
          </div>
          <div className="field" style={{ flex: 1, minWidth: 200 }}>
            <label htmlFor="maker-notes">Notes</label>
            <input
              id="maker-notes"
              className="input"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="optional"
            />
          </div>
          <button
            type="submit"
            className="btn"
            disabled={!valid || createMaker.isPending}
          >
            {createMaker.isPending ? "Adding…" : "Add maker"}
          </button>
        </form>
        {createMaker.isError && (
          <div className="err">Could not add maker.</div>
        )}
      </div>
    </section>
  );
}

/* -------------------------------------------------------------- Line types */

function LineTypesSection() {
  const { data: lineTypes, isLoading, isError } = useLineTypes();
  const createLineType = useCreateLineType();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const valid = name.trim().length > 0;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    await createLineType.mutateAsync({
      name: name.trim(),
      description: description.trim() || undefined,
    });
    setName("");
    setDescription("");
  };

  return (
    <section className="catalogue-section">
      <div className="toolbar">
        <h2>Line types</h2>
      </div>

      <div className="card">
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr>
                <th>Name</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {lineTypes?.map((t) => (
                <tr key={t.id}>
                  <td>{t.name}</td>
                  <td className="muted">{t.description || "—"}</td>
                </tr>
              ))}
              {isLoading && (
                <tr>
                  <td colSpan={2} className="muted catalogue-empty">
                    Loading…
                  </td>
                </tr>
              )}
              {isError && (
                <tr>
                  <td colSpan={2} className="err catalogue-empty">
                    Could not load line types.
                  </td>
                </tr>
              )}
              {!isLoading && !isError && lineTypes?.length === 0 && (
                <tr>
                  <td colSpan={2} className="muted catalogue-empty">
                    No line types.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <form className="catalogue-inline-form" onSubmit={submit}>
          <div className="field">
            <label htmlFor="line-type-name">Name</label>
            <input
              id="line-type-name"
              className="input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Mooring tail"
            />
          </div>
          <div className="field" style={{ flex: 1, minWidth: 200 }}>
            <label htmlFor="line-type-desc">Description</label>
            <input
              id="line-type-desc"
              className="input"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="optional"
            />
          </div>
          <button
            type="submit"
            className="btn"
            disabled={!valid || createLineType.isPending}
          >
            {createLineType.isPending ? "Adding…" : "Add line type"}
          </button>
        </form>
        {createLineType.isError && (
          <div className="err">Could not add line type.</div>
        )}
      </div>
    </section>
  );
}

/* ---------------------------------------------------------------- Products */

function ProductsSection() {
  const { data: products, isLoading, isError } = useProducts();
  const [addOpen, setAddOpen] = useState(false);

  return (
    <section className="catalogue-section">
      <div className="toolbar">
        <h2>Products</h2>
        <div className="grow" />
        <button className="btn" onClick={() => setAddOpen(true)}>
          + Add product
        </button>
      </div>

      <div className="card">
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr>
                <th>Product</th>
                <th>Maker</th>
                <th>Type</th>
                <th>Construction</th>
                <th>Default length</th>
                <th>SWL</th>
                <th>Break load</th>
                <th>Turnable</th>
              </tr>
            </thead>
            <tbody>
              {products?.map((p) => (
                <tr key={p.id}>
                  <td>{p.productName}</td>
                  <td className="muted">{p.makerName}</td>
                  <td>{p.lineTypeName}</td>
                  <td>{p.constructionType || "—"}</td>
                  <td>
                    {p.defaultLength != null ? `${p.defaultLength} m` : "—"}
                  </td>
                  <td>{p.swl != null ? `${p.swl} t` : "—"}</td>
                  <td>{p.breakLoad != null ? `${p.breakLoad} t` : "—"}</td>
                  <td>{p.canBeTurned ? "Yes" : "No"}</td>
                </tr>
              ))}
              {isLoading && (
                <tr>
                  <td colSpan={8} className="muted catalogue-empty">
                    Loading…
                  </td>
                </tr>
              )}
              {isError && (
                <tr>
                  <td colSpan={8} className="err catalogue-empty">
                    Could not load products.
                  </td>
                </tr>
              )}
              {!isLoading && !isError && products?.length === 0 && (
                <tr>
                  <td colSpan={8} className="muted catalogue-empty">
                    No products yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {addOpen && <AddProductDialog onClose={() => setAddOpen(false)} />}
    </section>
  );
}

/* ----------------------------------------------------- Add product dialog */

function AddProductDialog({ onClose }: { onClose: () => void }) {
  const { data: makers = [] } = useMakers();
  const { data: lineTypes = [] } = useLineTypes();
  const createProduct = useCreateProduct();

  const [form, setForm] = useState({
    makerId: "",
    lineTypeId: "",
    productName: "",
    constructionType: "",
    defaultLength: "",
    swl: "",
    breakLoad: "",
    canBeTurned: false,
    manufacturerManualRef: "",
    notes: "",
  });

  const setField =
    (k: keyof typeof form) =>
    (
      e: React.ChangeEvent<
        HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
      >,
    ) => {
      const value =
        e.target instanceof HTMLInputElement && e.target.type === "checkbox"
          ? e.target.checked
          : e.target.value;
      setForm((prev) => ({ ...prev, [k]: value }));
    };

  const valid =
    form.makerId !== "" &&
    form.lineTypeId !== "" &&
    form.productName.trim() !== "";

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    const body: CreateProductBody = {
      makerId: form.makerId,
      lineTypeId: form.lineTypeId,
      productName: form.productName.trim(),
      constructionType: form.constructionType.trim() || undefined,
      defaultLength: form.defaultLength
        ? Number(form.defaultLength)
        : undefined,
      swl: form.swl ? Number(form.swl) : undefined,
      breakLoad: form.breakLoad ? Number(form.breakLoad) : undefined,
      canBeTurned: form.canBeTurned,
      manufacturerManualRef: form.manufacturerManualRef.trim() || undefined,
      notes: form.notes.trim() || undefined,
    };
    await createProduct.mutateAsync(body);
    onClose();
  };

  return (
    <div className="overlay" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>Add product</h3>

        <div className="row2">
          <div className="field">
            <label htmlFor="prod-maker">Maker</label>
            <select
              id="prod-maker"
              className="input"
              value={form.makerId}
              onChange={setField("makerId")}
            >
              <option value="">Select maker…</option>
              {makers.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
          </div>
          <div className="field">
            <label htmlFor="prod-type">Line type</label>
            <select
              id="prod-type"
              className="input"
              value={form.lineTypeId}
              onChange={setField("lineTypeId")}
            >
              <option value="">Select type…</option>
              {lineTypes.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="field">
          <label htmlFor="prod-name">Product name</label>
          <input
            id="prod-name"
            className="input"
            value={form.productName}
            onChange={setField("productName")}
          />
        </div>

        <div className="row2">
          <div className="field">
            <label htmlFor="prod-construction">Construction type</label>
            <input
              id="prod-construction"
              className="input"
              value={form.constructionType}
              onChange={setField("constructionType")}
              placeholder="optional"
            />
          </div>
          <div className="field">
            <label htmlFor="prod-length">Default length (m)</label>
            <input
              id="prod-length"
              className="input"
              type="number"
              value={form.defaultLength}
              onChange={setField("defaultLength")}
              placeholder="optional"
            />
          </div>
        </div>

        <div className="row2">
          <div className="field">
            <label htmlFor="prod-swl">SWL (t)</label>
            <input
              id="prod-swl"
              className="input"
              type="number"
              min="0"
              step="0.1"
              value={form.swl}
              onChange={setField("swl")}
              placeholder="optional"
            />
          </div>
          <div className="field">
            <label htmlFor="prod-mbl">Break load (t)</label>
            <input
              id="prod-mbl"
              className="input"
              type="number"
              min="0"
              step="0.1"
              value={form.breakLoad}
              onChange={setField("breakLoad")}
              placeholder="optional"
            />
          </div>
        </div>

        <div className="field">
          <label htmlFor="prod-manual">Manufacturer manual ref</label>
          <input
            id="prod-manual"
            className="input"
            value={form.manufacturerManualRef}
            onChange={setField("manufacturerManualRef")}
            placeholder="optional"
          />
        </div>

        <div className="field">
          <label htmlFor="prod-notes">Notes</label>
          <textarea
            id="prod-notes"
            className="input"
            rows={2}
            value={form.notes}
            onChange={setField("notes")}
            placeholder="optional"
          />
        </div>

        <label className="catalogue-check">
          <input
            type="checkbox"
            checked={form.canBeTurned}
            onChange={setField("canBeTurned")}
          />
          Can be turned (end-for-end)
        </label>

        {createProduct.isError && (
          <div className="err">Could not create product.</div>
        )}

        <div className="dialog-actions">
          <button type="button" className="btn ghost" onClick={onClose}>
            Cancel
          </button>
          <button
            type="button"
            className="btn"
            disabled={!valid || createProduct.isPending}
            onClick={submit}
          >
            {createProduct.isPending ? "Saving…" : "Add product"}
          </button>
        </div>
      </div>
    </div>
  );
}
