import {
  IntegrationBinding,
  Policy_Spec_Rule_Surface_InteractionMode,
} from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { GetIntegrationBindingSummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
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
  Bell,
  CheckCircle2,
  CircleSlash,
  Clock3,
  Inbox,
  ListChecks,
  Plug,
  RefreshCw,
  Send,
  UserCheck,
  UserRound,
  Users,
} from "lucide-react";
import { getAudienceLabel } from "../Integration/utils";
import { getPurposeLabel, getStateMeta, isFailing, isOutOfDate } from "./utils";

export const getIntegrationRef = (
  item: IntegrationBinding,
): ObjectReference | undefined =>
  item.status?.integrationRef
    ? ObjectReference.create({
        ...item.status.integrationRef,
        apiVersion: item.status.integrationRef.apiVersion || "access/v1",
        kind: item.status.integrationRef.kind || "Integration",
      })
    : undefined;

export const getRequestRef = (
  item: IntegrationBinding,
): ObjectReference | undefined =>
  item.status?.requestRef
    ? ObjectReference.create({
        ...item.status.requestRef,
        apiVersion: item.status.requestRef.apiVersion || "access/v1",
        kind: item.status.requestRef.kind || "Request",
      })
    : undefined;

export const getUserRef = (
  item: IntegrationBinding,
): ObjectReference | undefined =>
  item.status?.userRef
    ? ObjectReference.create({
        ...item.status.userRef,
        apiVersion: item.status.userRef.apiVersion || "core/v1",
        kind: item.status.userRef.kind || "User",
      })
    : undefined;

export const LabelComponent = (props: { item: IntegrationBinding }) => {
  const { item } = props;
  const meta = getStateMeta(item.status?.state);
  const integrationRef = getIntegrationRef(item);
  const requestRef = getRequestRef(item);
  const userRef = getUserRef(item);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span className={meta.className}>{meta.label}</span>
      </ResourceListLabel>
      <ResourceListLabel label="Purpose">
        {getPurposeLabel(item.status?.purpose)}
      </ResourceListLabel>
      <ResourceListLabel label="Audience">
        {getAudienceLabel(item.status?.audience)}
      </ResourceListLabel>
      {(integrationRef?.name || integrationRef?.uid) && (
        <ResourceListLabel itemRef={integrationRef} />
      )}
      {(requestRef?.name || requestRef?.uid) && (
        <ResourceListLabel itemRef={requestRef} />
      )}
      {(userRef?.name || userRef?.uid) && (
        <ResourceListLabel label="Delivered to" itemRef={userRef} />
      )}
      {item.status?.stepName && (
        <ResourceListLabel label="Step">
          {item.status.stepName}
        </ResourceListLabel>
      )}
      {item.status?.interactionMode ===
        Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE && (
        <ResourceListLabel>Interactive</ResourceListLabel>
      )}
      {isOutOfDate(item) && <ResourceListLabel>Out of date</ResourceListLabel>}
      {isFailing(item) && (
        <ResourceListLabel label="Failed attempts">
          {item.status?.attempts}
        </ResourceListLabel>
      )}
      {item.status?.lastSuccessAt && (
        <ResourceListLabel label="Last delivery">
          <TimeAgo rfc3339={item.status.lastSuccessAt} />
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: IntegrationBinding }) => {
  return <div></div>;
};

const DoSummary = ({ resp }: { resp: GetIntegrationBindingSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount
          count={resp.totalNumber}
          to="/access/integrationbindings"
        >
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPending}
          to="/access/integrationbindings?state=PENDING"
          icon={Clock3}
        >
          Pending
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalReady}
          to="/access/integrationbindings?state=READY"
          icon={CheckCircle2}
        >
          Ready
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDegraded}
          to="/access/integrationbindings?state=DEGRADED"
          icon={AlertTriangle}
        >
          Degraded
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalClosed}
          to="/access/integrationbindings?state=CLOSED"
          icon={CircleSlash}
        >
          Closed
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOutOfDate}
          to="/access/integrationbindings?isOutOfDate=true"
          icon={RefreshCw}
        >
          Out of date
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalFailing}
          to="/access/integrationbindings?isFailing=true"
          icon={AlertTriangle}
        >
          Failing
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalReviewSurface}
          to="/access/integrationbindings?purpose=REVIEW_SURFACE"
          icon={ListChecks}
        >
          Review surfaces
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalNotification}
          to="/access/integrationbindings?purpose=NOTIFICATION"
          icon={Bell}
        >
          Notifications
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalInteractive}
          to="/access/integrationbindings?interactionMode=INTERACTIVE"
          icon={UserCheck}
        >
          Interactive
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalShared}
          to="/access/integrationbindings?audience=SHARED"
          icon={Send}
        >
          Shared
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalReviewers}
          to="/access/integrationbindings?audience=REVIEWERS"
          icon={Users}
        >
          Reviewers
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRequester}
          to="/access/integrationbindings?audience=REQUESTER"
          icon={UserRound}
        >
          Requesters
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSubject}
          to="/access/integrationbindings?audience=SUBJECT"
          icon={UserRound}
        >
          Subjects
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalIntegration} icon={Plug}>
          Integrations
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalRequest} icon={Inbox}>
          Requests
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "IntegrationBinding"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccess().getIntegrationBindingSummary({});
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
