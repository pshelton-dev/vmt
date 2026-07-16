import { useEffect } from "react";
import type { Attachment } from "../lib/api";

/**
 * Fullscreen photo viewer: horizontal scroll-snap gives native swipe between
 * photos on touch devices; arrow keys work on desktop. Tap outside or ✕ to
 * close.
 */
export default function Lightbox({
  photos,
  start,
  close,
}: {
  photos: Attachment[];
  start: number;
  close: () => void;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    };
  }, [close]);

  return (
    <div className="fixed inset-0 z-40 bg-black/90" role="dialog" aria-label="Photo viewer">
      <button
        onClick={close}
        aria-label="Close"
        className="absolute right-3 top-[calc(0.75rem+env(safe-area-inset-top))] z-50 flex h-11 w-11 items-center justify-center rounded-full bg-black/60 text-xl text-white"
      >
        ✕
      </button>
      <div
        className="flex h-full snap-x snap-mandatory overflow-x-auto"
        ref={(el) => {
          // jump to the tapped photo without animation on open
          if (el) el.scrollLeft = el.clientWidth * start;
        }}
      >
        {photos.map((p) => (
          <div
            key={p.id}
            className="flex h-full w-full shrink-0 snap-center items-center justify-center p-2"
            onClick={(e) => {
              if (e.target === e.currentTarget) close();
            }}
          >
            <img
              src={`/api/v1/files/${p.id}`}
              alt={p.original_name}
              className="max-h-full max-w-full object-contain"
            />
          </div>
        ))}
      </div>
    </div>
  );
}
