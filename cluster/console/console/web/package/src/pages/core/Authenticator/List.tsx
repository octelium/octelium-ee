import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import * as CoreP from "@/apis/corev1/corev1";
import { GetAuthenticatorSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
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
import { getType } from "./Main";

const ItemDetails = (props: { item: CoreP.Authenticator; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: CoreP.Authenticator }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>{getType(item)}</ResourceListLabel>

      <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: CoreP.Authenticator }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetAuthenticatorSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/authenticators`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalFIDO}
          to={toURLWithQry(`/core/authenticators`, {
            type: CoreP.Authenticator_Status_Type[
              CoreP.Authenticator_Status_Type.FIDO
            ],
          })}
          active={
            searchParams.get(`type`) ===
            CoreP.Authenticator_Status_Type[
              CoreP.Authenticator_Status_Type.FIDO
            ]
          }
        >
          FIDO
        </SummaryItemCount>

        <SummaryItemCount
          count={resp.totalTPM}
          to={toURLWithQry(`/core/authenticators`, {
            type: CoreP.Authenticator_Status_Type[
              CoreP.Authenticator_Status_Type.TPM
            ],
          })}
          active={
            searchParams.get(`type`) ===
            CoreP.Authenticator_Status_Type[CoreP.Authenticator_Status_Type.TPM]
          }
        >
          TPM
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalTOTP}
          to={toURLWithQry(`/core/authenticators`, {
            type: CoreP.Authenticator_Status_Type[
              CoreP.Authenticator_Status_Type.TOTP
            ],
          })}
          active={
            searchParams.get(`type`) ===
            CoreP.Authenticator_Status_Type[
              CoreP.Authenticator_Status_Type.TOTP
            ]
          }
        >
          TOTP
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalUser}>Users</SummaryItemCount>
        <SummaryItemCount count={resp.totalDevice}>Devices</SummaryItemCount>

        <SummaryItemCount
          count={resp.totalActive}
          to={toURLWithQry(`/core/authenticators`, {
            state:
              CoreP.Authenticator_Spec_State[
                CoreP.Authenticator_Spec_State.ACTIVE
              ],
          })}
          active={
            searchParams.get(`state`) ===
            CoreP.Authenticator_Spec_State[
              CoreP.Authenticator_Spec_State.ACTIVE
            ]
          }
        >
          Active
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRejected}
          to={toURLWithQry(`/core/authenticators`, {
            state:
              CoreP.Authenticator_Spec_State[
                CoreP.Authenticator_Spec_State.REJECTED
              ],
          })}
          active={
            searchParams.get(`state`) ===
            CoreP.Authenticator_Spec_State[
              CoreP.Authenticator_Spec_State.REJECTED
            ]
          }
        >
          Rejected
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPending}
          to={toURLWithQry(`/core/authenticators`, {
            state:
              CoreP.Authenticator_Spec_State[
                CoreP.Authenticator_Spec_State.PENDING
              ],
          })}
          active={
            searchParams.get(`state`) ===
            CoreP.Authenticator_Spec_State[
              CoreP.Authenticator_Spec_State.PENDING
            ]
          }
        >
          Pending
        </SummaryItemCount>

        <SummaryItemCount count={resp.totalFIDOPlatform}>
          Platform FIDO
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalFIDORoaming}>
          Roaming FIDO
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalFIDOIsHardware}>
          Hardware FIDO
        </SummaryItemCount>
        <SummaryItemCount count={resp.totalFIDOIsPasskey}>
          Passkeys
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
    queryKey: ["visibility", "core", "summary", "Authenticator"],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityCore().getAuthenticatorSummary({});

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
                  { name: "FIDO", value: d.totalFIDO },
                  { name: "TOTP", value: d.totalTOTP },
                  { name: "TPM", value: d.totalTPM },
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
        </div>
      )}

      {d.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
