import {
  Certificate,
  Certificate_Status_Issuance_State,
} from "@/apis/enterprisev1/enterprisev1";
import { ResourceListLabel } from "@/components/ResourceList";

import TimeAgo from "@/components/TimeAgo";
import { getDomain } from "@/utils";
import ClipLoader from "react-spinners/ClipLoader";
import { match } from "ts-pattern";

const ItemDetails = (props: { item: Certificate; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const getIssuanceState = (arg: Certificate_Status_Issuance_State) => {
  return match(arg)
    .with(Certificate_Status_Issuance_State.FAILED, () => "Failed")
    .with(Certificate_Status_Issuance_State.ISSUING, () => "Issuing")
    .with(
      Certificate_Status_Issuance_State.ISSUANCE_REQUESTED,
      () => "Issuance Requested",
    )
    .with(Certificate_Status_Issuance_State.SUCCESS, () => "Success")
    .otherwise(() => "");
};

export const LabelComponent = (props: { item: Certificate }) => {
  const { item } = props;
  const issuance = item.status?.issuance;
  const lastSuccess = item.status?.lastIssuances
    .filter((x) => x.state === Certificate_Status_Issuance_State.SUCCESS)
    .at(0);
  return (
    <div className="w-full mt-1 flex flex-row">
      {issuance && (
        <>
          {(issuance.state === Certificate_Status_Issuance_State.ISSUING ||
            issuance.state ===
              Certificate_Status_Issuance_State.ISSUANCE_REQUESTED) && (
            <ResourceListLabel>
              <ClipLoader
                loading={true}
                size={14}
                color="white"
                className="mr-1"
              />
              <span>{getIssuanceState(issuance.state)}</span>
            </ResourceListLabel>
          )}
        </>
      )}

      {issuance?.state === Certificate_Status_Issuance_State.SUCCESS && (
        <>
          <ResourceListLabel label="Current Issuance">
            <TimeAgo rfc3339={issuance.createdAt} />
          </ResourceListLabel>

          <ResourceListLabel label="Expiration">
            <TimeAgo rfc3339={issuance.expiresAt} />
          </ResourceListLabel>
        </>
      )}
      {item.status!.successfulIssuances > 0 && (
        <ResourceListLabel label="Successful Issuances">
          <span>{item.status!.successfulIssuances}</span>
        </ResourceListLabel>
      )}
    </div>
  );
};

export const ExtraComponent = (props: { item: Certificate }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};
