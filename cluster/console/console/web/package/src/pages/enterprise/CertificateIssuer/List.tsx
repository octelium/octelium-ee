import {
  CertificateIssuer,
  CertificateIssuer_Status_State,
} from "@/apis/enterprisev1/enterprisev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import { getDomain } from "@/utils";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
import { match } from "ts-pattern";

export const getType = (item: CertificateIssuer): string =>
  match(item.spec?.type.oneofKind)
    .with("acme", () => "ACME")
    .otherwise(() => "Not configured");

export const getIssuerState = (state?: CertificateIssuer_Status_State) =>
  match(state)
    .with(CertificateIssuer_Status_State.READY, () => "Ready")
    .with(CertificateIssuer_Status_State.PREPARING, () => "Preparing")
    .with(CertificateIssuer_Status_State.NOT_READY, () => "Not ready")
    .otherwise(() => "Unknown");

export const getIssuerPresentation = (item: CertificateIssuer) => {
  const state = item.status?.state;
  const acme =
    item.spec?.type.oneofKind === "acme" ? item.spec.type.acme : undefined;
  const accountSecret =
    item.status?.type.oneofKind === "acme"
      ? item.status.type.acme.secretRef
      : undefined;
  let serverHost: string | undefined;
  if (acme?.server) {
    try {
      serverHost = new URL(acme.server).hostname;
    } catch {
      serverHost = acme.server;
    }
  }

  return {
    state,
    stateLabel: getIssuerState(state),
    type: getType(item),
    acme,
    accountSecret,
    serverHost,
    solver:
      acme?.solver?.type.oneofKind === "dns" ? "DNS-01" : "Not configured",
  };
};

const ItemDetails = (props: { item: CertificateIssuer; domain: string }) => {
  const { item } = props;
  return <div></div>;
};

export const LabelComponent = (props: { item: CertificateIssuer }) => {
  const p = getIssuerPresentation(props.item);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">{p.type}</ResourceListLabel>
      <ResourceListLabel label="Status">
        {p.state === CertificateIssuer_Status_State.READY ? (
          <CheckCircle2 size={11} className="text-emerald-500" />
        ) : p.state === CertificateIssuer_Status_State.PREPARING ? (
          <Loader2 size={11} className="animate-spin text-blue-500" />
        ) : p.state === CertificateIssuer_Status_State.NOT_READY ? (
          <AlertTriangle size={11} className="text-red-500" />
        ) : null}
        {p.stateLabel}
      </ResourceListLabel>
      <ResourceListLabel label="Solver">{p.solver}</ResourceListLabel>
      {p.serverHost && (
        <ResourceListLabel label="Server">{p.serverHost}</ResourceListLabel>
      )}
      {p.accountSecret?.name && (
        <ResourceListLabel>Account configured</ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: CertificateIssuer }) => {
  const domain = getDomain();
  return <ItemDetails item={props.item} domain={domain} />;
};
