import { IntegrationBinding } from "@/apis/accessv1/accessv1";
import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlertTriangle,
  Building2,
  CheckCircle2,
  ChevronRight,
  Link2,
  Plug,
  PowerOff,
  RefreshCw,
  Send,
  Timer,
  UserCheck,
} from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import TimeAgo from "@/components/TimeAgo";
import { getPurposeLabel } from "../IntegrationBinding/utils";
import {
  useAccessCreated,
  useAccessTotals,
  useFailingBindings,
} from "./queries";

const SHOWN = 6;

const FailingRow = (props: { item: IntegrationBinding; returnTo: string }) => {
  const { item } = props;
  const md = item.metadata!;
  const status = item.status;

  return (
    <Link
      to={`/access/integrationbindings/${md.name}`}
      state={{ returnTo: props.returnTo }}
      preventScrollReset
      className="group flex min-w-0 items-center gap-3 rounded-lg border border-red-200 bg-red-50/40 px-3 py-2.5 outline-none transition-[border-color,box-shadow] duration-150 hover:border-red-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-body font-semibold text-slate-800">
          {status?.integrationRef?.name ?? md.name}
          <span className="ml-1.5 text-micro font-normal text-slate-500">
            {getPurposeLabel(status?.purpose)}
          </span>
        </span>
        <span className="truncate text-micro font-normal text-slate-500">
          {status?.requestRef?.name && <>{status.requestRef.name} · </>}
          {status?.lastError || "The last delivery failed"}
        </span>
      </span>

      <span className="hidden shrink-0 rounded-full border border-red-200 px-1.5 py-px text-micro font-semibold text-red-700 sm:inline">
        {status?.attempts ?? 0} failed
      </span>

      {status?.nextAttemptAt && (
        <span className="hidden shrink-0 items-center gap-1 text-micro font-normal text-slate-500 md:inline-flex">
          <Timer size={11} strokeWidth={2.3} />
          <TimeAgo rfc3339={status.nextAttemptAt} />
        </span>
      )}

      <ChevronRight
        size={13}
        strokeWidth={2.4}
        aria-hidden="true"
        className="shrink-0 text-slate-400 transition-colors duration-150 group-hover:text-slate-800"
      />
    </Link>
  );
};

