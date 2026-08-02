import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getClientCore } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { Network, PanelTop } from "lucide-react";

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

  return {
    items: [
      ...(status?.index !== undefined && status.index > 0
        ? [
            {
              label: "Index",
              value: <span className="font-semibold">{status.index}</span>,
            },
          ]
        : []),

      ...(status?.version
        ? [
            {
              label: "Version",
              value: (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-[0.72rem] font-bold bg-slate-100 border border-slate-200 text-slate-700">
                  {status.version}
                </span>
              ),
            },
          ]
        : []),

      {
        label: "Related resources",
        value: (
          <div className="flex flex-wrap gap-1">
            <ResourceListLabel
              label="Gateways"
              to={`/core/gateways?regionRef.name=${encodeURIComponent(itemName)}`}
            >
              <Network size={12} strokeWidth={2.5} />
              {qryGateways.data?.listResponseMeta?.totalCount?.toLocaleString() ??
                "…"}
            </ResourceListLabel>
            <ResourceListLabel
              label="Services"
              to={`/core/services?regionRef.name=${encodeURIComponent(itemName)}`}
            >
              <PanelTop size={12} strokeWidth={2.5} />
              {qryServices.data?.listResponseMeta?.totalCount?.toLocaleString() ??
                "…"}
            </ResourceListLabel>
          </div>
        ),
        span: "full",
      },

      ...(status?.publicHostname
        ? [
            {
              label: "Public Hostname",
              value: <CopyText value={status.publicHostname} />,
            },
          ]
        : []),

      ...(status?.ingressAddresses && status.ingressAddresses.length > 0
        ? [
            {
              label: "Ingress Addresses",
              value: (
                <div className="flex flex-col gap-0.5">
                  {status.ingressAddresses.map((addr) => (
                    <CopyText key={addr} value={addr} />
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(status?.versionInfoMap &&
      Object.keys(status.versionInfoMap).length > 0
        ? [
            {
              label: "Version Info",
              span: "full" as const,
              value: (
                <div className="flex flex-col gap-2 w-full">
                  {Object.entries(status.versionInfoMap).map(([key, info]) => (
                    <div
                      key={key}
                      className="flex flex-col gap-1 p-3 rounded-lg border border-slate-100 bg-slate-50/60"
                    >
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-500 mb-0.5">
                        {key}
                      </span>
                      <div className="flex flex-col gap-1">
                        {info.version && (
                          <div className="flex items-center gap-2">
                            <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                              Version
                            </span>
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[0.68rem] font-bold bg-white border border-slate-200 text-slate-700">
                              {info.version}
                            </span>
                          </div>
                        )}
                        {info.package && (
                          <div className="flex items-center gap-2">
                            <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                              Package
                            </span>
                            <span className="text-[0.75rem] font-semibold text-slate-600">
                              {info.package}
                            </span>
                          </div>
                        )}
                        {info.id && (
                          <div className="flex items-center gap-2">
                            <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                              ID
                            </span>
                            <CopyText value={info.id} />
                          </div>
                        )}
                        {info.setAt && (
                          <div className="flex items-center gap-2">
                            <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-20 shrink-0">
                              Set At
                            </span>
                            <span className="text-[0.75rem] font-semibold text-slate-600">
                              <TimeAgo rfc3339={info.setAt} />
                            </span>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              ),
            },
          ]
        : []),
    ],
  };
};
