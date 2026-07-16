import type { ReminderStatus } from "../lib/api";

const styles: Record<ReminderStatus, string> = {
  ok: "bg-ok text-white",
  soon: "bg-soon text-black",
  due: "bg-due text-white",
  overdue: "bg-overdue text-white",
};

export default function StatusBadge({
  status,
  text,
}: {
  status?: ReminderStatus;
  text?: string;
}) {
  if (!status) return null;
  return (
    <span
      className={`shrink-0 rounded-full px-2 py-0.5 text-[0.68rem] font-bold uppercase tracking-wide ${styles[status]}`}
    >
      {text ?? status}
    </span>
  );
}
