import { useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "../lib/api";

/** Sign-in (or first-run setup when no password exists yet). */
export default function Login({ configured }: { configured: boolean }) {
  const qc = useQueryClient();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    if (!configured && password !== confirm) {
      setError("Passwords do not match.");
      return;
    }
    setBusy(true);
    try {
      await api.post(configured ? "/login" : "/setup", { password });
      qc.invalidateQueries({ queryKey: ["session"] });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <form
        onSubmit={submit}
        className="w-full max-w-sm rounded-xl border border-border bg-surface p-6 text-center"
      >
        <h1 className="text-2xl font-bold">🔧 VMT</h1>
        <p className="mb-5 mt-1 text-sm text-muted">
          {configured
            ? "Vehicle Maintenance Tracker"
            : "Welcome — set a password to get started"}
        </p>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Password"
          autoFocus
          className="mb-3 w-full rounded-lg border border-border bg-bg px-3 py-2.5 outline-none focus:border-primary"
        />
        {!configured && (
          <input
            type="password"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder="Confirm password"
            className="mb-3 w-full rounded-lg border border-border bg-bg px-3 py-2.5 outline-none focus:border-primary"
          />
        )}
        {error && <p className="mb-3 text-sm text-danger">{error}</p>}
        <button
          type="submit"
          disabled={busy || !password}
          className="w-full rounded-lg bg-primary py-2.5 font-semibold text-white hover:bg-primary-strong disabled:opacity-50"
        >
          {busy ? "…" : configured ? "Sign in" : "Set password"}
        </button>
      </form>
    </div>
  );
}
