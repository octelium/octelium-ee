import {
  Certificate,
  Certificate_Spec_Mode,
  Certificate_Status_Issuance_State,
} from "@/apis/enterprisev1/enterprisev1";
import InfoItem from "@/components/InfoItem";
import TimeAgo from "@/components/TimeAgo";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Button, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { RefreshCcw } from "lucide-react";
import ClipLoader from "react-spinners/ClipLoader";
import { getIssuanceState } from "./List";

export const IssueC = (props: { item: Certificate }) => {
  const { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const mutationGenerate = useMutation({
    mutationFn: async () => {
      const { response } = await getClientEnterprise().issueCertificate({
        certificateRef: getResourceRef(item),
      });
      return response;
    },

    onSuccess: (response) => {
      invalidateResource(item);
      invalidateResourceList(item);
      close();
    },
    onError: onError,
  });

  return (
    <>
      <Button size={`xs`} onClick={open}>
        Issue/Re-issue this Certificate
      </Button>
      <Modal opened={opened} onClose={close} size={"xl"} centered>
        <div className="w-full">
          <div className="flex items-center justify-center my-8">
            <Button
              onClick={() => {
                mutationGenerate.mutate();
              }}
              loading={mutationGenerate.isPending}
              leftSection={<RefreshCcw />}
            >
              Issue/Re-issue this Certificate
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export const ItemInfo = (props: { item: Certificate }) => {
  const { item } = props;
  const issuance = item.status?.issuance;
  const lastSuccess = item.status?.lastIssuances
    .filter((x) => x.state === Certificate_Status_Issuance_State.SUCCESS)
    .at(0);

  return (
    <>
      {issuance && (
        <>
          {(issuance.state === Certificate_Status_Issuance_State.ISSUING ||
            issuance.state ===
              Certificate_Status_Issuance_State.ISSUANCE_REQUESTED) && (
            <InfoItem title="Issuance State">
              <ClipLoader
                loading={true}
                size={14}
                color="black"
                className="mr-1"
              />
              <span>{getIssuanceState(issuance.state)}</span>
            </InfoItem>
          )}
        </>
      )}

      {lastSuccess && (
        <>
          <InfoItem title="Last Issuance">
            <TimeAgo rfc3339={lastSuccess.createdAt} />
          </InfoItem>

          <InfoItem title="Expiration">
            <TimeAgo rfc3339={lastSuccess.expiresAt} />
          </InfoItem>
        </>
      )}

      {(issuance?.state === Certificate_Status_Issuance_State.FAILED ||
        issuance?.state === Certificate_Status_Issuance_State.SUCCESS) &&
        item.spec?.mode === Certificate_Spec_Mode.MANAGED && (
          <InfoItem title="Issue">
            <IssueC item={item} />
          </InfoItem>
        )}
    </>
  );
};

export default (props: { item: Certificate }) => {
  const { item } = props;
  return (
    <div className="w-full">
      <div className="w-full mb-8">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};
