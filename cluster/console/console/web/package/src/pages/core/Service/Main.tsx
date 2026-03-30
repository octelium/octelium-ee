import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { useUpdateResource } from "@/pages/utils/resource";
import { getDomain } from "@/utils";
import { getServicePrivateFQDN, getServicePublicFQDN } from "@/utils/octelium";
import { getResourceRef } from "@/utils/pb";
import { Button, Switch } from "@mantine/core";
import { FaLink } from "react-icons/fa";
import { twMerge } from "tailwind-merge";
import { getType } from "./List";

export const ItemInfo = (props: { item: CoreC.Service }) => {
  let { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <>
      <InfoItem title="Mode">
        <Label>{getType(item)}</Label>
      </InfoItem>
      {item.status?.port && item.status.port > 0 && (
        <InfoItem title="Port">
          <span>{item.status.port}</span>
        </InfoItem>
      )}
      {item.spec!.isTLS && <InfoItem title="TLS">Yes</InfoItem>}
      {item.spec!.isPublic && <InfoItem title="Public">Yes</InfoItem>}

      {item.spec!.isAnonymous && <InfoItem title="Anonymous">Yes</InfoItem>}
      <InfoItem title="Private FQDN">
        <CopyText value={getServicePrivateFQDN(item, getDomain())} />
      </InfoItem>

      {item.spec!.isPublic && (
        <div className="flex items-center">
          <span>
            <InfoItem title="Public FQDN">
              <CopyText value={getServicePublicFQDN(item, getDomain())} />
            </InfoItem>
          </span>
          {item.spec!.mode === CoreC.Service_Spec_Mode.WEB && (
            <Button
              className="ml-2 shadow-lg"
              component="a"
              target="_blank"
              href={`https://${getServicePublicFQDN(item, getDomain())}`}
              size={"xs"}
              rightSection={<FaLink />}
            >
              <span>Visit</span>
            </Button>
          )}
        </div>
      )}

      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined,
            )}
          >
            {item.spec!.isDisabled ? `No` : `Yes`}
          </span>
          <Switch
            className="ml-2"
            checked={item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>
    </>
  );
};

export const AccessLog = (props: { item: CoreC.Service }) => {
  return <AccessLogViewer serviceRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreC.Service }) => {
  const { item } = props;

  return (
    <div className="w-full">
      <div className="w-full mb-8">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};
