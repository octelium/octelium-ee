import { Credential_Spec_Type } from "@/apis/corev1/corev1";

import { Credential } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { GetCredentialSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import PieChart from "@/components/Charts/PieChart";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import TimeAgo from "@/components/TimeAgo";
import { toURLWithQry } from "@/pages/utils";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { Shield } from "lucide-react";
import { MdTimer } from "react-icons/md";
import { useSearchParams } from "react-router-dom";
import { match } from "ts-pattern";

export const getType = (svc: Credential): string => {
  return match(svc.spec?.type)
    .with(Credential_Spec_Type.AUTH_TOKEN, () => "Authentication Token")
    .with(Credential_Spec_Type.OAUTH2, () => "OAuth2 Client Credential")
    .with(Credential_Spec_Type.ACCESS_TOKEN, () => "Access Token")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: Credential; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Credential }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>{getType(item)}</ResourceListLabel>
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

      <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
      {item.spec?.isDisabled && (
        <ResourceListLabel>
          <span className="flex items-center text-red-400">Disabled</span>
        </ResourceListLabel>
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

export const ExtraComponent = (props: { item: Credential }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetCredentialSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/core/credentials">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalAuthenticationToken}
          to={toURLWithQry(`/core/credentials`, {
            type: Credential_Spec_Type[Credential_Spec_Type.AUTH_TOKEN],
          })}
          active={
            searchParams.get(`type`) ===
            Credential_Spec_Type[Credential_Spec_Type.AUTH_TOKEN]
          }
        >
          Authentication Tokens
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOAuth2}
          to={toURLWithQry(`/core/credentials`, {
            type: Credential_Spec_Type[Credential_Spec_Type.OAUTH2],
          })}
          active={
            searchParams.get(`type`) ===
            Credential_Spec_Type[Credential_Spec_Type.OAUTH2]
          }
        >
          OAuth2 Client Credentials
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalAccessToken}
          to={toURLWithQry(`/core/credentials`, {
            type: Credential_Spec_Type[Credential_Spec_Type.ACCESS_TOKEN],
          })}
          active={
            searchParams.get(`type`) ===
            Credential_Spec_Type[Credential_Spec_Type.ACCESS_TOKEN]
          }
        >
          Access Token
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOAuth2}
          to={toURLWithQry(`/core/credentials`, {
            isDisabled: "true",
          })}
          active={searchParams.get(`isDisabled`) === "true"}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser}>Users</SummaryItemCount>
        <SummaryItemCount count={resp.totalDisabled}>Disabled</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: {
  pieMain?: boolean;
  showNoItems?: boolean;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Credential"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getCredentialSummary(
        {}
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
            <PieChart
              data={[
                {
                  name: "Authentication Token",
                  value: d.totalAuthenticationToken,
                },
                { name: "Access Token", value: d.totalAccessToken },
                { name: "OAuth2 Client Credential", value: d.totalOAuth2 },
              ]}
            />
          )}
        </div>
      )}

      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
