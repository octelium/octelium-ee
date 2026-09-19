import * as AccessP from "@/apis/accessv1/accessv1";
import { ResourceStatusTone } from "@/pages/utils/types";
import { match } from "ts-pattern";

type SecretRef =
  | {
      type: { oneofKind: "fromSecret"; fromSecret: string } | { oneofKind: undefined };
    }
  | undefined;

const fromSecret = (arg: SecretRef): string | undefined =>
  arg?.type.oneofKind === "fromSecret" && arg.type.fromSecret
    ? arg.type.fromSecret
    : undefined;

export const getType = (item: AccessP.Integration): string =>
  match(item.spec?.type?.oneofKind)
    .with("slack", () => "Slack")
    .with("jira", () => "Jira")
    .with("webhook", () => "Webhook")
    .otherwise(() => "Not configured");

export const getTypeFilterValue = (
  item: AccessP.Integration,
): string | undefined =>
  match(item.spec?.type?.oneofKind)
    .with("slack", () => "SLACK")
    .with("jira", () => "JIRA")
    .with("webhook", () => "WEBHOOK")
    .otherwise(() => undefined);

export const getProviderSegment = (
  item: AccessP.Integration,
): string | undefined =>
  match(item.spec?.type?.oneofKind)
    .with("slack", () => "slack")
    .with("jira", () => "jira")
    .with("webhook", () => "webhook")
    .otherwise(() => undefined);

export const getStateMeta = (
  state?: AccessP.Integration_Status_State,
): { label: string; tone: ResourceStatusTone } =>
  match(state)
    .with(AccessP.Integration_Status_State.READY, () => ({
      label: "Ready",
      tone: "success" as const,
    }))
    .with(AccessP.Integration_Status_State.DEGRADED, () => ({
      label: "Degraded",
      tone: "warning" as const,
    }))
    .with(AccessP.Integration_Status_State.ERROR, () => ({
      label: "Error",
      tone: "danger" as const,
    }))
    .otherwise(() => ({ label: "Not checked", tone: "neutral" as const }));

export const getSyncStateMeta = (
  state?: AccessP.Integration_Status_Synchronization_State,
): { label: string; tone: ResourceStatusTone } =>
  match(state)
    .with(
      AccessP.Integration_Status_Synchronization_State.SYNC_REQUESTED,
      () => ({ label: "Sync requested", tone: "warning" as const }),
    )
    .with(AccessP.Integration_Status_Synchronization_State.SYNCING, () => ({
      label: "Synchronizing",
      tone: "info" as const,
    }))
    .with(AccessP.Integration_Status_Synchronization_State.SUCCESS, () => ({
      label: "Successful",
      tone: "success" as const,
    }))
    .with(AccessP.Integration_Status_Synchronization_State.FAILED, () => ({
      label: "Failed",
      tone: "danger" as const,
    }))
    .otherwise(() => ({ label: "Unknown", tone: "neutral" as const }));

export const getCapabilityLabel = (
  capability: AccessP.Integration_Status_Capability,
): string =>
  match(capability)
    .with(
      AccessP.Integration_Status_Capability.NOTIFICATION,
      () => "Shared notification",
    )
    .with(
      AccessP.Integration_Status_Capability.DIRECT_USER_DELIVERY,
      () => "Direct user delivery",
    )
    .with(
      AccessP.Integration_Status_Capability.INTERACTIVE_REVIEW,
      () => "Interactive review",
    )
    .with(
      AccessP.Integration_Status_Capability.REQUEST_CREATION,
      () => "Request creation",
    )
    .with(
      AccessP.Integration_Status_Capability.IDENTITY_RESOLUTION,
      () => "Identity resolution",
    )
    .with(
      AccessP.Integration_Status_Capability.PRESENTATION_UPDATE,
      () => "Presentation update",
    )
    .otherwise(() => "Unknown");

export const hasCapability = (
  item: AccessP.Integration,
  capability: AccessP.Integration_Status_Capability,
): boolean => (item.status?.capabilities ?? []).includes(capability);

