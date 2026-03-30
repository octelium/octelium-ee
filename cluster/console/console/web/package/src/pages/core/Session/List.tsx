import { Session_Spec_State, Session_Status_Type } from "@/apis/corev1/corev1";

import { Session } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import TimeAgo from "@/components/TimeAgo";
import { getDomain } from "@/utils";
import { MdTimer } from "react-icons/md";
import { match } from "ts-pattern";

import { GetSessionSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import PieChart from "@/components/Charts/PieChart";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { toURLWithQry } from "@/pages/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Shield } from "lucide-react";
import { BsFillTerminalFill } from "react-icons/bs";
import { IoBrowsers } from "react-icons/io5";
import { useSearchParams } from "react-router-dom";

export const getType = (svc: Session) => {
  return match(svc.status?.type)
    .with(Session_Status_Type.CLIENT, () => (
      <>
        <BsFillTerminalFill size={14} />
        <span className="ml-2">Client</span>
      </>
    ))
    .with(Session_Status_Type.CLIENTLESS, () => (
      <>
        {svc.status?.isBrowser ? (
          <>
            <IoBrowsers size={14} />
            <span className="ml-2">Browser</span>
          </>
        ) : (
          <span>Clientless</span>
        )}
      </>
    ))

    .otherwise(() => <></>);
};

const getState = (svc: Session) => {
  return match(svc.spec?.state)
    .with(Session_Spec_State.ACTIVE, () => (
      <span className="text-green-400">Active</span>
    ))
    .with(Session_Spec_State.REJECTED, () => (
      <span className="text-red-400">Rejected</span>
    ))
    .with(Session_Spec_State.PENDING, () => (
      <span className="text-yellow-500">Pending</span>
    ))
    .otherwise(() => "");
};

const ItemDetails = (props: { item: Session; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Session }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      {item.spec?.expiresAt && (
        <ResourceListLabel>
          <span className="mr-1">
            <MdTimer size={14} />
          </span>
          <span>
            Expires <TimeAgo rfc3339={item.spec.expiresAt} />
          </span>
        </ResourceListLabel>
      )}
      {item.status?.authentication &&
        item.status.lastAuthentications &&
        item.status.lastAuthentications.length > 0 &&
        item.status.authentication.setAt && (
          <ResourceListLabel label="Last Authentication">
            <TimeAgo rfc3339={item.status.authentication.setAt} />
          </ResourceListLabel>
        )}
      <ResourceListLabel>{getType(item)}</ResourceListLabel>
      <ResourceListLabel>{getState(item)}</ResourceListLabel>
      {item.status!.isConnected && (
        <ResourceListLabel>
          <div className="flex items-center">
            <div className="flex relative w-[10px] h-[10px]">
              <div className="bg-green-500 rounded-full w-[10px] h-[10px] animate-ping absolute"></div>
              <div className="bg-green-500 rounded-full w-[10px] h-[10px]"></div>
            </div>
            <span className="ml-1">Connected</span>
          </div>
        </ResourceListLabel>
      )}
      <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
      {item.status?.deviceRef?.name && (
        <ResourceListLabel itemRef={item.status!.deviceRef}></ResourceListLabel>
      )}

      {item.spec?.authorization &&
        item.spec?.authorization.policies.length > 0 && (
          <ResourceListLabel>
            <Shield size={14} className="mr-1" />
            {item.spec.authorization.policies.length} Policies
          </ResourceListLabel>
        )}
      {item.spec?.authorization &&
        item.spec?.authorization.inlinePolicies.length > 0 && (
          <ResourceListLabel>
            <Shield size={14} className="mr-1" />
            {item.spec.authorization.inlinePolicies.length} Inline Policies
          </ResourceListLabel>
        )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Session }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetSessionSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/core/sessions">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalClient}
          to={toURLWithQry(`/core/sessions`, {
            type: Session_Status_Type[Session_Status_Type.CLIENT],
          })}
          active={
            searchParams.get(`type`) ===
            Session_Status_Type[Session_Status_Type.CLIENT]
          }
        >
          Clients
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalClientless}
          to={toURLWithQry(`/core/sessions`, {
            type: Session_Status_Type[Session_Status_Type.CLIENTLESS],
          })}
          active={
            searchParams.get(`type`) ===
            Session_Status_Type[Session_Status_Type.CLIENTLESS]
          }
        >
          Clientless
        </SummaryItemCount>

        <SummaryItemCount
          count={resp.totalClientlessBrowser}
          to={toURLWithQry(`/core/sessions`, {
            isBrowser: "true",
          })}
          active={searchParams.get(`isBrowser`) === "true"}
        >
          Browsers
        </SummaryItemCount>

        <SummaryItemCount
          count={resp.totalActive}
          to={toURLWithQry(`/core/sessions`, {
            state: Session_Spec_State[Session_Spec_State.ACTIVE],
          })}
          active={
            searchParams.get(`state`) ===
            Session_Spec_State[Session_Spec_State.ACTIVE]
          }
        >
          Active
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRejected}
          to={toURLWithQry(`/core/sessions`, {
            state: Session_Spec_State[Session_Spec_State.REJECTED],
          })}
          active={
            searchParams.get(`state`) ===
            Session_Spec_State[Session_Spec_State.REJECTED]
          }
        >
          Rejected
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPending}
          to={toURLWithQry(`/core/sessions`, {
            state: Session_Spec_State[Session_Spec_State.PENDING],
          })}
          active={
            searchParams.get(`state`) ===
            Session_Spec_State[Session_Spec_State.PENDING]
          }
        >
          Pending
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalConnected}
          to={toURLWithQry(`/core/sessions`, {
            isConnected: "true",
          })}
          active={searchParams.get(`isConnected`) === "true"}
        >
          Connected
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser}>Users</SummaryItemCount>
        <SummaryItemCount count={resp.totalDevice}>Devices</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: {
  pieMain?: boolean;
  showNoItems?: boolean;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Session"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getSessionSummary(
        {},
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const d = qry.data;

  return (
    <div>
      {d.totalNumber > 0 && (
        <div>
          <DoSummary resp={qry.data} />
          {props.pieMain && (
            <div className="w-full grid grid-cols-2 gap-1">
              <PieChart
                data={[
                  { name: "Client", value: d.totalClient },
                  { name: "Clientless", value: d.totalClientless },
                ]}
              />
              <PieChart
                data={[
                  { name: "Active", value: d.totalActive },
                  { name: "Rejected", value: d.totalRejected },
                  { name: "Pending", value: d.totalPending },
                ]}
              />
            </div>
          )}

          {props.pieMain && (
            <div className="w-full">
              <PieChart
                data={[
                  { name: "Browser", value: d.totalClientlessBrowser },
                  { name: "OAuth2", value: d.totalClientlessOAuth2 },
                  { name: "SDK", value: d.totalClientlessSDK },
                ]}
              />
            </div>
          )}
        </div>
      )}
      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
