import {
  Integration,
  Integration_Status_Synchronization_State,
} from "@/apis/accessv1/accessv1";
import {
  GetIntegrationSummaryResponse,
  ListIntegrationBindingOptions,
  ListIntegrationIdentityOptions,
} from "@/apis/visibilityv1/access/vaccessv1";
import { CommonListOptions } from "@/apis/visibilityv1/meta/vmetav1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import TimeAgo from "@/components/TimeAgo";
import { getClientVisibilityAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  Ban,
  Building2,
  CheckCircle2,
  Link2,
  Loader2,
  MessageSquare,
  RefreshCw,
  Send,
  SquareKanban,
  UserCheck,
  UserRound,
  Webhook,
} from "lucide-react";
import { getIntegrationPresentation, getSharedDestination } from "./utils";

export const IntegrationInventoryLabels = (props: { item: Integration }) => {
  const common = CommonListOptions.create({ itemsPerPage: 1 });
  const identity = props.item.metadata?.uid || props.item.metadata?.name;
  const p = getIntegrationPresentation(props.item);
  const identities = useQuery({
    queryKey: ["access.integrationIdentity.count", identity],
    queryFn: async () =>
      (
        await getClientVisibilityAccess().listIntegrationIdentity(
          ListIntegrationIdentityOptions.create({
            common,
            integrationRef: getResourceRef(props.item),
          }),
        )
      ).response,
    enabled: !!identity,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchInterval: p.isSynchronizing ? 5_000 : false,
  });
  const bindings = useQuery({
    queryKey: ["access.integrationBinding.count", identity],
    queryFn: async () =>
      (
        await getClientVisibilityAccess().listIntegrationBinding(
          ListIntegrationBindingOptions.create({
            common,
            integrationRef: getResourceRef(props.item),
          }),
        )
      ).response,
    enabled: !!identity,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchInterval: p.isSynchronizing ? 5_000 : false,
  });

  return (
    <>
      <ResourceListLabel label="Identities">
        {identities.isError
          ? "Unavailable"
          : (identities.data?.listResponseMeta?.totalCount?.toLocaleString() ??
            "…")}
      </ResourceListLabel>
      <ResourceListLabel label="Presentations">
        {bindings.isError
          ? "Unavailable"
          : (bindings.data?.listResponseMeta?.totalCount?.toLocaleString() ??
            "…")}
      </ResourceListLabel>
    </>
  );
};

export const LabelComponent = (props: { item: Integration }) => {
  const { item } = props;
  const p = getIntegrationPresentation(item);
  const destination = getSharedDestination(item);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Provider">{p.type}</ResourceListLabel>
      {item.spec?.isDisabled && <ResourceListLabel>Disabled</ResourceListLabel>}
      <ResourceListLabel label="State">{p.stateMeta.label}</ResourceListLabel>
      {destination && (
        <ResourceListLabel label={destination.label}>
          {destination.value}
        </ResourceListLabel>
      )}
      {item.status?.externalTenantID && (
        <ResourceListLabel label="Tenant">
          {item.status.externalTenantName || item.status.externalTenantID}
        </ResourceListLabel>
      )}
      {p.currentSync && (
        <ResourceListLabel label="Synchronization">
          {p.isSynchronizing ? (
            <Loader2 size={11} className="animate-spin text-blue-500" />
          ) : p.currentSync.state ===
            Integration_Status_Synchronization_State.FAILED ? (
            <AlertTriangle size={11} className="text-red-500" />
          ) : p.currentSync.state ===
            Integration_Status_Synchronization_State.SUCCESS ? (
            <CheckCircle2 size={11} className="text-emerald-500" />
          ) : null}
          {p.syncMeta.label}
        </ResourceListLabel>
      )}
      {p.lastSuccessfulSync?.completedAt && (
        <ResourceListLabel label="Last synchronized">
          <TimeAgo rfc3339={p.lastSuccessfulSync.completedAt} />
        </ResourceListLabel>
      )}
      <IntegrationInventoryLabels item={item} />
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Integration }) => {
  return <div></div>;
};

const DoSummary = ({ resp }: { resp: GetIntegrationSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/access/integrations">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDisabled}
          to="/access/integrations?isDisabled=true"
          icon={Ban}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSlack}
          to="/access/integrations?type=SLACK"
          icon={MessageSquare}
        >
          Slack
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalJira}
          to="/access/integrations?type=JIRA"
          icon={SquareKanban}
        >
          Jira
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWebhook}
          to="/access/integrations?type=WEBHOOK"
          icon={Webhook}
        >
          Webhook
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalReady}
          to="/access/integrations?state=READY"
          icon={CheckCircle2}
        >
          Ready
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDegraded}
          to="/access/integrations?state=DEGRADED"
          icon={AlertTriangle}
        >
          Degraded
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalError}
          to="/access/integrations?state=ERROR"
          icon={AlertTriangle}
        >
          Failing
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSynchronizing}
          to="/access/integrations?synchronizationState=SYNCING"
          icon={RefreshCw}
        >
          Synchronizing
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSynchronizationSuccess}
          to="/access/integrations?synchronizationState=SUCCESS"
          icon={CheckCircle2}
        >
          Synchronized
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSynchronizationFailed}
          to="/access/integrations?synchronizationState=FAILED"
          icon={AlertTriangle}
        >
          Failed synchronization
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalExternalTenant} icon={Building2}>
          Provider tenants
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalNotification}
          to="/access/integrations?capability=NOTIFICATION"
          icon={Send}
        >
          Shared notification
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDirectUserDelivery}
          to="/access/integrations?capability=DIRECT_USER_DELIVERY"
          icon={UserRound}
        >
          Direct user delivery
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalInteractiveReview}
          to="/access/integrations?capability=INTERACTIVE_REVIEW"
          icon={UserCheck}
        >
          Interactive review
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRequestCreation}
          to="/access/integrations?capability=REQUEST_CREATION"
          icon={Send}
        >
          Request creation
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalIdentityResolution}
          to="/access/integrations?capability=IDENTITY_RESOLUTION"
          icon={Link2}
        >
          Identity resolution
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPresentationUpdate}
          to="/access/integrations?capability=PRESENTATION_UPDATE"
          icon={RefreshCw}
        >
          Presentation update
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Integration"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccess().getIntegrationSummary({});
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return (
    <div>
      {qry.data.totalNumber > 0 && <DoSummary resp={qry.data} />}
      {qry.data.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
