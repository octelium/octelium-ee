import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { Globe2, Hash, Network, Package, PanelTop } from "lucide-react";
import Label from "@/components/Label";
import {
  HistoryList,
  MutedValue,
} from "@/components/ResourceLayout/overview";

export const AccessLog = (props: { item: CoreC.Region }) => {
  return <AccessLogViewer regionRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreC.Region }) => {
  const { item } = props;
  return <div className="w-full"></div>;
};

export const MainInfo = (props: { item: CoreC.Region }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const itemName = item.metadata!.name;
  const itemRef = getResourceRef(item);
  const qryGateways = useQuery({
    queryKey: ["core.listGateway", "region", itemName],
    queryFn: () =>
      getClientCore().listGateway(
        CoreC.ListGatewayOptions.create({ regionRef: itemRef }),
      ).response,
  });
  const qryServices = useQuery({
    queryKey: ["core.listService", "region", itemName],
    queryFn: () =>
      getClientCore().listService(
        CoreC.ListServiceOptions.create({ regionRef: itemRef }),
      ).response,
  });
  const versions = Object.entries(status?.versionInfoMap ?? {});

  return {
    groupOrder: [
      { title: "Networking", icon: Globe2 },
      { title: "Components", icon: Package },
    ],
    items: [
      {
        label: "Version",
        primary: true,
        icon: Package,
        value: status?.version ? (
          <Label>{status.version}</Label>
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Gateways",
        primary: true,
        icon: Network,
        value: (
          <ResourceListLabel
            label="Gateways"
            to={`/core/gateways?regionRef.name=${encodeURIComponent(itemName)}`}
          >
            {qryGateways.data?.listResponseMeta?.totalCount?.toLocaleString() ??
              "…"}
          </ResourceListLabel>
        ),
      },
      {
        label: "Services",
        primary: true,
        icon: PanelTop,
        value: (
          <ResourceListLabel
            label="Services"
            to={`/core/services?regionRef.name=${encodeURIComponent(itemName)}`}
          >
            {qryServices.data?.listResponseMeta?.totalCount?.toLocaleString() ??
              "…"}
          </ResourceListLabel>
        ),
      },
      {
        label: "Index",
        primary: true,
        icon: Hash,
        value: (
          <span className="font-semibold tabular-nums">{status?.index ?? 0}</span>
        ),
      },

      {
        label: "Public hostname",
        group: "Networking",
        span: "full" as const,
        value: status?.publicHostname ? (
          <CopyText value={status.publicHostname} />
        ) : (
          <MutedValue>Not set</MutedValue>
        ),
      },
      ...(status?.ingressAddresses.length
        ? [
            {
              label: "Ingress addresses",
              group: "Networking",
              span: "full" as const,
              value: (
                <div className="flex flex-wrap gap-1">
                  {status.ingressAddresses.map((addr) => (
                    <ResourceListLabel key={addr} label="Address">
                      <CopyText value={addr} />
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(versions.length
        ? [
            {
              label: "",
              group: "Components",
              layout: "stacked" as const,
              value: (
                <HistoryList
                  entries={versions.map(([key, info]) => ({
                    key,
                    tone: "neutral" as const,
                    label: key,
                    meta: (
                      <>
                        {info.version && <Label size="sm">{info.version}</Label>}
                        {info.package && <span>{info.package}</span>}
                        {info.setAt && (
                          <span>
                            Set <TimeAgo rfc3339={info.setAt} />
                          </span>
                        )}
                      </>
                    ),
                    detail: info.id ? <CopyText value={info.id} /> : undefined,
                  }))}
                />
              ),
            },
          ]
        : []),
    ],
  };
};
