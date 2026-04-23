import * as CoreP from "@/apis/corev1/corev1";
import InfoItem from "@/components/InfoItem";
import { useUpdateResource } from "@/pages/utils/resource";
import { Select } from "@mantine/core";
import { match } from "ts-pattern";

import AccessLogViewer from "@/components/AccessLogViewer";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getResourceRef } from "@/utils/pb";
import { FaAndroid, FaApple, FaLinux, FaWindows } from "react-icons/fa";
import { twMerge } from "tailwind-merge";

export const getOSIcon = (item: CoreP.Device) => {
  return match(item.status!.osType)
    .with(CoreP.Device_Status_OSType.LINUX, () => <FaLinux />)
    .with(CoreP.Device_Status_OSType.ANDROID, () => <FaAndroid />)
    .with(CoreP.Device_Status_OSType.IOS, () => <FaApple />)
    .with(CoreP.Device_Status_OSType.MAC, () => <FaApple />)
    .with(CoreP.Device_Status_OSType.WINDOWS, () => <FaWindows />)
    .otherwise(() => undefined);
};

export const getOSStr = (item: CoreP.Device) => {
  return getOSTypeStr(item.status!.osType);
};

export const getOSTypeStr = (osType: CoreP.Device_Status_OSType) => {
  return match(osType)
    .with(CoreP.Device_Status_OSType.LINUX, () => "Linux")
    .with(CoreP.Device_Status_OSType.ANDROID, () => "Android")
    .with(CoreP.Device_Status_OSType.IOS, () => "IOS")
    .with(CoreP.Device_Status_OSType.MAC, () => "Mac OS")
    .with(CoreP.Device_Status_OSType.WINDOWS, () => "Windows")
    .with(CoreP.Device_Status_OSType.OS_TYPE_UNKNOWN, () => "Unknown OS")
    .otherwise(() => "");
};

export const ItemInfo = (props: { item: CoreP.Device }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="OS">
        <Label>
          <span className="flex items-center">
            <span className="mr-1">{getOSIcon(item)}</span>
            <span>{getOSStr(item)}</span>
          </span>
        </Label>
      </InfoItem>
      {item.status!.id && <InfoItem title="ID">{item.status!.id}</InfoItem>}
      {item.status!.hostname && (
        <InfoItem title="Hostname">{item.status!.hostname}</InfoItem>
      )}
      {item.status!.serialNumber && (
        <InfoItem title="Serial Number">{item.status!.serialNumber}</InfoItem>
      )}
      <InfoItem title="State">
        <EditItemWrap
          showComponent={
            <span
              className={twMerge(
                match(item.spec!.state)
                  .with(CoreP.Device_Spec_State.REJECTED, () => "text-red-600")
                  .with(
                    CoreP.Device_Spec_State.PENDING,
                    () => "text-yellow-600",
                  )
                  .otherwise(() => undefined),
              )}
            >
              {match(item.spec!.state)
                .with(CoreP.Device_Spec_State.ACTIVE, () => "Active")
                .with(CoreP.Device_Spec_State.REJECTED, () => "Rejected")
                .with(CoreP.Device_Spec_State.PENDING, () => "Pending")
                .otherwise(() => "")}
            </span>
          }
          editComponent={
            <Select
              data={[
                {
                  label: "Active",
                  value:
                    CoreP.Device_Spec_State[CoreP.Device_Spec_State.ACTIVE],
                },
                {
                  label: "Pending",
                  value:
                    CoreP.Device_Spec_State[CoreP.Device_Spec_State.PENDING],
                },
                {
                  label: "Rejected",
                  value:
                    CoreP.Device_Spec_State[CoreP.Device_Spec_State.REJECTED],
                },
              ]}
              value={CoreP.Device_Spec_State[item.spec!.state]}
              onChange={(v) => {
                if (!v) {
                  return;
                }
                item.spec!.state = CoreP.Device_Spec_State[v as "ACTIVE"];
                mutationUpdate.mutate(item);
              }}
            />
          }
        />
      </InfoItem>
      {item.status!.macAddresses && item.status!.macAddresses.length > 0 && (
        <InfoItem title="MAC Addresses">
          {item.status!.macAddresses.map((x) => (
            <Label>{x}</Label>
          ))}
        </InfoItem>
      )}
    </>
  );
};

export const AccessLog = (props: { item: CoreP.Device }) => {
  return <AccessLogViewer deviceRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreP.Device }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  return (
    <div className="w-full">
      <div className="w-full">
        <ItemInfo item={item} />
      </div>
    </div>
  );
};

export const MainInfo = (props: { item: CoreP.Device }): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return {
    items: [
      {
        label: "User",
        value: <ResourceListLabel itemRef={item.status!.userRef} />,
      },
      {
        label: "OS",
        value: (
          <Label>
            <span className="flex items-center gap-1">
              {getOSIcon(item)}
              <span>{getOSStr(item)}</span>
            </span>
          </Label>
        ),
      },

      {
        label: "State",
        value: (
          <EditItemWrap
            label="state"
            showComponent={
              <span
                className={twMerge(
                  "text-sm font-semibold",
                  match(item.spec!.state)
                    .with(
                      CoreP.Device_Spec_State.ACTIVE,
                      () => "text-emerald-600",
                    )
                    .with(
                      CoreP.Device_Spec_State.REJECTED,
                      () => "text-red-500",
                    )
                    .with(
                      CoreP.Device_Spec_State.PENDING,
                      () => "text-amber-500",
                    )
                    .otherwise(() => "text-slate-600"),
                )}
              >
                {match(item.spec!.state)
                  .with(CoreP.Device_Spec_State.ACTIVE, () => "Active")
                  .with(CoreP.Device_Spec_State.REJECTED, () => "Rejected")
                  .with(CoreP.Device_Spec_State.PENDING, () => "Pending")
                  .otherwise(() => "")}
              </span>
            }
            editComponent={
              <Select
                size="sm"
                data={[
                  {
                    label: "Active",
                    value:
                      CoreP.Device_Spec_State[CoreP.Device_Spec_State.ACTIVE],
                  },
                  {
                    label: "Pending",
                    value:
                      CoreP.Device_Spec_State[CoreP.Device_Spec_State.PENDING],
                  },
                  {
                    label: "Rejected",
                    value:
                      CoreP.Device_Spec_State[CoreP.Device_Spec_State.REJECTED],
                  },
                ]}
                value={CoreP.Device_Spec_State[item.spec!.state]}
                onChange={(v) => {
                  if (!v) return;
                  item.spec!.state = CoreP.Device_Spec_State[v as "ACTIVE"];
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },

      ...(item.status!.hostname
        ? [
            {
              label: "Hostname",
              value: (
                <span className="text-sm font-mono text-slate-700">
                  {item.status!.hostname}
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.id
        ? [
            {
              label: "Device ID",
              value: (
                <span className="text-sm font-mono text-slate-700">
                  {item.status!.id}
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.serialNumber
        ? [
            {
              label: "Serial number",
              value: (
                <span className="text-sm font-mono text-slate-700">
                  {item.status!.serialNumber}
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.macAddresses?.length > 0
        ? [
            {
              label: "MAC addresses",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.status!.macAddresses.map((x) => (
                    <Label key={x}>
                      <span className="font-mono">{x}</span>
                    </Label>
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),
    ],
  };
};
