import { Integration } from "@/apis/accessv1/accessv1";
import { GetIntegrationSummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
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
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  Ban,
  CheckCircle2,
  MessageSquare,
  SquareKanban,
  Webhook,
} from "lucide-react";
import {
  getProviderKind,
  PROVIDER_LABEL,
  ProviderMark,
  ToneBadge,
} from "../common";
import { getIntegrationPresentation, getSharedDestination } from "./utils";

export const LabelComponent = (props: { item: Integration }) => {
  const { item } = props;
  const p = getIntegrationPresentation(item);
  const provider = getProviderKind(item);
  const destination = getSharedDestination(item);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Provider">
        <ProviderMark provider={provider} size={12} />
        {provider ? PROVIDER_LABEL[provider] : "Not configured"}
      </ResourceListLabel>
      {item.spec?.isDisabled ? (
        <ToneBadge tone="neutral">Disabled</ToneBadge>
      ) : (
        <ToneBadge tone={p.stateMeta.tone}>{p.stateMeta.label}</ToneBadge>
      )}
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
      {item.status?.lastSuccessAt && (
        <ResourceListLabel label="Last success">
          <TimeAgo rfc3339={item.status.lastSuccessAt} />
        </ResourceListLabel>
      )}
      {item.status?.lastError && !item.spec?.isDisabled && (
        <ResourceListLabel label="Error">
          <AlertTriangle size={11} className="text-red-500" />
          <span className="truncate">{item.status.lastError}</span>
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetIntegrationSummaryResponse }) => (
  <div className="w-full">
    <SummaryItemCountWrap>
      <SummaryItemCount count={resp.totalNumber} to="/access/integrations">
        Total
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
    </SummaryItemCountWrap>
  </div>
);

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
