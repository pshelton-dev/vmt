// Theme preference: "auto" follows the device; "light"/"dark" force it by
// stamping data-theme on <html> (see theme.css).

export type ThemePref = "auto" | "light" | "dark";

const KEY = "vmt-theme";

export function getThemePref(): ThemePref {
  const v = localStorage.getItem(KEY);
  return v === "light" || v === "dark" ? v : "auto";
}

export function applyThemePref(pref: ThemePref) {
  if (pref === "auto") {
    localStorage.removeItem(KEY);
    delete document.documentElement.dataset.theme;
  } else {
    localStorage.setItem(KEY, pref);
    document.documentElement.dataset.theme = pref;
  }
}

/** Apply the stored preference at startup (before first paint). */
export function initTheme() {
  applyThemePref(getThemePref());
}
