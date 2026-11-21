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
import {
  AlertTriangle,
  ArrowUpRight,
  Ban,
  Folder,
  Hourglass,
  KeyRound,
  Layers,
  Scale,
  Server,
  Shield,
  User,
  UserCog,
  Users,
} from "lucide-react";
import { Link, useLocation } from "react-router";
import { EffectBadge, EmptyNote, formatDurationShort, RefChip } from "../common";
import { LiveRow, PendingRow, useNow } from "../Overview/Queues";
import { ConditionView, ConfigurationIssues, GrantSummary } from "../rules";

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

const Principals = (props: { users: string[]; groups: string[]; empty: string }) =>
  props.users.length === 0 && props.groups.length === 0 ? (
    <span className="text-xs text-slate-500">{props.empty}</span>
  ) : (
    <span className="flex flex-wrap gap-1.5">
      {props.groups.map((name) => (
        <RefChip
          key={`group-${name}`}
          itemRef={ObjectReference.create({ apiVersion: "core/v1", kind: "Group", name })}
          icon={Users}
        />
      ))}
      {props.users.map((name) => (
        <RefChip
          key={`user-${name}`}
          itemRef={ObjectReference.create({ apiVersion: "core/v1", kind: "User", name })}
          icon={User}
        />
      ))}
    </span>
  );

const AccessLevels = (props: { levels: AccessC.Catalog_Spec_AccessLevel[] }) =>
  props.levels.length === 0 ? (
    <span className="text-xs text-amber-700">
      No access level: this Catalog cannot be requested.
    </span>
  ) : (
    <ul className="flex flex-col gap-2">
      {props.levels.map((level, index) => (
        <li
          key={level.name}
          className="flex flex-col gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2"
        >
          <span className="flex flex-wrap items-center gap-2">
            <GrantSummary
              level={level}
              overridesGuardrails={
                level.type.oneofKind === "full" && (level.type.full.priority ?? 0) < 0
              }
              compact
            />
            {index === 0 && <span className="text-micro text-slate-500">default</span>}
          </span>
          {level.description && (
            <span className="text-xs leading-5 text-slate-600">{level.description}</span>
          )}
          {(level.defaultDuration || (level.verification?.minAAL ?? 0) > 0) && (
            <span className="text-micro text-slate-500">
              {level.defaultDuration &&
                `${formatDurationShort(level.defaultDuration)} by default`}
              {level.defaultDuration && (level.verification?.minAAL ?? 0) > 0 && " · "}
              {(level.verification?.minAAL ?? 0) > 0 &&
                `requires authentication assurance level ${level.verification!.minAAL}`}
            </span>
          )}
        </li>
      ))}
    </ul>
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
  const spec = item.spec;
  const levels = spec?.accessLevels ?? [];
  const audience = spec?.audience;
  const issues = item.status?.issues ?? [];
  const dynamic = spec?.membership === AccessC.Catalog_Spec_Membership.DYNAMIC;
  const ticket = spec?.requirements?.ticket;
  const justification = spec?.requirements?.justification;

  return {
    header: <Activity item={item} />,
    groupOrder: [
      ...(issues.length > 0 ? [{ title: "Configuration issues", icon: AlertTriangle }] : []),
      { title: "Access levels", icon: KeyRound },
      { title: "Who can request", icon: Users },
      { title: "Contents", icon: Folder },
      { title: "Decided by", icon: Scale },
      { title: "Live access", icon: KeyRound },
      { title: "Waiting for a decision", icon: Hourglass },
    ],
    items: [
      ...(spec?.isDisabled
        ? [
            {
              label: "State",
              primary: true,
              icon: Ban,
              value: <span className="font-semibold text-slate-600">Disabled</span>,
            },
          ]
        : []),
      {
        label: "Services",
        primary: true,
        icon: Server,
        value: (
          <span className="font-semibold">
            {item.status?.checkedAt ? item.status.totalService : services.length}
          </span>
        ),
      },
      {
        label: "Namespaces",
        primary: true,
        icon: Layers,
        value: <span className="font-semibold">{namespaces.length}</span>,
      },
      {
        label: "Access levels",
        primary: true,
        icon: KeyRound,
        value: <span className="font-semibold">{levels.length}</span>,
      },
      ...(issues.length > 0
        ? [
            {
              label: "Issues",
              primary: true,
              icon: AlertTriangle,
              value: <span className="font-semibold text-amber-700">{issues.length}</span>,
            },
            {
              label: "",
              group: "Configuration issues",
              layout: "stacked" as const,
              value: (
                <ConfigurationIssues issues={issues} checkedAt={item.status?.checkedAt} />
              ),
            },
          ]
        : []),
      {
        label: "",
        group: "Access levels",
        layout: "stacked" as const,
        value: <AccessLevels levels={levels} />,
      },
      {
        label: "Requesters",
        group: "Who can request",
        span: "full" as const,
        value: audience?.match ? (
          <span className="flex flex-col gap-1">
            <Principals
              users={audience.users}
              groups={audience.groups}
              empty="Users matching the expression below"
            />
            <code className="break-all rounded bg-slate-100 px-1.5 py-0.5 font-mono text-micro text-slate-700">
              {audience.match}
            </code>
          </span>
        ) : (
          <Principals
            users={audience?.users ?? []}
            groups={audience?.groups ?? []}
            empty="Every User"
          />
        ),
      },
      {
        label: "On behalf of others",
        group: "Who can request",
        span: "full" as const,
        value: (
          <Principals
            users={audience?.onBehalf?.users ?? []}
            groups={audience?.onBehalf?.groups ?? []}
            empty="Nobody"
          />
        ),
      },
      {
        label: "Owners",
        group: "Who can request",
        span: "full" as const,
        value: (
          <span className="flex items-center gap-1.5">
            <UserCog size={12} className="shrink-0 text-slate-500" />
            <Principals
              users={spec?.owners?.users ?? []}
              groups={spec?.owners?.groups ?? []}
              empty="No owner"
            />
          </span>
        ),
      },
      {
        label: "Requirements",
        group: "Who can request",
        span: "full" as const,
        value: (
          <span className="text-xs text-slate-700">
            {[
              justification?.isRequired &&
                `Justification${justification.minLength ? ` of at least ${justification.minLength} characters` : ""}`,
              ticket?.isRequired &&
                `${ticket.label || "Ticket"}${ticket.pattern ? ` matching ${ticket.pattern}` : ""}`,
              spec?.allowPartialRequests && "Requesters may pick individual Services",
            ]
              .filter(Boolean)
              .join(" · ") || "None"}
          </span>
        ),
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
                {dynamic
                  ? "Every Service of these Namespaces, including the ones added during an approved access."
                  : "The Services of these Namespaces when a request is approved."}
                {spec?.allowSystemResources
                  ? " System Services are included."
                  : " System Services are excluded."}
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
