import * as AccessC from "@/apis/accessv1/accessv1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListPolicyOptions,
  ListRequestOptions,
  ListRequestOptions_OrderBy_Type,
} from "@/apis/visibilityv1/access/vaccessv1";
import {
  CommonListOptions,
  CommonListOptions_OrderBy_Mode,
  CommonSummaryOptions,
} from "@/apis/visibilityv1/meta/vmetav1";
import InfoItem from "@/components/InfoItem";
import { twMerge } from "@/lib/utils";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { ArrowUpRight, Folder, Server, Shield } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { EffectBadge, EmptyNote, RefChip } from "../common";
import { LiveRow, PendingRow, useNow } from "../Overview/Queues";
import { ConditionView } from "../rules";

const WINDOW_DAYS = 30;

const useCatalogQuery = <T,>(
  item: AccessC.Catalog,
  key: string,
  fn: (signal: AbortSignal, ref: ObjectReference) => Promise<T>,
) =>
  useQuery({
    queryKey: ["access", "catalog", item.metadata?.uid, key],
    enabled: !!item.metadata?.uid,
    staleTime: 15_000,
    refetchOnWindowFocus: false,
    queryFn: ({ signal }) => fn(signal, getResourceRef(item)),
  });

const useCatalogActivity = (item: AccessC.Catalog) =>
  useCatalogQuery(item, "activity", async (signal, ref) => {
    const to = new Date();
    const from = new Date(to.getTime() - WINDOW_DAYS * 24 * 3600 * 1000);
    return (
      await getClientVisibilityAccess().getRequestSummary(
        {
          common: CommonSummaryOptions.create({
            from: Timestamp.fromDate(from),
            to: Timestamp.fromDate(to),
          }),
          catalogRef: ref,
        },
        { abort: signal },
      )
    ).response;
  });

const useCatalogRequests = (
  item: AccessC.Catalog,
  state: AccessC.Request_Status_State_Status,
) =>
  useCatalogQuery(item, `requests-${state}`, async (signal, ref) =>
    (
      await getClientVisibilityAccess().listRequest(
        ListRequestOptions.create({
          common: CommonListOptions.create({ itemsPerPage: 6 }),
          catalogRef: ref,
          state,
          ...(state === AccessC.Request_Status_State_Status.APPROVED
            ? {
                orderBy: {
                  type: ListRequestOptions_OrderBy_Type.ACCESS_ENDS_AT,
                  mode: CommonListOptions_OrderBy_Mode.ASC,
                },
              }
            : {}),
        }),
        { abort: signal },
      )
    ).response,
  );

const useCatalogPolicies = (item: AccessC.Catalog) =>
  useCatalogQuery(item, "policies", async (signal, ref) =>
    (
      await getClientVisibilityAccess().listPolicy(
        ListPolicyOptions.create({
          common: CommonListOptions.create({ itemsPerPage: 100 }),
          catalogRef: ref,
        }),
        { abort: signal },
      )
    ).response,
  );

const referencesCatalog = (
  condition: AccessC.Policy_Spec_Rule_Condition | undefined,
  catalog: AccessC.Catalog,
): boolean => {
  const type = condition?.type;
  if (!type) return false;
  if (type.oneofKind === "resource" && type.resource.type.oneofKind === "catalogRef") {
    const ref = type.resource.type.catalogRef;
    return ref.uid === catalog.metadata?.uid || ref.name === catalog.metadata?.name;
  }
  if (type.oneofKind === "all")
    return type.all.of.some((child) => referencesCatalog(child, catalog));
  if (type.oneofKind === "any")
    return type.any.of.some((child) => referencesCatalog(child, catalog));
  return false;
};

