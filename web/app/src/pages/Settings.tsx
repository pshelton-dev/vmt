import { useState } from "react";
import { applyThemePref, getThemePref, type ThemePref } from "../lib/theme";

const themeOptions: { value: ThemePref; label: string }[] = [
  { value: "auto", label: "Auto (device)" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

// Phase 3 adds the rest: preferences, password, notifications, import/export,
// backup/restore. The theme toggle lives client-side and ships now.
export default function Settings() {
  const [theme, setTheme] = useState<ThemePref>(getThemePref());
  const pick = (v: ThemePref) => {
    applyThemePref(v);
    setTheme(v);
  };

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-bold">Settings</h1>

      <section className="rounded-xl border border-border bg-surface p-4">
        <h2 className="mb-3 font-semibold">Appearance</h2>
        <div className="flex gap-2">
          {themeOptions.map((o) => (
            <button
              key={o.value}
              onClick={() => pick(o.value)}
              className={`min-h-11 flex-1 rounded-lg border px-3 text-sm ${
                theme === o.value
                  ? "border-primary bg-primary/10 font-semibold text-primary"
                  : "border-border text-muted hover:text-text"
              }`}
            >
              {o.label}
            </button>
          ))}
        </div>
      </section>

      <p className="text-muted">
        More settings coming in Phase 3 — use the classic UI meanwhile.
      </p>
    </div>
  );
}
