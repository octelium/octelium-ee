import { Request } from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { EmptyHint, Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import TimeAgo from "@/components/TimeAgo";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { ChevronRight, Timer } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { useAccessTotals, useActiveGrants } from "./queries";

const SHOWN = 8;

const resourceOf = (item: Request): ObjectReference | undefined => {
  const resource = item.spec?.resource?.type;
  if (resource?.oneofKind === "serviceRef") return resource.serviceRef;
  if (resource?.oneofKind === "catalog") return resource.catalog.catalogRef;
  return undefined;
};

const subjectOf = (item: Request): ObjectReference | undefined => {
  const subject = item.spec?.subject?.type;
  return subject?.oneofKind === "userRef" ? subject.userRef : undefined;
};

const endsAt = (item: Request) => item.status?.accessEndsAt?.seconds ?? 0;

const Grants = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;

  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const grants = useActiveGrants(QUERY_PRIORITY.low);

  const withEnd = (grants.data?.items ?? []).filter(
    (item) => !!item.status?.accessEndsAt,
  );
  const items = [...withEnd]
    .sort((a, b) => endsAt(a) - endsAt(b))
    .slice(0, SHOWN);

  const totalActive = n(totals.data?.access?.request?.totalActive);

  return (
    <Panel
      icon={Timer}
      title="Active grants"
      description={
        totalActive > 0
          ? `${compact(totalActive)} Request${totalActive === 1 ? "" : "s"} currently grant access · the ones ending soonest come first`
          : "No access Request currently grants access"
      }
      to="/access/requests?isActive=true"
      toLabel="All grants"
    >
      {grants.isLoading ? (
        <div className="flex flex-col gap-1.5">
          {[0, 1, 2, 3].map((index) => (
            <div
              key={index}
              className="h-[46px] animate-pulse rounded-lg border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : items.length === 0 ? (
        <EmptyHint>
          No grant has an end date that the Cluster can show right now.
        </EmptyHint>
      ) : (
        <div className="flex flex-col gap-1.5">
          {items.map((item) => {
            const md = item.metadata!;
            const resource = resourceOf(item);
            const subject = subjectOf(item);

            return (
              <Link
                key={md.uid}
                to={`/access/requests/${md.name}`}
                state={{ returnTo }}
                preventScrollReset
                className="group flex min-w-0 items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2 outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="truncate text-body font-semibold text-slate-800">
                    {md.displayName || md.name}
                  </span>
                  <span className="truncate text-micro font-normal text-slate-500">
                    {subject?.name ?? item.status?.userRef?.name ?? "Unknown"}
                    {resource?.name && <> · {resource.name}</>}
                  </span>
                </span>

                <span className="shrink-0 text-micro font-semibold text-slate-600">
                  ends <TimeAgo rfc3339={item.status?.accessEndsAt} />
                </span>

                <ChevronRight
                  size={13}
                  strokeWidth={2.4}
                  aria-hidden="true"
                  className="shrink-0 text-slate-400 transition-colors duration-150 group-hover:text-slate-800"
                />
              </Link>
            );
          })}
        </div>
      )}
    </Panel>
  );
};

export default Grants;
