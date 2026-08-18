import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import { Activity, ClipboardList } from "lucide-react";
import * as React from "react";
import AuditLogHealthWidget from "../AuditLogViewer/AuditLogWidget";
import AuditLogViewer from "../AuditLogViewer";
import { useContextResource } from "./utils";

export const ResourceAuditLogs = (props: { resource: Resource }) => {
  const { resource } = props;
  const [periodMinutes, setPeriodMinutes] = React.useState(360);
  const displayName =
    resource.metadata?.displayName || resource.metadata?.name || "Resource";

  return (
    <div className="flex w-full flex-col gap-4">
      <header className="flex flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_2px_10px_rgba(15,23,42,0.04)] sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
            <ClipboardList size={18} strokeWidth={2.2} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h1 className="truncate text-base font-bold text-slate-900">
                Audit logs
              </h1>
              <Activity size={14} className="shrink-0 text-slate-400" />
            </div>
            <p className="mt-0.5 truncate text-[0.72rem] font-semibold text-slate-500">
              Review administrative changes and resource operations.
            </p>
            <span className="mt-2 inline-flex max-w-full truncate rounded-md bg-slate-100 px-2 py-1 text-[0.65rem] font-bold text-slate-600">
              {resource.kind} · {displayName}
            </span>
          </div>
        </div>
        <span className="inline-flex w-fit shrink-0 items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-[0.64rem] font-bold uppercase tracking-[0.06em] text-slate-500">
          Resource scope
        </span>
      </header>

      <AuditLogHealthWidget
        resourceRef={getResourceRef(resource)}
        periodMinutes={periodMinutes}
        onPeriodChange={setPeriodMinutes}
      />

      <section className="flex w-full flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_2px_10px_rgba(15,23,42,0.03)]">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-sm font-bold text-slate-800">Event stream</h2>
            <p className="mt-1 text-[0.72rem] font-medium text-slate-500">
              Individual operations matching the selected time range.
            </p>
          </div>
          <span className="text-[0.65rem] font-bold text-slate-400">
            Updates automatically
          </span>
        </div>

        <AuditLogViewer
          resourceRef={getResourceRef(resource)}
          periodMinutes={periodMinutes}
        />
      </section>
    </div>
  );
};

const ResourceItemAuditLogsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceAuditLogs resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemAuditLogsPage;
