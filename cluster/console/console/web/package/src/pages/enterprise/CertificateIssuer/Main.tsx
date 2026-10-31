import * as E from "@/apis/enterprisev1/enterprisev1";
import CopyText from "@/components/CopyText";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import {
  AlertTriangle,
  Globe2,
  Landmark,
  Loader2,
  Mail,
  Waypoints,
} from "lucide-react";
import Label from "@/components/Label";
import {
  MutedValue,
  OverviewHero,
  OverviewNotice,
} from "@/components/ResourceLayout/overview";
import { getIssuerPresentation } from "./List";

export default (_props: { item: E.CertificateIssuer }) => <></>;

const stateTone = (state?: E.CertificateIssuer_Status_State) =>
  state === E.CertificateIssuer_Status_State.READY
    ? ("success" as const)
    : state === E.CertificateIssuer_Status_State.PREPARING
      ? ("info" as const)
      : state === E.CertificateIssuer_Status_State.NOT_READY
        ? ("danger" as const)
        : ("neutral" as const);

const IssuerHero = (props: { item: E.CertificateIssuer }) => {
  const p = getIssuerPresentation(props.item);
  const tone = stateTone(p.state);

  return (
    <OverviewHero
      tone={tone}
      icon={
        p.state === E.CertificateIssuer_Status_State.PREPARING ? (
          <Loader2 size={18} className="motion-safe:animate-spin" />
        ) : p.state === E.CertificateIssuer_Status_State.NOT_READY ? (
          <AlertTriangle size={18} strokeWidth={2.2} />
        ) : (
          <Landmark size={18} strokeWidth={2.2} />
        )
      }
      title={
        p.state === E.CertificateIssuer_Status_State.READY
          ? "Ready to issue certificates"
          : p.state === E.CertificateIssuer_Status_State.PREPARING
            ? "Preparing the ACME account"
            : p.state === E.CertificateIssuer_Status_State.NOT_READY
              ? "Not ready"
              : p.stateLabel
      }
      detail={[p.type, p.serverHost, `${p.solver} challenges`]
        .filter(Boolean)
        .join(" · ")}
    >
      {p.state === E.CertificateIssuer_Status_State.NOT_READY && (
        <OverviewNotice tone="danger" icon={AlertTriangle} title="Managed Certificates cannot be issued.">
          The API does not expose a readiness reason; inspect the component
          logs for the underlying failure.
        </OverviewNotice>
      )}
    </OverviewHero>
  );
};

export const MainInfo = (props: {
  item: E.CertificateIssuer;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getIssuerPresentation(item);

  return {
    header: <IssuerHero item={item} />,
    status: { label: p.stateLabel, tone: stateTone(p.state) },
    groupOrder: [{ title: "ACME", icon: Landmark }],
    items: [
      {
        label: "Type",
        primary: true,
        icon: Landmark,
        value: <Label>{p.type}</Label>,
      },
      {
        label: "Challenge solver",
        primary: true,
        icon: Waypoints,
        value: <span className="font-semibold">{p.solver}</span>,
      },
      {
        label: "Directory",
        primary: true,
        icon: Globe2,
        value: p.serverHost ? (
          <span className="font-semibold">{p.serverHost}</span>
        ) : (
          <MutedValue>Not configured</MutedValue>
        ),
      },
      {
        label: "Account email",
        primary: true,
        icon: Mail,
        value: p.acme?.email ? (
          <CopyText value={p.acme.email} />
        ) : (
          <MutedValue>Not set</MutedValue>
        ),
      },
      ...(p.acme?.server
        ? [
            {
              label: "Directory URL",
              group: "ACME",
              value: <CopyText value={p.acme.server} />,
              span: "full" as const,
            },
          ]
        : []),
      ...(p.accountSecret?.name
        ? [
            {
              label: "Account key",
              group: "ACME",
              value: <ResourceListLabel itemRef={p.accountSecret} />,
            },
          ]
        : []),
      {
        label: "Usage",
        group: "ACME",
        span: "full" as const,
        value: (
          <MutedValue>
            Managed Certificates use this issuer to request and renew their
            certificates.
          </MutedValue>
        ),
      },
    ],
  };
};
