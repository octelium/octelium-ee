import * as CoreC from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { ResourceMainInfo } from "@/pages/utils/types";
import { MutedValue } from "@/components/ResourceLayout/overview";
import { Cable, Fingerprint, Globe2, MapPinned, Zap } from "lucide-react";

export default (props: { item: CoreC.Gateway }) => {
  const { item } = props;
  return <></>;
};

export const MainInfo = (props: { item: CoreC.Gateway }): ResourceMainInfo => {
  const { item } = props;
  const status = item.status;
  const wireguard = status?.wireguard;

  return {
    groupOrder: [
      { title: "Networking", icon: Globe2 },
      { title: "Tunnels", icon: Cable },
      { title: "Identity", icon: Fingerprint },
    ],
    items: [
      {
        label: "Region",
        primary: true,
        icon: MapPinned,
        value: status?.regionRef?.name || status?.regionRef?.uid ? (
          <ResourceListLabel itemRef={status.regionRef} />
        ) : (
          <MutedValue>Unknown</MutedValue>
        ),
      },
      {
        label: "Public IP",
        primary: true,
        icon: Globe2,
        value: status?.publicIPs.length ? (
          <CopyText value={status.publicIPs[0]} />
        ) : (
          <MutedValue>None</MutedValue>
        ),
      },
      {
        label: "WireGuard",
        primary: true,
        icon: Cable,
        value: wireguard?.port ? (
          <span className="font-semibold tabular-nums">:{wireguard.port}</span>
        ) : (
          <MutedValue>Disabled</MutedValue>
        ),
      },
      {
        label: "QUIC",
        primary: true,
        icon: Zap,
        value: status?.quicv0?.port ? (
          <span className="font-semibold tabular-nums">:{status.quicv0.port}</span>
        ) : (
          <MutedValue>Disabled</MutedValue>
        ),
      },

      ...(status?.hostname
        ? [
            {
              label: "Hostname",
              group: "Networking",
              span: "full" as const,
              value: <CopyText value={status.hostname} />,
            },
          ]
        : []),
      ...(status?.publicIPs.length
        ? [
            {
              label: "Public IPs",
              group: "Networking",
              span: "full" as const,
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
      ...(status?.cidr?.v4 || status?.cidr?.v6
        ? [
            {
              label: "Client CIDR",
              group: "Networking",
              span: "full" as const,
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

      ...(wireguard?.publicKey
        ? [
            {
              label: "WireGuard public key",
              group: "Tunnels",
              span: "full" as const,
              value: <CopyText value={wireguard.publicKey} />,
            },
          ]
        : []),
      ...(wireguard?.keyRotatedAt
        ? [
            {
              label: "Key rotated",
              group: "Tunnels",
              value: <TimeAgo rfc3339={wireguard.keyRotatedAt} />,
            },
          ]
        : []),
      ...(wireguard?.port
        ? [
            {
              label: "WireGuard port",
              group: "Tunnels",
              value: <span className="tabular-nums">{wireguard.port}</span>,
            },
          ]
        : []),
      ...(status?.quicv0?.port
        ? [
            {
              label: "QUIC port",
              group: "Tunnels",
              value: <span className="tabular-nums">{status.quicv0.port}</span>,
            },
          ]
        : []),

      ...(status?.id
        ? [
            {
              label: "Gateway ID",
              group: "Identity",
              value: <CopyText value={status.id} />,
            },
          ]
        : []),
      ...(status?.index !== undefined
        ? [
            {
              label: "Index",
              group: "Identity",
              value: <span className="tabular-nums">{status.index}</span>,
            },
          ]
        : []),
      ...(status?.nodeRef?.name
        ? [
            {
              label: "Node",
              group: "Identity",
              value: status.nodeRef.name,
            },
          ]
        : []),
    ],
  };
};
