import * as AccessP from "@/apis/accessv1/accessv1";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { SegmentedControl } from "@mantine/core";
import {
  AlertTriangle,
  ArrowUpRight,
  BellRing,
  Inbox,
  Link2,
  RefreshCw,
  UserRound,
} from "lucide-react";
import * as React from "react";
import {
  EmptyNote,
  normalizeRef,
  RefChip,
  TONE_SOFT,
  ToneBadge,
} from "../common";
import {
  useIntegrationDeliveries,
  useIntegrationDeliverySummary,
  useIntegrationIdentities,
} from "./queries";
import {
  getAudienceLabel,
  getDeliveryPurposeLabel,
  getDeliveryStateMeta,
  getIdentitySourceLabel,
  isDeliveryFailing,
  isDeliveryOutOfDate,
} from "./utils";

const Count = (props: {
  label: string;
  value: number;
  tone?: "neutral" | "success" | "warning" | "danger";
  isLoading?: boolean;
}) => (
  <div
    className={twMerge(
      "flex min-w-0 flex-col gap-0.5 rounded-lg border px-3 py-2",
      props.value > 0 && props.tone && props.tone !== "neutral"
        ? TONE_SOFT[props.tone]
        : "border-slate-200 bg-white",
    )}
  >
    <span className="truncate text-micro font-semibold uppercase tracking-[0.06em] opacity-80">
      {props.label}
    </span>
    {props.isLoading ? (
      <span className="block h-6 w-10 motion-safe:animate-pulse rounded bg-slate-100" />
    ) : (
      <span className="text-lg font-bold leading-6 tabular-nums">
        {props.value.toLocaleString()}
      </span>
    )}
  </div>
);

export const DeliveryRow = (props: {
  item: AccessP.IntegrationBinding;
  showIntegration?: boolean;
}) => {
  const status = props.item.status;
  const meta = getDeliveryStateMeta(status?.state);
  const failing = isDeliveryFailing(props.item);
  const outOfDate = isDeliveryOutOfDate(props.item);
  const requestRef = normalizeRef(status?.requestRef, "access/v1", "Request");
  const userRef = normalizeRef(status?.userRef, "core/v1", "User");
  const integrationRef = normalizeRef(
    status?.integrationRef,
    "access/v1",
    "Integration",
  );
  const PurposeIcon =
    status?.purpose === AccessP.IntegrationBinding_Status_Purpose.NOTIFICATION
      ? BellRing
      : Inbox;

  return (
    <li
      className={twMerge(
        "flex flex-col gap-2 border-b border-slate-100 px-3.5 py-3 last:border-0",
        failing && "bg-red-50/40",
      )}
    >
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5">
        <PurposeIcon
          size={13}
          strokeWidth={2.3}
          aria-hidden="true"
          className="shrink-0 text-slate-500"
        />
        <span className="text-body font-semibold text-slate-800">
          {getDeliveryPurposeLabel(status?.purpose)}
        </span>
        {requestRef && <RefChip itemRef={requestRef} icon={Inbox} />}
        {props.showIntegration && integrationRef && (
          <RefChip itemRef={integrationRef} />
        )}
        <span className="text-micro font-normal text-slate-500">
          {getAudienceLabel(status?.audience)}
          {userRef?.name && ` · ${userRef.name}`}
          {status?.stepName && ` · step ${status.stepName}`}
          {status?.interactionMode ===
            AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE &&
            " · interactive"}
        </span>
        <span className="ml-auto flex items-center gap-1.5">
          {outOfDate && !failing && (
            <ToneBadge tone="warning">Updating</ToneBadge>
          )}
          <ToneBadge tone={meta.tone}>{meta.label}</ToneBadge>
        </span>
      </div>

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 pl-5 text-micro font-normal text-slate-500">
        {status?.lastSuccessAt && (
          <span>
            Last delivered <TimeAgo rfc3339={status.lastSuccessAt} />
          </span>
        )}
        {status?.externalRecipientID && (
          <span>
            To <span className="font-mono">{status.externalRecipientID}</span>
          </span>
        )}
        {status?.externalID && (
          <span>
            Object <span className="font-mono">{status.externalID}</span>
          </span>
        )}
        {status?.externalURL && (
          <a
            href={status.externalURL}
            target="_blank"
            rel="noreferrer noopener"
            className="inline-flex items-center gap-0.5 font-semibold text-blue-700 hover:text-blue-800"
          >
            Open
            <ArrowUpRight size={11} strokeWidth={2.5} />
          </a>
        )}
      </div>

      {failing && (
        <div className="ml-5 flex items-start gap-2 rounded-md border border-red-200 bg-white px-2.5 py-1.5 text-xs text-red-700">
          <AlertTriangle size={13} strokeWidth={2.3} className="mt-0.5 shrink-0" />
          <span className="min-w-0 break-words">
            {status?.lastError || "The last delivery failed"} ·{" "}
            {status?.attempts} failed attempt{status?.attempts === 1 ? "" : "s"}
            {status?.nextAttemptAt && (
              <>
                {" "}
                · next retry <TimeAgo rfc3339={status.nextAttemptAt} />
              </>
            )}
          </span>
        </div>
      )}
    </li>
  );
};

