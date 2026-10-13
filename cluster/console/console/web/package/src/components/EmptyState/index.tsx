import { Inbox } from "lucide-react";

const EmptyState = (props: { title: string; minHeight?: number }) => (
  <div
    className="flex w-full flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-slate-200 bg-slate-50/50 px-4 text-center"
    style={{ minHeight: props.minHeight ?? 160 }}
  >
    <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-500">
      <Inbox size={16} strokeWidth={2} aria-hidden="true" />
    </span>
    <p className="text-xs font-normal text-slate-500">{props.title}</p>
  </div>
);

export default EmptyState;
