import { useState } from "react";
import {
  useMakers,
  useLineTypes,
  useProducts,
  useCreateMaker,
  useCreateLineType,
  useCreateProduct,
  useUpdateMaker,
  useUpdateLineType,
  useUpdateProduct,
  type Maker,
  type LineType,
  type Product,
  type CreateProductBody,
} from "./api";
import "./catalogue.css";

type Tab = "makers" | "line-types" | "products";

const TABS: { id: Tab; label: string }[] = [
  { id: "makers", label: "Makers" },
  { id: "line-types", label: "Line types" },
  { id: "products", label: "Products" },
];

export function CataloguePage() {
  const [tab, setTab] = useState<Tab>("makers");

  return (
    <>
      <h1 className="page-title">Catalogue</h1>
      <p className="page-sub">Makers, line types and products — master data.</p>

      <div className="tabs" role="tablist" aria-label="Catalogue sections">
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={tab === t.id}
            className={`tab${tab === t.id ? " active" : ""}`}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "makers" && <MakersPanel />}
      {tab === "line-types" && <LineTypesPanel />}
      {tab === "products" && <ProductsPanel />}
    </>
  );
}

/* A row that opens its edit dialog on click or keyboard activation. */
function EditableRow({
  onEdit,
  children,
}: {
  onEdit: () => void;
  children: React.ReactNode;
}) {
  return (
    <tr
      tabIndex={0}
      onClick={onEdit}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onEdit();
        }
      }}
    >
      {children}
    </tr>
  );
}

/* ------------------------------------------------------------------ Makers */

