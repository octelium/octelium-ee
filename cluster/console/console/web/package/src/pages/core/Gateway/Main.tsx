import * as CoreC from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";

export default (props: { item: CoreC.Gateway }) => {
  const { item } = props;
  return <></>;
};

export const MainInfo = (props: { item: CoreC.Gateway }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;

  return {
    items: [
      ...(status?.id
        ? [
            {
              label: "ID",
              value: <CopyText value={status.id} />,
            },
          ]
        : []),

      ...(status?.regionRef?.name || status?.regionRef?.uid
        ? [
            {
              label: "Region",
              value: <ResourceListLabel itemRef={item.status!.regionRef} />,
            },
          ]
        : []),

      ...(status?.index !== undefined
        ? [
            {
              label: "Index",
              value: <span className="font-semibold">{status.index}</span>,
            },
          ]
        : []),

      ...(status?.cidr?.v4 || status?.cidr?.v6
        ? [
            {
              label: "CIDR",
              value: (
                <div className="flex flex-wrap gap-1">
                  {status.cidr.v4 && (
                    <ResourceListLabel label="IPv4">
                      <CopyText value={status.cidr.v4} />
                    </ResourceListLabel>
                  )}
                  {status.cidr.v6 && (
                    <ResourceListLabel label="IPv6">
                      <CopyText value={status.cidr.v6} />
                    </ResourceListLabel>
                  )}
                </div>
              ),
            },
          ]
        : []),

      ...(status?.publicIPs && status.publicIPs.length > 0
        ? [
            {
              label: "Public IPs",
              value: (
                <div className="flex flex-wrap gap-1">
                  {status.publicIPs.map((ip) => (
                    <ResourceListLabel key={ip} label="Address">
                      <CopyText value={ip} />
                    </ResourceListLabel>
                  ))}
                </div>
              ),
            },
          ]
        : []),

      ...(status?.hostname
        ? [
            {
              label: "Hostname",
              value: <CopyText value={status.hostname} />,
            },
          ]
        : []),

      ...(status?.wireguard &&
      (status.wireguard.port > 0 ||
        !!status.wireguard.publicKey ||
        !!status.wireguard.keyRotatedAt)
        ? [
            {
              label: "WireGuard",
              span: "full" as const,
              value: (
                <div className="flex flex-col gap-1.5 w-full">
                  {status.wireguard.port > 0 && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-24 shrink-0">
                        Port
                      </span>
                      <span className="text-[0.78rem] font-semibold text-slate-700">
                        {status.wireguard.port}
                      </span>
                    </div>
                  )}
                  {status.wireguard.publicKey && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-24 shrink-0">
                        Public Key
                      </span>
                      <CopyText
                        value={status.wireguard.publicKey}
                        truncate={32}
                      />
                    </div>
                  )}
                  {status.wireguard.keyRotatedAt && (
                    <div className="flex items-center gap-2">
                      <span className="text-[0.68rem] font-bold uppercase tracking-[0.05em] text-slate-400 w-24 shrink-0">
                        Key Rotated
                      </span>
                      <span className="text-[0.78rem] font-semibold text-slate-700">
                        <TimeAgo rfc3339={status.wireguard.keyRotatedAt} />
                      </span>
                    </div>
                  )}
                </div>
              ),
            },
          ]
        : []),

      ...(status?.quicv0 && status.quicv0.port > 0
        ? [
            {
              label: "QUICv0",
              value: (
                <span className="font-semibold">:{status.quicv0.port}</span>
              ),
            },
          ]
        : []),
    ],
  };
};
