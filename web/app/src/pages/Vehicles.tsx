import { Route, Routes } from "react-router-dom";
import VehicleList from "./vehicles/VehicleList";
import VehicleForm from "./vehicles/VehicleForm";
import VehicleDetail from "./vehicles/VehicleDetail";
import ServiceForm from "./vehicles/ServiceForm";
import ReminderForm from "./vehicles/ReminderForm";
import Archive from "./vehicles/Archive";

export default function Vehicles() {
  return (
    <Routes>
      <Route index element={<VehicleList />} />
      <Route path="archive" element={<Archive />} />
      <Route path="new" element={<VehicleForm />} />
      <Route path=":id" element={<VehicleDetail />} />
      <Route path=":id/edit" element={<VehicleForm />} />
      <Route path=":vid/services/new" element={<ServiceForm />} />
      <Route path=":vid/reminders/new" element={<ReminderForm />} />
    </Routes>
  );
}
