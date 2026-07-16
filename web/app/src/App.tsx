import { Navigate, Route, Routes } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { api, type SessionInfo } from "./lib/api";
import Layout from "./components/Layout";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Vehicles from "./pages/Vehicles";
import Reminders from "./pages/Reminders";
import Reports from "./pages/Reports";
import Settings from "./pages/Settings";
import ServiceForm from "./pages/vehicles/ServiceForm";
import ReminderForm from "./pages/vehicles/ReminderForm";

export default function App() {
  const session = useQuery({
    queryKey: ["session"],
    queryFn: () => api.get<SessionInfo>("/session"),
  });

  if (session.isLoading) {
    return (
      <div className="flex min-h-dvh items-center justify-center text-muted">
        Loading…
      </div>
    );
  }
  if (session.isError) {
    return (
      <div className="flex min-h-dvh items-center justify-center text-danger">
        Cannot reach the VMT server.
      </div>
    );
  }

  if (!session.data!.authed) {
    return <Login configured={session.data!.configured} />;
  }

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="vehicles/*" element={<Vehicles />} />
        <Route path="services/:sid/edit" element={<ServiceForm />} />
        <Route path="reminders" element={<Reminders />} />
        <Route path="reminders/:rid/edit" element={<ReminderForm />} />
        <Route path="reports" element={<Reports />} />
        <Route path="settings" element={<Settings />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
