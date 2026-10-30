import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import InfoItem from "@/components/InfoItem";
import TimeAgo from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getDomain, onError } from "@/utils";
import { getClientAccess } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Button, Tooltip } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  BookOpen,
  Check,
  Hash,
  KeyRound,
  Loader2,
  PowerOff,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import * as React from "react";
import { notifications } from "@mantine/notifications";
import {
  Callout,
  CopyField,
  EmptyNote,
  getProviderKind,
  InlineCopy,
  PROVIDER_LABEL,
  ProviderTile,
  RefChip,
  ToneBadge,
} from "../common";
import IntegrationActivity from "./Activity";
import {
  getIntegrationUsages,
  useAccessPolicies,
  useIntegrationDeliverySummary,
  useIntegrationIdentities,
} from "./queries";
import {
  getCapabilityLabel,
  getInboundEndpoints,
  getIntegrationPresentation,
  getSyncStateMeta,
} from "./utils";

const IntegrationGuide = React.lazy(() => import("./Guide"));

const CAPABILITY_HINTS: Record<number, string> = {
  [AccessC.Integration_Status_Capability.NOTIFICATION]:
    "Can post to its shared destination (channel, project or URL).",
  [AccessC.Integration_Status_Capability.DIRECT_USER_DELIVERY]:
    "Can deliver directly to individual people, e.g. reviewer DMs.",
  [AccessC.Integration_Status_Capability.INTERACTIVE_REVIEW]:
    "Can submit reviewers' decisions back to the Cluster.",
  [AccessC.Integration_Status_Capability.REQUEST_CREATION]:
    "Can create access requests on behalf of its users.",
  [AccessC.Integration_Status_Capability.IDENTITY_RESOLUTION]:
    "Can match its accounts to Cluster Users by email.",
  [AccessC.Integration_Status_Capability.PRESENTATION_UPDATE]:
    "Keeps delivered messages up to date as requests move on.",
};

const SynchronizeButton = (props: { item: AccessC.Integration }) => {
  const p = getIntegrationPresentation(props.item);
  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientAccess().synchronizeIntegration({
          integrationRef: getResourceRef(props.item),
        })
      ).response,
    onSuccess: () => {
      notifications.show({
        color: "green",
        message: "Synchronization requested",
      });
      invalidateResource(props.item);
      invalidateResourceList(props.item);
    },
    onError,
  });

  return (
    <Button
      type="button"
      color="ink"
      size="compact-xs"
      leftSection={<RefreshCw size={12} />}
      disabled={props.item.spec?.isDisabled || !p.isConfigured}
      loading={mutation.isPending || p.isSynchronizing}
      title={
        props.item.spec?.isDisabled
          ? "Enable the Integration before synchronizing"
          : undefined
      }
      onClick={() => mutation.mutate()}
    >
      Synchronize now
    </Button>
  );
};

const describeDestination = (item: AccessC.Integration) => {
  const type = item.spec?.type;
  if (type?.oneofKind === "slack")
    return type.slack.channelID
      ? `Channel ${type.slack.channelID}`
      : "No shared channel";
  if (type?.oneofKind === "jira")
    return type.jira.projectKey
      ? `Project ${type.jira.projectKey}`
      : "No Jira project";
  if (type?.oneofKind === "webhook") return type.webhook.url || "No URL";
  return "Not configured";
};

const HealthHeader = (props: { item: AccessC.Integration }) => {
  const { item } = props;
  const p = getIntegrationPresentation(item);
  const provider = getProviderKind(item);
  const status = item.status;

  return (
    <div className="flex flex-col gap-3 border-b border-slate-200 bg-white px-4 py-4">
      <div className="flex flex-wrap items-center gap-3">
        <ProviderTile provider={provider} size="lg" />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold text-slate-900">
              {provider ? PROVIDER_LABEL[provider] : "No provider"}
              {status?.externalTenantName && (
                <span className="font-normal text-slate-500">
                  {" "}
                  · {status.externalTenantName}
                </span>
              )}
            </span>
            <ToneBadge tone={item.spec?.isDisabled ? "neutral" : p.stateMeta.tone}>
              {item.spec?.isDisabled ? "Disabled" : p.stateMeta.label}
            </ToneBadge>
          </div>
          <span className="truncate text-xs font-normal text-slate-500">
            {describeDestination(item)}
            {status?.lastSuccessAt && (
              <>
                {" · "}last success <TimeAgo rfc3339={status.lastSuccessAt} />
              </>
            )}
          </span>
        </div>
      </div>

      {item.spec?.isDisabled && (
        <Callout tone="warning" icon={PowerOff} title="This Integration is disabled">
          It delivers nothing and rejects every inbound request until it is
          enabled again.
        </Callout>
      )}
      {!item.spec?.isDisabled && status?.lastError && (
        <Callout
          tone={
            status.state === AccessC.Integration_Status_State.ERROR
              ? "danger"
              : "warning"
          }
          icon={AlertTriangle}
          title={
            <>
              Last operation failed
              {status.lastFailureAt && (
                <span className="font-normal">
                  {" "}
                  · <TimeAgo rfc3339={status.lastFailureAt} />
                </span>
              )}
            </>
          }
        >
          <span className="break-words font-mono text-micro">
            {status.lastError}
          </span>
        </Callout>
      )}
      {!item.spec?.isDisabled && !status?.id && (
        <Callout tone="info" icon={RefreshCw} title="Not synchronized yet">
          Synchronize the Integration to verify its credentials and get its
          callback URLs. The Setup guide tab walks through every step.
        </Callout>
      )}
    </div>
  );
};

