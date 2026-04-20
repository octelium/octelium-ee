import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getDomain } from "@/utils";
import { getServicePrivateFQDN, getServicePublicFQDN } from "@/utils/octelium";
import { getResourceRef } from "@/utils/pb";
import { Button, Switch } from "@mantine/core";
import { ExternalLink } from "lucide-react";
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

export const MainInfo = (props: { item: CoreC.Service }): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const domain = getDomain();
  const privateFQDN = getServicePrivateFQDN(item, domain);
  const publicFQDN = getServicePublicFQDN(item, domain);

  return {
    items: [
      {
        label: "Mode",
        value: <Label>{getType(item)}</Label>,
      },

      ...(item.status?.port && item.status.port > 0
        ? [
            {
              label: "Port",
              value: (
                <span className="font-mono text-[0.75rem]">
                  {item.status.port}
                </span>
              ),
            },
          ]
        : []),

      {
        label: "Private FQDN",
        value: <CopyText value={privateFQDN} />,
        span: "full" as const,
      },

      ...(item.spec!.isPublic
        ? [
            {
              label: "Public FQDN",
              value: (
                <div className="flex items-center gap-2">
                  <CopyText value={publicFQDN} />
                  {item.spec!.mode === CoreC.Service_Spec_Mode.WEB && (
                    <a
                      href={`https://${publicFQDN}`}
                      target="_blank"
                      rel="noreferrer noopener"
                      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[0.68rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150"
                    >
                      <ExternalLink size={10} strokeWidth={2.5} />
                      Visit
                    </a>
                  )}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.spec!.isTLS || item.spec!.isPublic || item.spec!.isAnonymous
        ? [
            {
              label: "Flags",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec!.isTLS && <Label>TLS</Label>}
                  {item.spec!.isPublic && <Label>Public</Label>}
                  {item.spec!.isAnonymous && <Label>Anonymous</Label>}
                </div>
              ),
            },
          ]
        : []),

      {
        label: "Active",
        value: (
          <EditItemWrap
            label="active"
            showComponent={
              <span
                className={twMerge(
                  "text-[0.75rem] font-semibold",
                  item.spec!.isDisabled ? "text-red-500" : "text-emerald-600",
                )}
              >
                {item.spec!.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec!.isDisabled}
                onChange={(v) => {
                  item.spec!.isDisabled = !v.currentTarget.checked;
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },
    ],
  };
};