export const IdentityRow = (props: {
  item: AccessP.IntegrationIdentity;
  showIntegration?: boolean;
}) => {
  const status = props.item.status;
  const userRef = normalizeRef(status?.userRef, "core/v1", "User");
  const integrationRef = normalizeRef(
    status?.integrationRef,
    "access/v1",
    "Integration",
  );

  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-slate-100 px-3.5 py-2.5 last:border-0">
      {props.showIntegration && integrationRef && (
        <RefChip itemRef={integrationRef} />
      )}
      {userRef ? (
        <RefChip itemRef={userRef} icon={UserRound} />
      ) : (
        <span className="text-xs text-slate-500">Unknown User</span>
      )}
      <Link2 size={12} strokeWidth={2.3} aria-hidden="true" className="text-slate-400" />
      <span className="flex min-w-0 flex-col">
        <span className="truncate text-xs font-semibold text-slate-700">
          {status?.externalUsername || status?.externalID}
        </span>
        <span className="truncate font-mono text-micro text-slate-500">
          {status?.externalID}
          {status?.externalEmail && ` · ${status.externalEmail}`}
        </span>
      </span>
      <span className="ml-auto flex items-center gap-2 text-micro font-normal text-slate-500">
        {getIdentitySourceLabel(status?.source)}
        {status?.verifiedAt && (
          <span>
            · verified <TimeAgo rfc3339={status.verifiedAt} />
          </span>
        )}
      </span>
    </li>
  );
};

const IntegrationActivity = (props: { item: AccessP.Integration }) => {
  const [filter, setFilter] = React.useState<"all" | "failing">("all");
  const summary = useIntegrationDeliverySummary(props.item);
  const deliveries = useIntegrationDeliveries(props.item, {
    failingOnly: filter === "failing",
    itemsPerPage: 25,
  });
  const identities = useIntegrationIdentities(props.item);

  const total = Number(summary.data?.totalNumber ?? 0);
  const items = deliveries.data?.items ?? [];
  const identityItems = identities.data?.items ?? [];

  return (
    <div className="flex flex-col gap-5 px-4 py-4">
      <section className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="text-sm font-semibold text-slate-900">Deliveries</h3>
            <p className="text-xs font-normal text-slate-500">
              Every request this Integration presented, with the message or
              issue it created. Failed deliveries are retried automatically.
            </p>
          </div>
          <SegmentedControl
            size="xs"
            value={filter}
            onChange={(value) => setFilter(value as "all" | "failing")}
            data={[
              { value: "all", label: "All" },
              { value: "failing", label: "Failing" },
            ]}
          />
        </div>

        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <Count label="Total" value={total} isLoading={summary.isLoading} />
          <Count
            label="Delivered"
            value={Number(summary.data?.totalReady ?? 0)}
            tone="success"
            isLoading={summary.isLoading}
          />
          <Count
            label="Failing"
            value={Number(summary.data?.totalFailing ?? 0)}
            tone="danger"
            isLoading={summary.isLoading}
          />
          <Count
            label="Updating"
            value={Number(summary.data?.totalOutOfDate ?? 0)}
            tone="warning"
            isLoading={summary.isLoading}
          />
        </div>

        {deliveries.isLoading ? (
          <div className="h-40 motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ) : deliveries.isError ? (
          <EmptyNote>The deliveries could not be loaded.</EmptyNote>
        ) : items.length === 0 ? (
          <EmptyNote>
            {filter === "failing"
              ? "No delivery is failing."
              : "Nothing was delivered yet. Use this Integration as a surface of a Policy rule, then create a matching request."}
          </EmptyNote>
        ) : (
          <ul className="overflow-hidden rounded-xl border border-slate-200 bg-white">
            {items.map((item) => (
              <DeliveryRow key={item.metadata?.uid} item={item} />
            ))}
          </ul>
        )}
        {(deliveries.data?.listResponseMeta?.totalCount ?? 0) > items.length && (
          <span className="flex items-center gap-1 text-micro font-normal text-slate-500">
            <RefreshCw size={11} strokeWidth={2.4} />
            Showing the {items.length} most recent of{" "}
            {deliveries.data?.listResponseMeta?.totalCount.toLocaleString()}.
          </span>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">
            Linked accounts
          </h3>
          <p className="text-xs font-normal text-slate-500">
            Provider accounts the Cluster matched to its Users. They are
            created automatically the first time someone needs to be resolved.
          </p>
        </div>
        {identities.isLoading ? (
          <div className="h-24 motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50" />
        ) : identityItems.length === 0 ? (
          <EmptyNote>
            No account is linked yet. Use Resolve identity in the header to
            check how a person resolves.
          </EmptyNote>
        ) : (
          <ul className="overflow-hidden rounded-xl border border-slate-200 bg-white">
            {identityItems.map((item) => (
              <IdentityRow key={item.metadata?.uid} item={item} />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
};

export default IntegrationActivity;