const Stat = (props: {
  label: string;
  value?: number;
  to?: string;
  tone?: "success" | "danger" | "warning";
  loading?: boolean;
}) => {
  const body = (
    <>
      <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
        {props.label}
      </span>
      <span
        className={twMerge(
          "text-lg font-bold tabular-nums text-slate-900",
          !!props.value && props.tone === "success" && "text-emerald-700",
          !!props.value && props.tone === "danger" && "text-red-700",
          !!props.value && props.tone === "warning" && "text-amber-700",
        )}
      >
        {props.loading ? (
          <span className="block h-6 w-10 motion-safe:animate-pulse rounded bg-slate-100" />
        ) : (
          (props.value ?? "—")
        )}
      </span>
    </>
  );
  const className =
    "flex min-w-0 flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2";
  return props.to ? (
    <Link
      to={props.to}
      className={twMerge(
        className,
        "outline-none transition-colors duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {body}
    </Link>
  ) : (
    <div className={className}>{body}</div>
  );
};

const Activity = (props: { item: AccessC.Catalog }) => {
  const query = useCatalogActivity(props.item);
  const data = query.data;
  const base = `/access/requests?catalogRef.name=${encodeURIComponent(props.item.metadata?.name ?? "")}`;
  const value = (x?: number) => (data ? Number(x ?? 0) : undefined);

  return (
    <div className="flex flex-col gap-2 border-b border-slate-200 bg-white px-4 py-4">
      <span className="flex items-center justify-between gap-2">
        <span className="text-xs font-semibold text-slate-700">
          Requests for this Catalog · last {WINDOW_DAYS} days
        </span>
        <Link
          to={base}
          className="inline-flex items-center gap-0.5 text-micro font-semibold text-blue-700 hover:text-blue-800"
        >
          All its requests <ArrowUpRight size={11} strokeWidth={2.5} />
        </Link>
      </span>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
        <Stat label="Requests" value={value(data?.totalNumber)} loading={query.isLoading} to={base} />
        <Stat label="Granted" value={value(data?.totalGranted)} loading={query.isLoading} tone="success" />
        <Stat label="Rejected" value={value(data?.totalRejected)} loading={query.isLoading} tone="danger" />
        <Stat
          label="Pending"
          value={value(data?.totalPending)}
          loading={query.isLoading}
          tone="warning"
          to={`${base}&state=PENDING`}
        />
        <Stat label="Requesters" value={value(data?.totalUser)} loading={query.isLoading} />
      </div>
    </div>
  );
};

const RequestRows = (props: {
  item: AccessC.Catalog;
  state: AccessC.Request_Status_State_Status;
  empty: string;
}) => {
  const query = useCatalogRequests(props.item, props.state);
  const now = useNow();
  if (query.isLoading)
    return <span className="block h-14 motion-safe:animate-pulse rounded-lg bg-slate-100" />;
  const items = query.data?.items ?? [];
  if (items.length === 0) return <EmptyNote>{props.empty}</EmptyNote>;
  return (
    <div className="flex flex-col gap-1.5">
      {items.map((request) =>
        props.state === AccessC.Request_Status_State_Status.APPROVED ? (
          <LiveRow key={request.metadata?.uid} item={request} now={now} />
        ) : (
          <PendingRow key={request.metadata?.uid} item={request} now={now} />
        ),
      )}
    </div>
  );
};

const Governance = (props: { item: AccessC.Catalog }) => {
  const location = useLocation();
  const query = useCatalogPolicies(props.item);
  if (query.isLoading)
    return <span className="block h-14 motion-safe:animate-pulse rounded-lg bg-slate-100" />;

  const rules = (query.data?.items ?? []).flatMap((policy) =>
    (policy.spec?.rules ?? [])
      .filter((rule) => referencesCatalog(rule.condition, props.item))
      .map((rule) => ({ policy, rule })),
  );
  rules.sort((a, b) => a.rule.priority - b.rule.priority);

  return (
    <div className="flex flex-col gap-2">
      {rules.length === 0 ? (
        <EmptyNote>
          No rule names this Catalog. Its requests are decided by broader
          rules, or rejected when nothing matches.{" "}
          <Link
            to="/access/policies/create"
            state={{ returnTo: `${location.pathname}${location.search}` }}
            className="font-semibold text-blue-700 hover:text-blue-800"
          >
            Create a Policy
          </Link>
        </EmptyNote>
      ) : (
        <ul className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200">
          {rules.map(({ policy, rule }) => (
            <li
              key={`${policy.metadata?.uid}-${rule.name}`}
              className={twMerge(
                "flex flex-col gap-1.5 px-3 py-2",
                policy.spec?.isDisabled && "opacity-60",
              )}
            >
              <span className="flex flex-wrap items-center gap-2">
                <RefChip
                  itemRef={ObjectReference.create({
                    apiVersion: "access/v1",
                    kind: "Policy",
                    name: policy.metadata?.name,
                    uid: policy.metadata?.uid,
                  })}
                  icon={Shield}
                />
                <span className="text-xs font-semibold text-slate-800">{rule.name}</span>
                <EffectBadge effect={rule.effect} />
                {policy.spec?.isDisabled && (
                  <span className="text-micro font-semibold text-slate-500">Policy disabled</span>
                )}
                <span className="ml-auto text-micro tabular-nums text-slate-500">
                  priority {rule.priority > 0 ? `+${rule.priority}` : rule.priority}
                </span>
              </span>
              <span className="text-xs text-slate-600">
                <ConditionView condition={rule.condition} />
              </span>
            </li>
          ))}
        </ul>
      )}
      <span className="text-micro leading-4 text-slate-500">
        Rules that do not name this Catalog, such as Group or CEL conditions,
        can decide its requests too. The first matching rule wins.
      </span>
    </div>
  );
};

const Contents = (props: { values: string[]; kind: "Service" | "Namespace" }) =>
  props.values.length === 0 ? (
    <span className="text-xs text-slate-500">None</span>
  ) : (
    <span className="flex flex-wrap gap-1.5">
      {props.values.map((name) => (
        <RefChip
          key={name}
          itemRef={ObjectReference.create({ apiVersion: "core/v1", kind: props.kind, name })}
          icon={props.kind === "Service" ? Server : Folder}
        />
      ))}
    </span>
  );

export const ItemInfo = (props: { item: AccessC.Catalog }) => {
  const service = props.item.spec?.resourceCollection?.service;
  return (
    <>
      <InfoItem title="Services">
        <span>{service?.services.length ?? 0}</span>
      </InfoItem>
      <InfoItem title="Namespaces">
        <span>{service?.namespaces.length ?? 0}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Catalog }) => (
  <div className="w-full">
    <ItemInfo item={props.item} />
  </div>
);

export const MainInfo = (props: { item: AccessC.Catalog }): ResourceMainInfo => {
  const { item } = props;
  const service = item.spec?.resourceCollection?.service;
  const services = service?.services.filter(Boolean) ?? [];
  const namespaces = service?.namespaces.filter(Boolean) ?? [];

  return {
    header: <Activity item={item} />,
    groupOrder: ["Contents", "Decided by", "Live access", "Waiting for a decision"],
    items: [
      {
        label: "Services",
        primary: true,
        value: <span className="font-semibold">{services.length}</span>,
      },
      {
        label: "Namespaces",
        primary: true,
        value: <span className="font-semibold">{namespaces.length}</span>,
      },
      {
        label: "Services",
        group: "Contents",
        span: "full" as const,
        value: <Contents kind="Service" values={services} />,
      },
      {
        label: "Namespaces",
        group: "Contents",
        span: "full" as const,
        value: (
          <span className="flex flex-col gap-1">
            <Contents kind="Namespace" values={namespaces} />
            {namespaces.length > 0 && (
              <span className="text-micro text-slate-500">
                Every Service of these Namespaces, including the ones added later.
              </span>
            )}
          </span>
        ),
      },
      {
        label: "",
        group: "Decided by",
        layout: "stacked" as const,
        value: <Governance item={item} />,
      },
      {
        label: "",
        group: "Live access",
        layout: "stacked" as const,
        value: (
          <RequestRows
            item={item}
            state={AccessC.Request_Status_State_Status.APPROVED}
            empty="Nobody has access through this Catalog right now."
          />
        ),
      },
      {
        label: "",
        group: "Waiting for a decision",
        layout: "stacked" as const,
        value: (
          <RequestRows
            item={item}
            state={AccessC.Request_Status_State_Status.PENDING}
            empty="No pending request for this Catalog."
          />
        ),
      },
    ],
  };
};
