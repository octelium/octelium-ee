import { Integration, IntegrationBinding } from "@/apis/accessv1/accessv1";
import { Breakdown, EmptyHint, Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { AlertTriangle, ChevronRight, Plug, Plus } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import {
  getProviderKind,
  PROVIDER_LABEL,
  ProviderTile,
  ToneBadge,
} from "../common";
import {
  getDeliveryPurposeLabel,
  getIntegrationPresentation,
  getSharedDestination,
} from "../Integration/utils";
import {
  useAccessTotals,
  useFailingDeliveries,
  useIntegrationList,
} from "./queries";

const MAX_CARDS = 6;

const attentionRank = (item: Integration) => {
  if (item.spec?.isDisabled) return 4;
  const tone = getIntegrationPresentation(item).stateMeta.tone;
  if (tone === "danger") return 0;
  if (tone === "warning") return 1;
  if (tone === "success") return 2;
  return 3;
};

const IntegrationCard = (props: { item: Integration }) => {
  const location = useLocation();
  const { item } = props;
  const p = getIntegrationPresentation(item);
  const provider = getProviderKind(item);
  const destination = getSharedDestination(item);
  const tone = item.spec?.isDisabled ? "neutral" : p.stateMeta.tone;

  return (
    <Link
      to={`/access/integrations/${item.metadata?.name}`}
      state={{ returnTo: `${location.pathname}${location.search}` }}
      preventScrollReset
      className={twMerge(
        "group flex min-w-0 flex-col gap-2.5 rounded-xl border bg-white px-3.5 py-3 outline-none transition-[border-color,box-shadow] duration-150 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
        tone === "danger"
          ? "border-red-200 hover:border-red-300"
          : tone === "warning"
            ? "border-amber-200 hover:border-amber-300"
            : "border-slate-200 hover:border-slate-300",
      )}
    >
      <div className="flex items-center gap-2.5">
        <ProviderTile provider={provider} size="sm" />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-body font-semibold text-slate-800">
            {item.metadata?.displayName || item.metadata?.name}
          </span>
          <span className="truncate text-micro font-normal text-slate-500">
            {provider ? PROVIDER_LABEL[provider] : "Not configured"}
            {destination && ` · ${destination.value}`}
          </span>
        </span>
        <ToneBadge tone={tone}>
          {item.spec?.isDisabled ? "Disabled" : p.stateMeta.label}
        </ToneBadge>
      </div>
      {item.status?.lastError && !item.spec?.isDisabled ? (
        <span className="flex items-start gap-1.5 text-micro font-normal text-red-700">
          <AlertTriangle size={11} strokeWidth={2.4} className="mt-px shrink-0" />
          <span className="line-clamp-2 break-words">{item.status.lastError}</span>
        </span>
      ) : (
        <span className="text-micro font-normal text-slate-500">
          {item.status?.lastSuccessAt ? (
            <>
              Last success <TimeAgo rfc3339={item.status.lastSuccessAt} />
            </>
          ) : (
            "No successful operation yet"
          )}
        </span>
      )}
    </Link>
  );
};

const FailingRow = (props: { item: IntegrationBinding }) => {
  const location = useLocation();
  const status = props.item.status;
  return (
    <Link
      to={`/access/integrations/${status?.integrationRef?.name}`}
      state={{
        returnTo: `${location.pathname}${location.search}`,
        section: "activity",
      }}
      preventScrollReset
      className="group flex min-w-0 items-center gap-3 rounded-lg border border-red-200 bg-red-50/40 px-3 py-2 outline-none transition-[border-color,box-shadow] duration-150 hover:border-red-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-xs font-semibold text-slate-800">
          {status?.integrationRef?.name}
          <span className="ml-1.5 font-normal text-slate-500">
            {getDeliveryPurposeLabel(status?.purpose)}
            {status?.requestRef?.name && ` · ${status.requestRef.name}`}
          </span>
        </span>
        <span className="truncate text-micro font-normal text-red-700">
          {status?.lastError || "The last delivery failed"}
        </span>
      </span>
      <span className="shrink-0 text-micro font-semibold tabular-nums text-red-700">
        {status?.attempts} attempts
      </span>
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
  useChartColorScheme();
  const totals = useAccessTotals(props.periodMinutes, QUERY_PRIORITY.critical);
  const list = useIntegrationList();
  const failing = useFailingDeliveries();

  const deliveries = totals.data?.access?.integrationBinding;
  const all = list.data?.items ?? [];
  const items = [...all]
    .sort((a, b) => attentionRank(a) - attentionRank(b))
    .slice(0, MAX_CARDS);
  const failingItems = failing.data?.items ?? [];
  const totalIntegrations = n(totals.data?.access?.integration?.totalNumber);

  return (
    <Panel
      icon={Plug}
      title="Integrations"
      description="Where requests are presented and decided outside the access portal"
      to="/access/integrations"
      toLabel="All integrations"
      actions={
        <Link
          to="/access/integrations/create"
          className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-micro font-semibold text-slate-600 outline-none transition-colors duration-150 hover:border-slate-300 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <Plus size={12} strokeWidth={2.5} />
          Add
        </Link>
      }
    >
      <div className="flex flex-col gap-5">
        {list.isLoading ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {[0, 1, 2].map((index) => (
              <div
                key={index}
                className="h-[92px] motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50"
              />
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyHint>
            No Integration yet. Connect Slack, Jira or any system with a
            webhook so reviewers can decide where they already work.
          </EmptyHint>
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {items.map((item) => (
              <IntegrationCard key={item.metadata?.uid} item={item} />
            ))}
          </div>
        )}
        {totalIntegrations > items.length && items.length > 0 && (
          <Link
            to="/access/integrations"
            className="w-fit text-micro font-semibold text-blue-700 hover:text-blue-800"
          >
            {compact(totalIntegrations - items.length)} more Integrations
          </Link>
        )}

        <Breakdown
          title="Deliveries"
          note={`${compact(n(deliveries?.totalNumber))} messages, issues and webhook calls`}
        >
          <CompositionBar
            segments={[
              {
                label: "Delivered",
                value: n(deliveries?.totalReady),
                color: STATUS_COLORS.good,
              },
              {
                label: "Pending",
                value: n(deliveries?.totalPending),
                color: STATUS_COLORS.warning,
              },
              {
                label: "Failing",
                value: n(deliveries?.totalDegraded),
                color: STATUS_COLORS.critical,
              },
              { label: "Closed", value: n(deliveries?.totalClosed) },
            ]}
            total={n(deliveries?.totalNumber)}
          />
        </Breakdown>

        {failingItems.length > 0 && (
          <div className="flex flex-col gap-1.5">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-red-700">
              Failing deliveries · retried automatically
            </span>
            {failingItems.map((item) => (
              <FailingRow key={item.metadata?.uid} item={item} />
            ))}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Integrations;
