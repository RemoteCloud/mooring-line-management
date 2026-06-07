import { StatusDot } from "../../components/ui";
import { dateLabel } from "../../lib/format";
import { usePhotos, useDeletePhoto } from "./api";
import "./files.css";

export function PhotoTimeline({ lineId }: { lineId: string }) {
  const { data: photos, isLoading } = usePhotos(lineId);
  const del = useDeletePhoto(lineId);

  if (isLoading) return <p className="muted">Loading photos…</p>;
  if (!photos || photos.length === 0) {
    return <div className="stub"><h3>No photos yet</h3>Upload a condition photo above.</div>;
  }

  return (
    <div className="photo-grid">
      {photos.map((p) => (
        <figure key={p.id} className="photo-card">
          {p.url ? (
            <a href={p.url} target="_blank" rel="noreferrer">
              <img src={p.url} alt={`Condition photo ${dateLabel(p.taken_at)}`} loading="lazy" />
            </a>
          ) : (
            <div className="photo-missing muted">No image</div>
          )}
          <figcaption>
            <div className="photo-meta">
              <span>{dateLabel(p.taken_at)}</span>
              {p.side && <span className="muted">Side {p.side}</span>}
            </div>
            {p.condition_at_capture && (
              <div className="photo-cond">
                <StatusDot condition={p.condition_at_capture as never} /> {p.condition_at_capture}
              </div>
            )}
            <button
              className="btn photo-del"
              disabled={del.isPending}
              onClick={() => {
                if (window.confirm("Delete this photo?")) del.mutate(p.id);
              }}
            >
              Delete
            </button>
          </figcaption>
        </figure>
      ))}
    </div>
  );
}
