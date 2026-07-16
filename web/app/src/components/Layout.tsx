import { NavLink, Outlet } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/api";

const tabs = [
  { to: "/", label: "Dashboard", icon: "🏠", end: true },
  { to: "/vehicles", label: "Vehicles", icon: "🚗" },
  { to: "/reminders", label: "Reminders", icon: "🔔" },
  { to: "/reports", label: "Reports", icon: "📊" },
  { to: "/settings", label: "Settings", icon: "⚙️" },
];

/** App shell: top bar with inline nav on desktop, bottom tab bar on phones. */
export default function Layout() {
  const qc = useQueryClient();
  const logout = async () => {
    await api.post("/logout");
    qc.invalidateQueries({ queryKey: ["session"] });
  };

  return (
    <div className="min-h-dvh pb-[calc(3.75rem+env(safe-area-inset-bottom))] md:pb-0">
      <header className="sticky top-0 z-10 border-b border-border bg-surface">
        <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
          <span className="text-lg font-bold">🔧 VMT</span>
          <nav className="hidden flex-1 gap-5 md:flex">
            {tabs.map((t) => (
              <NavLink
                key={t.to}
                to={t.to}
                end={t.end}
                className={({ isActive }) =>
                  `border-b-2 py-1 text-sm ${
                    isActive
                      ? "border-primary text-text"
                      : "border-transparent text-muted hover:text-text"
                  }`
                }
              >
                {t.label}
              </NavLink>
            ))}
          </nav>
          <button
            onClick={logout}
            className="ml-auto text-sm text-muted hover:text-text"
          >
            Sign out
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-5">
        <Outlet />
      </main>

      {/* Bottom tab bar (phones) */}
      <nav className="fixed inset-x-0 bottom-0 z-10 border-t border-border bg-surface pb-[env(safe-area-inset-bottom)] md:hidden">
        <div className="flex">
          {tabs.map((t) => (
            <NavLink
              key={t.to}
              to={t.to}
              end={t.end}
              className={({ isActive }) =>
                `flex min-h-11 flex-1 flex-col items-center justify-center gap-0.5 py-1.5 text-[0.65rem] ${
                  isActive ? "text-primary" : "text-muted"
                }`
              }
            >
              <span className="text-lg leading-none">{t.icon}</span>
              {t.label}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  );
}
