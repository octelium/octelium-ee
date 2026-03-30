import { IdentityProvider_Status_Type } from "@/apis/corev1/corev1";

import { IdentityProvider } from "@/apis/corev1/corev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import { GetIdentityProviderSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import PieChart from "@/components/Charts/PieChart";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { toURLWithQry } from "@/pages/utils";
import { getDomain } from "@/utils";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { match } from "ts-pattern";

export const getType = (svc: IdentityProvider): string => {
  return match(svc.spec?.type.oneofKind)
    .with("github", () => "GitHub")
    .with("oidc", () => "OpenID Connect")
    .with("saml", () => "SAML")
    .with("oidcIdentityToken", () => "OIDC Identity Token")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: IdentityProvider; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: IdentityProvider }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      {item.spec!.isDisabled && (
        <ResourceListLabel>
          <span className="text-red-400">Disabled</span>
        </ResourceListLabel>
      )}
      <ResourceListLabel label="Type">{getType(item)}</ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: IdentityProvider }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetIdentityProviderSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber}>Total</SummaryItemCount>
        <SummaryItemCount
          count={resp.totalDisabled}
          to={toURLWithQry(`/core/identityproviders`, {
            isDisabled: "true",
          })}
          active={searchParams.get(`isDisabled`) === "true"}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalGithub}
          to={toURLWithQry(`/core/identityproviders`, {
            type: IdentityProvider_Status_Type[
              IdentityProvider_Status_Type.GITHUB
            ],
          })}
          active={
            searchParams.get(`type`) ===
            IdentityProvider_Status_Type[IdentityProvider_Status_Type.GITHUB]
          }
        >
          GitHub
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOIDC}
          to={toURLWithQry(`/core/identityproviders`, {
            type: IdentityProvider_Status_Type[
              IdentityProvider_Status_Type.OIDC
            ],
          })}
          active={
            searchParams.get(`type`) ===
            IdentityProvider_Status_Type[IdentityProvider_Status_Type.OIDC]
          }
        >
          OpenID Connect
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalSAML}
          to={toURLWithQry(`/core/identityproviders`, {
            type: IdentityProvider_Status_Type[
              IdentityProvider_Status_Type.SAML
            ],
          })}
          active={
            searchParams.get(`type`) ===
            IdentityProvider_Status_Type[IdentityProvider_Status_Type.SAML]
          }
        >
          SAML 2.0
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalOIDCIdentityToken}
          to={toURLWithQry(`/core/identityproviders`, {
            type: IdentityProvider_Status_Type[
              IdentityProvider_Status_Type.OIDC_IDENTITY_TOKEN
            ],
          })}
          active={
            searchParams.get(`type`) ===
            IdentityProvider_Status_Type[
              IdentityProvider_Status_Type.OIDC_IDENTITY_TOKEN
            ]
          }
        >
          OIDC Identity Token
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: {
  pieMain?: boolean;
  showNoItems?: boolean;
}) => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "IdentityProvider"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCore().getIdentityProviderSummary({});

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
          <DoSummary resp={d} />
          {props.pieMain && (
            <PieChart
              data={[
                { name: "OpenID Connect", value: d.totalOIDC },
                { name: "SAML", value: d.totalSAML },
                { name: "GitHub", value: d.totalGithub },
                { name: "OIDC Assertion", value: d.totalOIDCIdentityToken },
              ]}
            />
          )}
        </div>
      )}

      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