export const getAudienceLabel = (
  audience?: AccessP.Policy_Spec_Rule_Surface_Destination_Audience,
): string =>
  match(audience)
    .with(
      AccessP.Policy_Spec_Rule_Surface_Destination_Audience.SHARED,
      () => "Shared destination",
    )
    .with(
      AccessP.Policy_Spec_Rule_Surface_Destination_Audience.REVIEWERS,
      () => "Reviewers",
    )
    .with(
      AccessP.Policy_Spec_Rule_Surface_Destination_Audience.REQUESTER,
      () => "Requester",
    )
    .with(
      AccessP.Policy_Spec_Rule_Surface_Destination_Audience.SUBJECT,
      () => "Subject",
    )
    .otherwise(() => "Unset");

export const getInteractionModeLabel = (
  mode?: AccessP.Policy_Spec_Rule_Surface_InteractionMode,
): string =>
  match(mode)
    .with(
      AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE,
      () => "Interactive",
    )
    .otherwise(() => "Deep link only");

export const getSharedDestination = (
  item: AccessP.Integration,
): { label: string; value: string } | undefined => {
  const type = item.spec?.type;
  if (type?.oneofKind === "slack" && type.slack.channelID)
    return { label: "Channel ID", value: type.slack.channelID };
  if (type?.oneofKind === "jira" && type.jira.projectKey)
    return { label: "Project key", value: type.jira.projectKey };
  if (type?.oneofKind === "webhook" && type.webhook.url)
    return { label: "Delivery URL", value: type.webhook.url };
  return undefined;
};

export const integrationSecretNames = (item: AccessP.Integration): string[] => {
  const type = item.spec?.type;
  const names =
    type?.oneofKind === "slack"
      ? [fromSecret(type.slack.botToken), fromSecret(type.slack.signingSecret)]
      : type?.oneofKind === "jira"
        ? [fromSecret(type.jira.apiToken), fromSecret(type.jira.webhookSecret)]
        : type?.oneofKind === "webhook"
          ? [
              fromSecret(type.webhook.signingSecret),
              fromSecret(type.webhook.inboundSecret),
            ]
          : [];

  return [...new Set(names.filter((name): name is string => !!name))];
};

export const getInboundEndpoints = (
  item: AccessP.Integration,
  domain: string,
): string[] => {
  const id = item.status?.id;
  const segment = getProviderSegment(item);
  if (!id || !segment) return [];
  const base = `https://public.octelium.${domain}/integration/v1/callback/${id}/${segment}`;
  return match(segment)
    .with("slack", () => [
      `${base}/interactions`,
      `${base}/commands`,
      `${base}/events`,
    ])
    .with("jira", () => [`${base}/webhook`])
    .otherwise(() => [base]);
};

const syncTime = (item: AccessP.Integration_Status_Synchronization) => {
  const time = item.completedAt ?? item.createdAt;
  return time ? Number(time.seconds) : 0;
};

export const getIntegrationPresentation = (item: AccessP.Integration) => {
  const currentSync = item.status?.synchronization;
  const previousSyncs = item.status?.lastSynchronizations ?? [];
  const allSynchronizations = [
    ...(currentSync ? [currentSync] : []),
    ...previousSyncs,
  ];
  const lastSuccessfulSync = allSynchronizations
    .filter(
      (entry) =>
        entry.state ===
        AccessP.Integration_Status_Synchronization_State.SUCCESS,
    )
    .sort((a, b) => syncTime(b) - syncTime(a))[0];
  const isSynchronizing =
    currentSync?.state ===
      AccessP.Integration_Status_Synchronization_State.SYNC_REQUESTED ||
    currentSync?.state ===
      AccessP.Integration_Status_Synchronization_State.SYNCING;

  return {
    type: getType(item),
    isConfigured: !!item.spec?.type?.oneofKind,
    capabilities: item.status?.capabilities ?? [],
    stateMeta: getStateMeta(item.status?.state),
    syncMeta: getSyncStateMeta(currentSync?.state),
    sharedDestination: getSharedDestination(item),
    secretNames: integrationSecretNames(item),
    currentSync,
    previousSyncs,
    allSynchronizations,
    lastSuccessfulSync,
    isSynchronizing,
  };
};
