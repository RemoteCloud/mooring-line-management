// Placeholder for feature pages whose backend slice is not built yet.
export function Stub({ title, sub, needs }: { title: string; sub: string; needs: string }) {
  return (
    <>
      <h1 className="page-title">{title}</h1>
      <p className="page-sub">{sub}</p>
      <div className="stub">
        <h3>Coming with its backend slice</h3>
        Wires up to <code>{needs}</code>.
      </div>
    </>
  );
}