const DeliveriesTile = (props: { item: AccessC.Integration }) => {
  const summary = useIntegrationDeliverySummary(props.item);
  if (summary.isLoading)
    return <span className="block h-5 w-16 motion-safe:animate-pulse rounded bg-slate-100" />;
  const total = Number(summary.data?.totalNumber ?? 0);
  const failing = Number(summary.data?.totalFailing ?? 0);
  return (
    <span className="flex flex-wrap items-baseline gap-2">
      <span className="text-lg font-bold tabular-nums text-slate-900">
        {total.toLocaleString()}
      </span>
      {failing > 0 ? (
        <span className="text-xs font-semibold text-red-700">
          {failing} failing
        </span>
      ) : (
        total > 0 && (
          <span className="inline-flex items-center gap-0.5 text-xs font-normal text-emerald-700">
            <Check size={11} strokeWidth={2.6} /> healthy
          </span>
        )
      )}
    </span>
  );
};

const LinkedAccountsTile = (props: { item: AccessC.Integration }) => {
  const identities = useIntegrationIdentities(props.item);
  if (identities.isLoading)
    return <span className="block h-5 w-10 motion-safe:animate-pulse rounded bg-slate-100" />;
  return (
    <span className="text-lg font-bold tabular-nums text-slate-900">
      {(identities.data?.listResponseMeta?.totalCount ?? 0).toLocaleString()}
    </span>
  );
};

const UsedByTile = (props: { item: AccessC.Integration }) => {
  const policies = useAccessPolicies();
  if (policies.isLoading)
    return <span className="block h-5 w-10 motion-safe:animate-pulse rounded bg-slate-100" />;
  const usages = getIntegrationUsages(policies.data?.items ?? [], props.item);
  const rules = new Set(
    usages.map((usage) => `${usage.policy.metadata?.uid}/${usage.rule.name}`),
  );
  return (
    <span className="flex flex-wrap items-baseline gap-2">
      <span className="text-lg font-bold tabular-nums text-slate-900">
        {rules.size}
      </span>
      <span className="text-xs font-normal text-slate-500">
        rule{rules.size === 1 ? "" : "s"}
      </span>
    </span>
  );
};

const UsageList = (props: { item: AccessC.Integration }) => {
  const policies = useAccessPolicies();
  if (policies.isLoading)
    return <span className="block h-6 w-48 motion-safe:animate-pulse rounded bg-slate-100" />;

  const usages = getIntegrationUsages(policies.data?.items ?? [], props.item);
  if (usages.length === 0) {
    return (
      <EmptyNote>
        No Policy rule uses this Integration yet, so nothing is delivered. Add
        it as a review surface or an outcome notification of a rule.
      </EmptyNote>
    );
  }

  return (
    <ul className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200">
      {usages.map((usage, index) => (
        <li
          key={`${usage.policy.metadata?.uid}-${usage.rule.name}-${index}`}
          className="flex flex-wrap items-center gap-2 px-3 py-2"
        >
          <RefChip
            itemRef={ObjectReference.create({
              apiVersion: "access/v1",
              kind: "Policy",
              name: usage.policy.metadata?.name,
              uid: usage.policy.metadata?.uid,
            })}
            icon={ShieldCheck}
          />
          <span className="text-xs font-semibold text-slate-700">
            {usage.rule.name}
          </span>
          <span className="text-micro font-normal text-slate-500">
            {usage.isNotification
              ? "Outcome notification"
              : `Review step ${usage.stepName}`}
          </span>
          <span className="ml-auto flex items-center gap-1.5">
            <ToneBadge tone="neutral">
              {usage.surface.destination?.audience ===
              AccessC.Policy_Spec_Rule_Surface_Destination_Audience.SHARED
                ? "Shared"
                : usage.surface.destination?.audience ===
                    AccessC.Policy_Spec_Rule_Surface_Destination_Audience
                      .REVIEWERS
                  ? "Reviewers' DMs"
                  : usage.surface.destination?.audience ===
                      AccessC.Policy_Spec_Rule_Surface_Destination_Audience
                        .REQUESTER
                    ? "Requester's DM"
                    : "Subject's DM"}
            </ToneBadge>
            {!usage.isNotification &&
              usage.surface.interactionMode ===
                AccessC.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE && (
                <ToneBadge tone="info">Interactive</ToneBadge>
              )}
          </span>
        </li>
      ))}
    </ul>
  );
};

