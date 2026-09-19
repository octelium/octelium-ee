import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getDomain, onError } from "@/utils";
import { getClientAccess } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Button } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangle, Loader2, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { IntegrationInventoryLabels } from "./List";
import {
  getCapabilityLabel,
  getIntegrationPresentation,
  getInboundEndpoints,
  getSyncStateMeta,
} from "./utils";

const toneClass = (tone: string) =>
  tone === "success"
    ? "text-emerald-600"
    : tone === "danger"
      ? "text-red-600"
      : tone === "warning"
        ? "text-amber-600"
        : "text-blue-600";

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
      toast.success("Synchronization requested");
      invalidateResource(props.item);
      invalidateResourceList(props.item);
    },
    onError,
  });

  return (
    <Button
      type="button"
      color="ink"
      size="sm"
      leftSection={<RefreshCw size={13} />}
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

const SecretRefs = (props: { names: string[] }) => (
  <div className="flex flex-wrap gap-1.5">
    {props.names.map((name) => (
      <ResourceListLabel
        key={name}
        itemRef={ObjectReference.create({
          apiVersion: "access/v1",
          kind: "Secret",
          name,
        })}
      />
    ))}
  </div>
);

export const ItemInfo = (props: { item: AccessC.Integration }) => {
  const p = getIntegrationPresentation(props.item);
  return (
    <>
      <InfoItem title="Provider">
        <span>{p.type}</span>
      </InfoItem>
      <InfoItem title="State">
        <span className={toneClass(p.stateMeta.tone)}>{p.stateMeta.label}</span>
      </InfoItem>
    </>
  );
};

export default (props: { item: AccessC.Integration }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: AccessC.Integration;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getIntegrationPresentation(item);
  const type = item.spec?.type;
  const status = item.status;
  const endpoints = getInboundEndpoints(item, getDomain());

  return {
    actions: <SynchronizeButton item={item} />,
    groupOrder: [
      "Integration details",
      "Provider configuration",
      "Inbound endpoints",
      "Synchronization",
    ],
    items: [
      { label: "Provider", value: <Label>{p.type}</Label>, primary: true },
      {
        label: "Health",
        primary: true,
        value: (
          <span
            className={twMerge("font-semibold", toneClass(p.stateMeta.tone))}
          >
            {p.stateMeta.label}
          </span>
        ),
      },
      ...(status?.id
        ? [
            {
              label: "Integration ID",
              value: <CopyText value={status.id} />,
            },
          ]
        : []),
      ...(status?.externalTenantID
        ? [
            {
              label: "Provider tenant",
              hint: "Every Integration bound to this tenant shares the same credentials and the same inbound endpoint.",
              value: (
                <div className="flex flex-wrap items-center gap-2">
                  <CopyText value={status.externalTenantID} />
                  <ResourceListLabel
                    label="Siblings"
                    to={`/access/integrations?externalTenantID=${encodeURIComponent(status.externalTenantID)}`}
                  >
                    {status.externalTenantName || "View"}
                  </ResourceListLabel>
                </div>
              ),
            },
          ]
        : []),
      ...(p.sharedDestination
        ? [
            {
              label: `Shared destination · ${p.sharedDestination.label}`,
              value: <CopyText value={p.sharedDestination.value} />,
            },
          ]
        : []),
      {
        label: "Capabilities",
        span: "full" as const,
        value:
          p.capabilities.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {p.capabilities.map((capability) => (
                <Label key={capability} tone="info" size="sm">
                  {getCapabilityLabel(capability)}
                </Label>
              ))}
            </div>
          ) : (
            <span className="text-xs font-normal text-slate-500">
              The Cluster has not reported any capability yet
            </span>
          ),
      },
      {
        label: "Email discovery",
        value: item.spec?.identityResolution?.disableEmailDiscovery ? (
          <span className="font-semibold text-amber-600">Disabled</span>
        ) : (
          <span className="font-semibold text-emerald-600">Enabled</span>
        ),
        hint: "Email discovery matches the email that the provider reports for an external actor against the email of a Cluster User.",
      },
      ...(p.secretNames.length > 0
        ? [
            {
              label: "Secrets",
              span: "full" as const,
              value: <SecretRefs names={p.secretNames} />,
            },
          ]
        : []),
      {
        label: "Integration inventory",
        span: "full" as const,
        value: (
          <div className="flex flex-wrap gap-1">
            <IntegrationInventoryLabels item={item} />
          </div>
        ),
      },

      ...(type?.oneofKind === "slack"
        ? [
            {
              label: "Channel ID",
              group: "Provider configuration",
              value: type.slack.channelID ? (
                <CopyText value={type.slack.channelID} />
              ) : (
                <span className="text-xs font-normal text-slate-500">
                  Not set · the shared Surfaces cannot be delivered
                </span>
              ),
            },
            {
              label: "Workspace ID",
              group: "Provider configuration",
              value: type.slack.teamID ? (
                <CopyText value={type.slack.teamID} />
              ) : (
                <span className="text-xs font-normal text-slate-500">
                  Discovered by the Cluster
                </span>
              ),
            },
            ...(type.slack.mentionUserGroupID
              ? [
                  {
                    label: "Mention user group",
                    group: "Provider configuration",
                    value: <CopyText value={type.slack.mentionUserGroupID} />,
                  },
                ]
              : []),
            ...(type.slack.baseURL
              ? [
                  {
                    label: "API base URL",
                    group: "Provider configuration",
                    value: <CopyText value={type.slack.baseURL} />,
                  },
                ]
              : []),
          ]
        : []),

      ...(type?.oneofKind === "jira"
        ? [
            {
              label: "Site URL",
              group: "Provider configuration",
              span: "full" as const,
              value: <CopyText value={type.jira.url} />,
            },
            {
              label: "Account email",
              group: "Provider configuration",
              value: <CopyText value={type.jira.email} />,
            },
            {
              label: "Project key",
              group: "Provider configuration",
              value: type.jira.projectKey ? (
                <CopyText value={type.jira.projectKey} />
              ) : (
                <span className="text-xs font-normal text-slate-500">
                  Not set · the shared Surfaces cannot be delivered
                </span>
              ),
            },
            {
              label: "Issue type",
              group: "Provider configuration",
              value: type.jira.issueTypeName || "Task",
            },
            {
              label: "Approving status",
              group: "Provider configuration",
              value: type.jira.approveStatus || "Not set",
            },
            {
              label: "Rejecting status",
              group: "Provider configuration",
              value: type.jira.rejectStatus || "Not set",
            },
            ...(!type.jira.webhookSecret
              ? [
                  {
                    label: "Inbound requests",
                    group: "Provider configuration",
                    span: "full" as const,
                    value: (
                      <div className="flex items-start gap-2 rounded-lg border border-amber-100 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-700">
                        <AlertTriangle size={13} className="mt-0.5 shrink-0" />
                        No webhook secret is set, so every inbound Jira request
                        is rejected.
                      </div>
                    ),
                  },
                ]
              : []),
          ]
        : []),

      ...(type?.oneofKind === "webhook"
        ? [
            {
              label: "Delivery URL",
              group: "Provider configuration",
              span: "full" as const,
              value: <CopyText value={type.webhook.url} />,
            },
            ...(type.webhook.name
              ? [
                  {
                    label: "Name",
                    group: "Provider configuration",
                    value: <CopyText value={type.webhook.name} />,
                  },
                ]
              : []),
            ...(!type.webhook.inboundSecret
              ? [
                  {
                    label: "Inbound requests",
                    group: "Provider configuration",
                    span: "full" as const,
                    value: (
                      <div className="flex items-start gap-2 rounded-lg border border-amber-100 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-700">
                        <AlertTriangle size={13} className="mt-0.5 shrink-0" />
                        No inbound secret is set, so every inbound request is
                        rejected.
                      </div>
                    ),
                  },
                ]
              : []),
          ]
        : []),

      ...(endpoints.length > 0
        ? [
            {
              label: "Callback URLs",
              group: "Inbound endpoints",
              span: "full" as const,
              hint: "A provider that only accepts a single inbound URL for the whole tenant is configured with the endpoint of any one of the Integrations bound to that tenant.",
              value: (
                <div className="flex w-full flex-col gap-1.5">
                  {endpoints.map((endpoint) => (
                    <CopyText key={endpoint} value={endpoint} />
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(p.currentSync
        ? [
            {
              label: "Current synchronization",
              group: "Synchronization",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap items-center gap-2">
                  {p.isSynchronizing && (
                    <Loader2 size={13} className="animate-spin text-blue-500" />
                  )}
                  <span
                    className={twMerge(
                      "font-semibold",
                      toneClass(p.syncMeta.tone),
                    )}
                  >
                    {p.syncMeta.label}
                  </span>
                  {p.currentSync.createdAt && (
                    <span className="text-xs font-normal text-slate-500">
                      Started <TimeAgo rfc3339={p.currentSync.createdAt} />
                    </span>
                  )}
                  {p.currentSync.completedAt && (
                    <span className="text-xs font-normal text-slate-500">
                      Completed <TimeAgo rfc3339={p.currentSync.completedAt} />
                    </span>
                  )}
                </div>
              ),
            },
          ]
        : []),
      ...(status?.lastSuccessAt
        ? [
            {
              label: "Last successful operation",
              group: "Synchronization",
              value: <TimeAgo rfc3339={status.lastSuccessAt} />,
            },
          ]
        : []),
      ...(status?.lastFailureAt
        ? [
            {
              label: "Last failed operation",
              group: "Synchronization",
              value: <TimeAgo rfc3339={status.lastFailureAt} />,
            },
          ]
        : []),
      ...(status?.lastError
        ? [
            {
              label: "Last error",
              group: "Synchronization",
              span: "full" as const,
              value: (
                <div className="flex items-start gap-2 rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700">
                  <AlertTriangle size={13} className="mt-0.5 shrink-0" />
                  {status.lastError}
                </div>
              ),
            },
          ]
        : []),
      ...(p.allSynchronizations.length > 0
        ? [
            {
              label: "Synchronization history",
              group: "Synchronization",
              span: "full" as const,
              value: (
                <div className="w-full overflow-hidden rounded-lg border border-slate-200">
                  {p.allSynchronizations.slice(0, 6).map((entry, index) => {
                    const meta = getSyncStateMeta(entry.state);
                    return (
                      <div
                        key={`${entry.createdAt?.seconds ?? 0}-${index}`}
                        className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2 last:border-0"
                      >
                        <span
                          className={twMerge(
                            "text-xs font-semibold",
                            toneClass(meta.tone),
                          )}
                        >
                          {index === 0 && p.currentSync === entry
                            ? "Current · "
                            : ""}
                          {meta.label}
                        </span>
                        <div className="flex gap-3 text-xs font-normal text-slate-500">
                          {entry.createdAt && (
                            <span>
                              Started <TimeAgo rfc3339={entry.createdAt} />
                            </span>
                          )}
                          {entry.completedAt && (
                            <span>
                              Completed <TimeAgo rfc3339={entry.completedAt} />
                            </span>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              ),
            },
          ]
        : []),
    ],
  };
};
