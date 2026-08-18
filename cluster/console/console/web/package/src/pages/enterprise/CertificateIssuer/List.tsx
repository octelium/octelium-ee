import {
  CertificateIssuer,
  CertificateIssuer_Status_State,
} from "@/apis/enterprisev1/enterprisev1";
import { GetCertificateIssuerSummaryResponse } from "@/apis/visibilityv1/enterprise/venterprisev1";
import { SummaryItemCount, SummaryItemCountWrap, SummaryNoItems } from "@/components/Summary";
import { getClientVisibilityEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import { getDomain } from "@/utils";
import { AlertTriangle, CheckCircle2, Globe2, Loader2 } from "lucide-react";
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

const DoSummary = ({ resp }: { resp: GetCertificateIssuerSummaryResponse }) => <SummaryItemCountWrap>
  <SummaryItemCount count={resp.totalNumber} to="/enterprise/certificateissuers">Total</SummaryItemCount>
  <SummaryItemCount count={resp.totalACME} to="/enterprise/certificateissuers?type=ACME" icon={Globe2}>ACME</SummaryItemCount>
  <SummaryItemCount count={resp.totalPreparing} to="/enterprise/certificateissuers?state=PREPARING" icon={Loader2}>Preparing</SummaryItemCount>
  <SummaryItemCount count={resp.totalReady} to="/enterprise/certificateissuers?state=READY" icon={CheckCircle2}>Ready</SummaryItemCount>
  <SummaryItemCount count={resp.totalNotReady} to="/enterprise/certificateissuers?state=NOT_READY" icon={AlertTriangle}>Not ready</SummaryItemCount>
</SummaryItemCountWrap>;

export const Summary = ({ showNoItems }: { showNoItems?: boolean }) => {
  const query = useQuery({ queryKey: ["visibility", "enterprise", "summary", "CertificateIssuer"], queryFn: async () => (await getClientVisibilityEnterprise().getCertificateIssuerSummary({})).response });
  if (!query.data) return null;
  return query.data.totalNumber > 0 ? <DoSummary resp={query.data} /> : showNoItems ? <SummaryNoItems /> : null;
};
