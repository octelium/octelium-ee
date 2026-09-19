import { Certificate } from "@/apis/enterprisev1/enterprisev1";
import {
  Breakdown,
  EmptyHint,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import TimeAgo from "@/components/TimeAgo";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Boxes,
  CalendarX2,
  ChevronRight,
  Crown,
  Server,
  ShieldCheck,
  TriangleAlert,
} from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { useEnterpriseTotals, useExpiringCertificates } from "./queries";

const SHOWN = 8;

const expiresAt = (item: Certificate) =>
  item.status?.info?.notAfter?.seconds ?? 0;

const Certificates = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);
  const expiring = useExpiringCertificates(QUERY_PRIORITY.normal);

  const data = totals.data?.enterprise?.certificate;
  const issuers = totals.data?.enterprise?.certificateIssuer;

  const items = [...(expiring.data?.items ?? [])]
    .sort((a, b) => expiresAt(a) - expiresAt(b))
    .slice(0, SHOWN);

  return (
    <Panel
      icon={ShieldCheck}
      title="Certificates"
      description={`Issuance and expiry of the Cluster's TLS Certificates over the last ${rangeLabel}`}
      to="/enterprise/certificates"
      toLabel="All Certificates"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Certificates"
            value={n(data?.totalNumber)}
            icon={ShieldCheck}
            to="/enterprise/certificates"
          />
          <MiniStat
            label="Expiring soon"
            value={n(data?.totalExpiringSoon)}
            tone={n(data?.totalExpiringSoon) > 0 ? "warning" : "default"}
            icon={CalendarX2}
            to="/enterprise/certificates?isExpiringSoon=true"
          />
          <MiniStat
            label="Expired"
            value={n(data?.totalExpired)}
            tone={n(data?.totalExpired) > 0 ? "critical" : "default"}
            icon={CalendarX2}
            to="/enterprise/certificates?isExpired=true"
          />
          <MiniStat
            label="Issuance failed"
            value={n(data?.totalIssuanceFailed)}
            tone={n(data?.totalIssuanceFailed) > 0 ? "critical" : "default"}
            icon={TriangleAlert}
            to="/enterprise/certificates?issuanceState=FAILED"
          />
          <MiniStat
            label="Issuers ready"
            value={n(issuers?.totalReady)}
            tone="positive"
            icon={Crown}
            to="/enterprise/certificateissuers"
          />
          <MiniStat
            label="Issuers not ready"
            value={n(issuers?.totalNotReady)}
            tone={n(issuers?.totalNotReady) > 0 ? "critical" : "default"}
            icon={Crown}
            to="/enterprise/certificateissuers?state=NOT_READY"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <Breakdown
            title="Issuance state"
            note={`${compact(n(data?.totalManaged))} managed`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Success",
                  value: n(data?.totalIssuanceSuccess),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Issuing",
                  value: n(data?.totalIssuing),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Requested",
                  value: n(data?.totalIssuanceRequested),
                },
                {
                  label: "Failed",
                  value: n(data?.totalIssuanceFailed),
                  color: STATUS_COLORS.critical,
                },
              ]}
              total={n(data?.totalManaged)}
            />
          </Breakdown>

          <Breakdown
            title="How they are obtained"
            note={`${compact(n(data?.totalNumber))} total`}
          >
            <CompositionBar
              segments={[
                { label: "Managed", value: n(data?.totalManaged) },
                { label: "Manual", value: n(data?.totalManual) },
              ]}
              total={n(data?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Issuer readiness"
            note={`${compact(n(issuers?.totalACME))} ACME`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Ready",
                  value: n(issuers?.totalReady),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Preparing",
                  value: n(issuers?.totalPreparing),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Not ready",
                  value: n(issuers?.totalNotReady),
                  color: STATUS_COLORS.critical,
                },
              ]}
              total={n(issuers?.totalNumber)}
            />
          </Breakdown>
        </div>

        <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
          <MiniStat
            label="Services covered"
            value={n(data?.totalService)}
            icon={Server}
            to="/core/services"
          />
          <MiniStat
            label="Namespaces covered"
            value={n(data?.totalNamespace)}
            icon={Boxes}
            to="/core/namespaces"
          />
          <MiniStat
            label="Issuers in use"
            value={n(data?.totalCertificateIssuer)}
            icon={Crown}
            to="/enterprise/certificateissuers"
          />
        </div>

        {expiring.isLoading ? (
          <div className="flex flex-col gap-1.5">
            {[0, 1, 2].map((index) => (
              <div
                key={index}
                className="h-[46px] animate-pulse rounded-lg border border-slate-200 bg-slate-50"
              />
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyHint>No Certificate expires within the next 30 days.</EmptyHint>
        ) : (
          <div className="flex flex-col gap-2 rounded-lg border border-amber-200 bg-amber-50/40 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="inline-flex items-center gap-2 text-micro font-semibold uppercase tracking-[0.07em] text-amber-700">
                <CalendarX2 size={12} strokeWidth={2.3} />
                Renew these first
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(n(data?.totalExpiringSoon))} expiring
              </span>
            </div>

            {items.map((item) => {
              const md = item.metadata!;
              return (
                <Link
                  key={md.uid}
                  to={`/enterprise/certificates/${md.name}`}
                  state={{ returnTo }}
                  preventScrollReset
                  className="group flex min-w-0 items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2 outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
                >
                  <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span className="truncate text-body font-semibold text-slate-800">
                      {md.displayName || md.name}
                    </span>
                    <span className="truncate text-micro font-normal text-slate-500">
                      {item.status?.info?.commonName ||
                        item.status?.serviceRef?.name ||
                        item.status?.namespaceRef?.name ||
                        "No subject"}
                    </span>
                  </span>

                  <span className="shrink-0 text-micro font-semibold text-amber-700">
                    expires <TimeAgo rfc3339={item.status?.info?.notAfter} />
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
      </div>
    </Panel>
  );
};

export default Certificates;
