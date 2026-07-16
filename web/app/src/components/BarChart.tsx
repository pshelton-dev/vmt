import { useState } from "react";

export interface Bar {
  label: string;
  value: number;
}

/**
 * Lightweight responsive SVG bar chart (no chart library — keeps the bundle
 * small). Tap/hover a bar to see its value.
 */
export default function BarChart({
  bars,
  format,
}: {
  bars: Bar[];
  format: (v: number) => string;
}) {
  const [active, setActive] = useState<number | null>(null);
  const max = Math.max(...bars.map((b) => b.value), 1);
  const W = 720;
  const H = 220;
  const pad = { top: 24, bottom: 26, left: 8, right: 8 };
  const innerW = W - pad.left - pad.right;
  const innerH = H - pad.top - pad.bottom;
  const step = innerW / bars.length;
  const barW = Math.min(step * 0.62, 56);

  return (
    <svg viewBox={`0 0 ${W} ${H}`} className="h-auto w-full" role="img">
      {bars.map((b, i) => {
        const h = (b.value / max) * innerH;
        const x = pad.left + i * step + (step - barW) / 2;
        const y = pad.top + innerH - h;
        const isActive = active === i;
        return (
          <g
            key={i}
            onMouseEnter={() => setActive(i)}
            onMouseLeave={() => setActive(null)}
            onClick={() => setActive(isActive ? null : i)}
          >
            {/* invisible full-height hit target for touch */}
            <rect x={pad.left + i * step} y={0} width={step} height={H} fill="transparent" />
            <rect
              x={x}
              y={y}
              width={barW}
              height={h}
              rx={4}
              fill="var(--primary)"
              opacity={active === null || isActive ? 1 : 0.45}
            />
            {(isActive || bars.length <= 8) && b.value > 0 && (
              <text
                x={x + barW / 2}
                y={y - 6}
                textAnchor="middle"
                fontSize="11"
                fill="var(--muted)"
              >
                {format(b.value)}
              </text>
            )}
            <text
              x={pad.left + i * step + step / 2}
              y={H - 8}
              textAnchor="middle"
              fontSize="11"
              fill="var(--muted)"
            >
              {b.label}
            </text>
          </g>
        );
      })}
      <line
        x1={pad.left}
        y1={pad.top + innerH}
        x2={W - pad.right}
        y2={pad.top + innerH}
        stroke="var(--border)"
      />
    </svg>
  );
}
