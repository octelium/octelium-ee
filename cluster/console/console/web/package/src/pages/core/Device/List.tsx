import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import { Device } from "@/apis/corev1/corev1";

import { getDomain, toNumOrZero } from "@/utils";
import Label from "@/components/Label";
import { match } from "ts-pattern";
import * as CoreP from "@/apis/corev1/corev1";
import { twMerge } from "tailwind-merge";
import { getOSIcon, getOSStr } from "./Main";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { GetDeviceSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { useQuery } from "@tanstack/react-query";
import { getClientVisibilityCore } from "@/utils/client";
import { toURLWithQry } from "@/pages/utils";
import { useSearchParams } from "react-router-dom";
import PieChart from "@/components/Charts/PieChart";

export const getState = (itm: CoreP.Device): string => {
  return match(itm.spec?.state)
    .with(CoreP.Device_Spec_State.ACTIVE, () => "Active")
    .with(CoreP.Device_Spec_State.REJECTED, () => "Rejected")
    .with(CoreP.Device_Spec_State.PENDING, () => "Pending")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: Device; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const DeviceTypeLabel = (props: { item: Device }) => {
  const { item } = props;
  return (
    <ResourceListLabel>
      <span className="flex items-center">
        <span className="mr-1">{getOSIcon(item)}</span>
        <span>{getOSStr(item)}</span>
      </span>
    </ResourceListLabel>
  );
};

export const LabelComponent = (props: { item: Device }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel>
        <span
          className={twMerge(
            match(item.spec!.state)
              .with(CoreP.Device_Spec_State.REJECTED, () => `text-red-400`)
              .with(CoreP.Device_Spec_State.ACTIVE, () => `text-green-400`)
              .with(CoreP.Device_Spec_State.PENDING, () => `text-yellow-400`)
              .otherwise(() => undefined)
          )}
        >
          {getState(item)}
        </span>
      </ResourceListLabel>
      <DeviceTypeLabel item={item} />
      {item.status?.hostname && (
        <ResourceListLabel label="Hostname">
          {item.status!.hostname}
        </ResourceListLabel>
      )}

      <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Device }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: { resp: GetDeviceSummaryResponse }) => {
  const { resp } = props;
  const [searchParams, _] = useSearchParams();
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/devices`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalActive}
          to={toURLWithQry(`/core/devices`, {
            state: CoreP.Device_Spec_State[CoreP.Device_Spec_State.ACTIVE],
          })}
          active={
            searchParams.get(`state`) ===
            CoreP.Device_Spec_State[CoreP.Device_Spec_State.ACTIVE]
          }
        >
          Active
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalRejected}
          to={toURLWithQry(`/core/devices`, {
            state: CoreP.Device_Spec_State[CoreP.Device_Spec_State.REJECTED],
          })}
          active={
            searchParams.get(`state`) ===
            CoreP.Device_Spec_State[CoreP.Device_Spec_State.REJECTED]
          }
        >
          Rejected
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalPending}
          to={toURLWithQry(`/core/devices`, {
            state: CoreP.Device_Spec_State[CoreP.Device_Spec_State.PENDING],
          })}
          active={
            searchParams.get(`state`) ===
            CoreP.Device_Spec_State[CoreP.Device_Spec_State.PENDING]
          }
        >
          Pending
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalLinux}
          to={toURLWithQry(`/core/devices`, {
            osType:
              CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.LINUX],
          })}
          active={
            searchParams.get(`osType`) ===
            CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.LINUX]
          }
        >
          Linux
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalWindows}
          to={toURLWithQry(`/core/devices`, {
            osType:
              CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.WINDOWS],
          })}
          active={
            searchParams.get(`osType`) ===
            CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.WINDOWS]
          }
        >
          Windows
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalMac}
          to={toURLWithQry(`/core/devices`, {
            osType: CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.MAC],
          })}
          active={
            searchParams.get(`osType`) ===
            CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.MAC]
          }
        >
          Mac OS
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
    queryKey: ["visibility", "core", "summary", "Device"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getDeviceSummary({});

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
                  { name: "Linux", value: d.totalLinux },
                  { name: "Windows", value: d.totalWindows },
                  { name: "MacOS", value: d.totalMac },
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
