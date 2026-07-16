import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";

// Small form primitives so every form shares one look (and 44px touch targets).

export const inputCls =
  "mt-1 block w-full rounded-lg border border-border bg-bg px-3 py-2.5 text-text outline-none focus:border-primary";

export function Field({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <label className="block text-sm text-muted">
      {label}
      {children}
    </label>
  );
}

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={inputCls} />;
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select {...props} className={inputCls} />;
}

export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea {...props} className={inputCls} />;
}

export function PrimaryButton(props: InputHTMLAttributes<HTMLButtonElement> & { children: ReactNode }) {
  const { children, disabled, ...rest } = props;
  return (
    <button
      {...(rest as object)}
      type="submit"
      disabled={disabled}
      className="min-h-11 rounded-lg bg-primary px-5 font-semibold text-white hover:bg-primary-strong disabled:opacity-50"
    >
      {children}
    </button>
  );
}

export function SecondaryButton({
  children,
  onClick,
  type = "button",
}: {
  children: ReactNode;
  onClick?: () => void;
  type?: "button" | "submit";
}) {
  return (
    <button
      type={type}
      onClick={onClick}
      className="min-h-11 rounded-lg border border-border px-5 text-text hover:border-muted"
    >
      {children}
    </button>
  );
}

/** Section card used across all pages. */
export function Card({ title, actions, children }: { title?: ReactNode; actions?: ReactNode; children: ReactNode }) {
  return (
    <section className="rounded-xl border border-border bg-surface p-4">
      {(title || actions) && (
        <div className="mb-3 flex items-center justify-between gap-2">
          {title && <h2 className="font-semibold">{title}</h2>}
          {actions}
        </div>
      )}
      {children}
    </section>
  );
}