const Integrations = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.high);
  const failing = useFailingBindings(QUERY_PRIORITY.low);

  const failingItems = (failing.data?.items ?? []).slice(0, SHOWN);
  const totalFailing = n(totals.data?.access?.integrationBinding?.totalFailing);

  const integrations = totals.data?.access?.integration;
  const identities = totals.data?.access?.integrationIdentity;
  const bindings = totals.data?.access?.integrationBinding;
  const freshBindings = created.data?.access?.integrationBinding;

  return (
    <Panel
      icon={Plug}
      title="Integrations & presentations"
      description="The external providers that present the Requests and the state of what the Cluster has delivered to them"
      to="/access/integrations"
      toLabel="Integrations"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Integrations"
            value={n(integrations?.totalNumber)}
            icon={Plug}
            to="/access/integrations"
          />
          <MiniStat
            label="Ready"
            value={n(integrations?.totalReady)}
            tone="positive"
            icon={CheckCircle2}
            to="/access/integrations?state=READY"
          />
          <MiniStat
            label="Failing"
            value={n(integrations?.totalError)}
            tone={n(integrations?.totalError) > 0 ? "critical" : "default"}
            icon={AlertTriangle}
            to="/access/integrations?state=ERROR"
          />
          <MiniStat
            label="Disabled"
            value={n(integrations?.totalDisabled)}
            tone={n(integrations?.totalDisabled) > 0 ? "warning" : "default"}
            icon={PowerOff}
            to="/access/integrations?isDisabled=true"
          />
          <MiniStat
            label="Provider tenants"
            value={n(integrations?.totalExternalTenant)}
            icon={Building2}
          />
          <MiniStat
            label="Identities"
            value={n(identities?.totalNumber)}
            icon={Link2}
            to="/access/integrationidentities"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <Breakdown
            title="Providers"
            note={`${compact(n(integrations?.totalNumber))} Integrations`}
          >
            <CompositionBar
              segments={[
                { label: "Slack", value: n(integrations?.totalSlack) },
                { label: "Jira", value: n(integrations?.totalJira) },
                { label: "Webhook", value: n(integrations?.totalWebhook) },
              ]}
              total={n(integrations?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Interactive review
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(integrations?.totalInteractiveReview))}
                </span>
              </span>
              <span>
                Direct user delivery
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(integrations?.totalDirectUserDelivery))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title="Presentation state"
            note={`${compact(n(bindings?.totalNumber))} presentations`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Ready",
                  value: n(bindings?.totalReady),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Pending",
                  value: n(bindings?.totalPending),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Degraded",
                  value: n(bindings?.totalDegraded),
                  color: STATUS_COLORS.critical,
                },
                { label: "Closed", value: n(bindings?.totalClosed) },
              ]}
              total={n(bindings?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Out of date
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(bindings?.totalOutOfDate))}
                </span>
              </span>
              <span>
                Failing
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(bindings?.totalFailing))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title="Audience"
            note={`${compact(n(bindings?.totalInteractive))} interactive`}
          >
            <CompositionBar
              segments={[
                { label: "Shared", value: n(bindings?.totalShared) },
                { label: "Reviewers", value: n(bindings?.totalReviewers) },
                { label: "Requester", value: n(bindings?.totalRequester) },
                { label: "Subject", value: n(bindings?.totalSubject) },
              ]}
              total={n(bindings?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Review surfaces
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(bindings?.totalReviewSurface))}
                </span>
              </span>
              <span>
                Notifications
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(bindings?.totalNotification))}
                </span>
              </span>
            </div>
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Presentations · ${rangeLabel}`}
            value={n(freshBindings?.totalNumber)}
            prev={n(freshBindings?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={Send}
            to="/access/integrationbindings"
          />
          <DeltaStat
            label="Delivered"
            value={n(freshBindings?.totalReady)}
            prev={n(freshBindings?.previous?.totalReady)}
            rangeLabel={rangeLabel}
            icon={CheckCircle2}
            to="/access/integrationbindings?state=READY"
          />
          <DeltaStat
            label="Out of date"
            value={n(freshBindings?.totalOutOfDate)}
            prev={n(freshBindings?.previous?.totalOutOfDate)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={RefreshCw}
            to="/access/integrationbindings?isOutOfDate=true"
          />
          <DeltaStat
            label="Failing"
            value={n(freshBindings?.totalFailing)}
            prev={n(freshBindings?.previous?.totalFailing)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={AlertTriangle}
            to="/access/integrationbindings?isFailing=true"
          />
          <DeltaStat
            label="Interactive"
            value={n(freshBindings?.totalInteractive)}
            prev={n(freshBindings?.previous?.totalInteractive)}
            rangeLabel={rangeLabel}
            icon={UserCheck}
            to="/access/integrationbindings?interactionMode=INTERACTIVE"
          />
        </DeltaStatGrid>

        {failingItems.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Failing presentations
            </span>
            {failingItems.map((item) => (
              <FailingRow
                key={item.metadata?.uid}
                item={item}
                returnTo={returnTo}
              />
            ))}
            {totalFailing > failingItems.length && (
              <Link
                to="/access/integrationbindings?isFailing=true"
                className="mt-1 flex items-center justify-center gap-1.5 rounded-lg border border-dashed border-slate-200 px-3 py-2 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50/70 hover:text-slate-800 focus-visible:ring-2 focus-visible:ring-slate-400"
              >
                {compact(totalFailing - failingItems.length)} more failing
              </Link>
            )}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Integrations;