const SecretChips = (props: { names: string[] }) =>
  props.names.length > 0 ? (
    <div className="flex flex-wrap gap-1.5">
      {props.names.map((name) => (
        <RefChip
          key={name}
          itemRef={ObjectReference.create({
            apiVersion: "access/v1",
            kind: "Secret",
            name,
          })}
          icon={KeyRound}
        />
      ))}
    </div>
  ) : (
    <span className="text-xs text-slate-500">None</span>
  );

export const ItemInfo = (props: { item: AccessC.Integration }) => {
  const p = getIntegrationPresentation(props.item);
  return (
    <>
      <InfoItem title="Provider">
        <span>{p.type}</span>
      </InfoItem>
      <InfoItem title="State">
        <span>{p.stateMeta.label}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Integration }) => (
  <div className="w-full">
    <ItemInfo item={props.item} />
  </div>
);

const muted = (text: string) => (
  <span className="text-xs font-normal text-slate-500">{text}</span>
);

export const MainInfo = (props: {
  item: AccessC.Integration;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getIntegrationPresentation(item);
  const type = item.spec?.type;
  const status = item.status;
  const endpoints = getInboundEndpoints(item, getDomain());

  const providerItems =
    type?.oneofKind === "slack"
      ? [
          {
            label: "Channel",
            value: type.slack.channelID ? (
              <span className="inline-flex items-center gap-1">
                <Hash size={12} className="text-slate-400" />
                <InlineCopy value={type.slack.channelID} />
              </span>
            ) : (
              muted("Not set · shared deliveries are disabled")
            ),
          },
          {
            label: "Workspace",
            value: type.slack.teamID ? (
              <InlineCopy value={type.slack.teamID} />
            ) : status?.externalTenantID ? (
              <span>
                <InlineCopy value={status.externalTenantID} />{" "}
                {muted("(discovered)")}
              </span>
            ) : (
              muted("Discovered on synchronization")
            ),
          },
          ...(type.slack.mentionUserGroupID
            ? [
                {
                  label: "Mentions",
                  value: <InlineCopy value={type.slack.mentionUserGroupID} />,
                },
              ]
            : []),
          ...(type.slack.baseURL
            ? [
                {
                  label: "API base URL",
                  value: <InlineCopy value={type.slack.baseURL} />,
                },
              ]
            : []),
        ]
      : type?.oneofKind === "jira"
        ? [
            { label: "Site", value: <InlineCopy value={type.jira.url} /> },
            { label: "Account", value: <InlineCopy value={type.jira.email} /> },
            {
              label: "Project",
              value: type.jira.projectKey ? (
                <span>
                  <InlineCopy value={type.jira.projectKey} />{" "}
                  {muted(`· ${type.jira.issueTypeName || "Task"} issues`)}
                </span>
              ) : (
                muted("Not set · issues cannot be created")
              ),
            },
            {
              label: "Decisions",
              value:
                type.jira.approveStatus || type.jira.rejectStatus ? (
                  <span className="flex flex-wrap items-center gap-1.5">
                    <ToneBadge tone="success">
                      {type.jira.approveStatus || "No approving status"}
                    </ToneBadge>
                    <ToneBadge tone="danger">
                      {type.jira.rejectStatus || "No rejecting status"}
                    </ToneBadge>
                    {!type.jira.webhookSecret && muted("· no webhook secret")}
                  </span>
                ) : (
                  muted("Made in the access portal only")
                ),
            },
          ]
        : type?.oneofKind === "webhook"
          ? [
              {
                label: "Delivery URL",
                span: "full" as const,
                value: <InlineCopy value={type.webhook.url} />,
              },
              ...(type.webhook.name
                ? [{ label: "Name", value: <InlineCopy value={type.webhook.name} /> }]
                : []),
              {
                label: "Inbound calls",
                value: type.webhook.inboundSecret
                  ? "Accepted (signed)"
                  : muted("Rejected · no inbound secret"),
              },
            ]
          : [];

  const lastSync = p.currentSync;
  const syncMeta = getSyncStateMeta(lastSync?.state);

  return {
    actions: <SynchronizeButton item={item} />,
    header: <HealthHeader item={item} />,
    groupOrder: [
      "Connection",
      "Callback URLs",
      "Used by",
      "Capabilities",
      "Synchronization",
    ],
    items: [
      {
        label: "Deliveries",
        primary: true,
        value: <DeliveriesTile item={item} />,
      },
      {
        label: "Linked accounts",
        primary: true,
        value: <LinkedAccountsTile item={item} />,
      },
      {
        label: "Used by",
        primary: true,
        value: <UsedByTile item={item} />,
      },

      ...providerItems.map((entry) => ({ ...entry, group: "Connection" })),
      {
        label: "Secrets",
        group: "Connection",
        value: <SecretChips names={p.secretNames} />,
      },
      {
        label: "Email matching",
        group: "Connection",
        value: item.spec?.identityResolution?.disableEmailDiscovery
          ? muted("Off · only existing linked accounts work")
          : "On",
      },
      ...(status?.id
        ? [
            {
              label: "Integration ID",
              group: "Connection",
              value: <InlineCopy value={status.id} />,
            },
          ]
        : []),

      ...(endpoints.length > 0
        ? [
            {
              label: "",
              group: "Callback URLs",
              layout: "stacked" as const,
              value: (
                <div className="grid grid-cols-1 gap-2 @3xl:grid-cols-2">
                  {endpoints.map((endpoint) => (
                    <CopyField
                      key={endpoint.key}
                      label={`${endpoint.label}${endpoint.isOptional ? " · optional" : ""}`}
                      value={endpoint.url}
                      hint={endpoint.hint}
                    />
                  ))}
                </div>
              ),
            },
          ]
        : []),

      {
        label: "",
        group: "Used by",
        layout: "stacked" as const,
        value: <UsageList item={item} />,
      },

      {
        label: "",
        group: "Capabilities",
        layout: "stacked" as const,
        value:
          p.capabilities.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {p.capabilities.map((capability) => (
                <Tooltip
                  key={capability}
                  label={CAPABILITY_HINTS[capability]}
                  withArrow
                  multiline
                  w={240}
                >
                  <span className="inline-flex cursor-default items-center gap-1 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-xs font-medium text-slate-700">
                    <Check size={11} strokeWidth={2.6} className="text-emerald-600" />
                    {getCapabilityLabel(capability)}
                  </span>
                </Tooltip>
              ))}
            </div>
          ) : (
            muted("Reported after the first synchronization")
          ),
      },

      ...(lastSync
        ? [
            {
              label: "Last synchronization",
              group: "Synchronization",
              value: (
                <span className="flex flex-wrap items-center gap-2">
                  {p.isSynchronizing && (
                    <Loader2 size={13} className="motion-safe:animate-spin text-blue-500" />
                  )}
                  <ToneBadge tone={syncMeta.tone}>{syncMeta.label}</ToneBadge>
                  {lastSync.createdAt && (
                    <span className="text-xs text-slate-500">
                      <TimeAgo rfc3339={lastSync.createdAt} />
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),
      ...(p.previousSyncs.length > 0
        ? [
            {
              label: "History",
              group: "Synchronization",
              value: (
                <span className="flex flex-wrap gap-1">
                  {p.previousSyncs.slice(0, 8).map((entry, index) => {
                    const meta = getSyncStateMeta(entry.state);
                    return (
                      <Tooltip
                        key={`${entry.createdAt?.seconds ?? 0}-${index}`}
                        label={`${meta.label}${entry.createdAt ? ` · ${new Date(Number(entry.createdAt.seconds) * 1000).toLocaleString()}` : ""}`}
                        withArrow
                      >
                        <span
                          className={twMerge(
                            "h-3 w-3 rounded-full border",
                            meta.tone === "success"
                              ? "border-emerald-300 bg-emerald-400"
                              : meta.tone === "danger"
                                ? "border-red-300 bg-red-400"
                                : "border-slate-300 bg-slate-200",
                          )}
                        />
                      </Tooltip>
                    );
                  })}
                </span>
              ),
            },
          ]
        : []),
    ],
    sections: [
      {
        value: "activity",
        label: "Activity",
        icon: Activity,
        content: <IntegrationActivity item={item} />,
      },
      {
        value: "guide",
        label: "Setup guide",
        icon: BookOpen,
        content: (
          <React.Suspense
            fallback={
              <div className="m-4 h-64 motion-safe:animate-pulse rounded-xl bg-slate-100" />
            }
          >
            <IntegrationGuide item={item} />
          </React.Suspense>
        ),
      },
    ],
  };
};
