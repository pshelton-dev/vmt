import { Link } from "react-router-dom";
import type { Vehicle } from "../lib/api";

/**
 * Photo collage of the active vehicles for the top of the dashboard. Each tile
 * links to its vehicle. Tiles are laid out in justified rows: alternating wide
 * and narrow bases give the collage look, and flex-grow stretches each row to
 * the full width, so there are no gaps whatever the number of vehicles.
 */
export default function VehicleCollage({ vehicles }: { vehicles: Vehicle[] }) {
  if (vehicles.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1.5 overflow-hidden rounded-xl">
      {vehicles.map((v, i) => (
        <Link
          key={v.id}
          to={`/vehicles/${v.id}`}
          title={v.name}
          className={`group relative h-28 grow overflow-hidden bg-surface-2 sm:h-40 ${
            i % 3 === 0 ? "basis-full sm:basis-[40%]" : "basis-[40%] sm:basis-[25%]"
          }`}
        >
          {v.photo_id ? (
            <img
              src={`/api/v1/files/${v.photo_id}`}
              alt={v.name}
              loading="lazy"
              className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
            />
          ) : (
            <div className="flex h-full w-full items-center justify-center text-4xl">🚗</div>
          )}
          <span className="absolute inset-x-0 bottom-0 truncate bg-gradient-to-t from-black/70 to-transparent px-2 pb-1.5 pt-6 text-sm font-semibold text-white">
            {v.name}
          </span>
        </Link>
      ))}
    </div>
  );
}