function MakersPanel() {
  const { data: makers, isLoading, isError } = useMakers();
  // null = closed, "new" = add, Maker = edit that row.
  const [editing, setEditing] = useState<"new" | Maker | null>(null);

  return (
    <section className="catalogue-section">
      <div className="toolbar">
        <span className="count">{makers?.length ?? 0} makers</span>
        <div className="grow" />
        <button className="btn" onClick={() => setEditing("new")}>
          + Add maker
        </button>
      </div>

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
              <EditableRow key={m.id} onEdit={() => setEditing(m)}>
                <td>{m.name}</td>
                <td className="muted">{m.notes || "—"}</td>
              </EditableRow>
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
                  No makers yet. Add the first one.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {editing && (
        <MakerDialog
          maker={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </section>
  );
}

function MakerDialog({
  maker,
  onClose,
}: {
  maker: Maker | null;
  onClose: () => void;
}) {
  const create = useCreateMaker();
  const update = useUpdateMaker();
  const [name, setName] = useState(maker?.name ?? "");
  const [notes, setNotes] = useState(maker?.notes ?? "");

  const valid = name.trim().length > 0;
  const pending = create.isPending || update.isPending;
  const failed = create.isError || update.isError;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    const body = { name: name.trim(), notes: notes.trim() || undefined };
    if (maker) await update.mutateAsync({ id: maker.id, body });
    else await create.mutateAsync(body);
    onClose();
  };

  return (
    <DialogShell
      title={maker ? "Edit maker" : "Add maker"}
      onClose={onClose}
      onSubmit={submit}
      valid={valid}
      pending={pending}
      error={failed ? "Could not save maker." : null}
    >
      <div className="field">
        <label htmlFor="maker-name">Name</label>
        <input
          id="maker-name"
          className="input"
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Samson"
        />
      </div>
      <div className="field">
        <label htmlFor="maker-notes">Notes</label>
        <input
          id="maker-notes"
          className="input"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="optional"
        />
      </div>
    </DialogShell>
  );
}

/* -------------------------------------------------------------- Line types */

function LineTypesPanel() {
  const { data: lineTypes, isLoading, isError } = useLineTypes();
  const [editing, setEditing] = useState<"new" | LineType | null>(null);

  return (
    <section className="catalogue-section">
      <div className="toolbar">
        <span className="count">{lineTypes?.length ?? 0} line types</span>
        <div className="grow" />
        <button className="btn" onClick={() => setEditing("new")}>
          + Add line type
        </button>
      </div>

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
              <EditableRow key={t.id} onEdit={() => setEditing(t)}>
                <td>{t.name}</td>
                <td className="muted">{t.description || "—"}</td>
              </EditableRow>
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
                  No line types yet. Add the first one.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {editing && (
        <LineTypeDialog
          lineType={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </section>
  );
}

function LineTypeDialog({
  lineType,
  onClose,
}: {
  lineType: LineType | null;
  onClose: () => void;
}) {
  const create = useCreateLineType();
  const update = useUpdateLineType();
  const [name, setName] = useState(lineType?.name ?? "");
  const [description, setDescription] = useState(lineType?.description ?? "");

  const valid = name.trim().length > 0;
  const pending = create.isPending || update.isPending;
  const failed = create.isError || update.isError;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    const body = {
      name: name.trim(),
      description: description.trim() || undefined,
    };
    if (lineType) await update.mutateAsync({ id: lineType.id, body });
    else await create.mutateAsync(body);
    onClose();
  };

  return (
    <DialogShell
      title={lineType ? "Edit line type" : "Add line type"}
      onClose={onClose}
      onSubmit={submit}
      valid={valid}
      pending={pending}
      error={failed ? "Could not save line type." : null}
    >
      <div className="field">
        <label htmlFor="line-type-name">Name</label>
        <input
          id="line-type-name"
          className="input"
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Mooring tail"
        />
      </div>
      <div className="field">
        <label htmlFor="line-type-desc">Description</label>
        <input
          id="line-type-desc"
          className="input"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="optional"
        />
      </div>
    </DialogShell>
  );
}

/* ---------------------------------------------------------------- Products */

function ProductsPanel() {
  const { data: products, isLoading, isError } = useProducts();
  const [editing, setEditing] = useState<"new" | Product | null>(null);

  return (
    <section className="catalogue-section">
      <div className="toolbar">
        <span className="count">{products?.length ?? 0} products</span>
        <div className="grow" />
        <button className="btn" onClick={() => setEditing("new")}>
          + Add product
        </button>
      </div>

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
              <EditableRow key={p.id} onEdit={() => setEditing(p)}>
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
              </EditableRow>
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
                  No products yet. Add the first one.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {editing && (
        <ProductDialog
          product={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </section>
  );
}

function ProductDialog({
  product,
  onClose,
}: {
  product: Product | null;
  onClose: () => void;
}) {
  const { data: makers = [] } = useMakers();
  const { data: lineTypes = [] } = useLineTypes();
  const create = useCreateProduct();
  const update = useUpdateProduct();

  const num = (n?: number) => (n != null ? String(n) : "");
  const [form, setForm] = useState({
    makerId: product?.makerId ?? "",
    lineTypeId: product?.lineTypeId ?? "",
    productName: product?.productName ?? "",
    constructionType: product?.constructionType ?? "",
    defaultLength: num(product?.defaultLength),
    swl: num(product?.swl),
    breakLoad: num(product?.breakLoad),
    canBeTurned: product?.canBeTurned ?? false,
    manufacturerManualRef: product?.manufacturerManualRef ?? "",
    notes: product?.notes ?? "",
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
  const pending = create.isPending || update.isPending;
  const failed = create.isError || update.isError;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid) return;
    const body: CreateProductBody = {
      makerId: form.makerId,
      lineTypeId: form.lineTypeId,
      productName: form.productName.trim(),
      constructionType: form.constructionType.trim() || undefined,
      defaultLength: form.defaultLength ? Number(form.defaultLength) : undefined,
      swl: form.swl ? Number(form.swl) : undefined,
      breakLoad: form.breakLoad ? Number(form.breakLoad) : undefined,
      canBeTurned: form.canBeTurned,
      manufacturerManualRef: form.manufacturerManualRef.trim() || undefined,
      notes: form.notes.trim() || undefined,
    };
    if (product) await update.mutateAsync({ id: product.id, body });
    else await create.mutateAsync(body);
    onClose();
  };

  return (
    <DialogShell
      title={product ? "Edit product" : "Add product"}
      onClose={onClose}
      onSubmit={submit}
      valid={valid}
      pending={pending}
      error={failed ? "Could not save product." : null}
      wide
    >
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
    </DialogShell>
  );
}

/* ------------------------------------------------------------ Dialog shell */

function DialogShell({
  title,
  onClose,
  onSubmit,
  valid,
  pending,
  error,
  wide,
  children,
}: {
  title: string;
  onClose: () => void;
  onSubmit: (e: React.FormEvent) => void;
  valid: boolean;
  pending: boolean;
  error: string | null;
  wide?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="overlay" onClick={onClose}>
      <form
        className="dialog"
        onClick={(e) => e.stopPropagation()}
        onSubmit={onSubmit}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
        }}
        style={wide ? undefined : { width: 460 }}
      >
        <h3>{title}</h3>
        {children}
        {error && <div className="err">{error}</div>}
        <div className="dialog-actions">
          <button type="button" className="btn ghost" onClick={onClose}>
            Cancel
          </button>
          <button type="submit" className="btn" disabled={!valid || pending}>
            {pending ? "Saving…" : "Save"}
          </button>
        </div>
      </form>
    </div>
  );
}
