import * as E from "@/apis/enterprisev1/enterprisev1";
import CopyText from "@/components/CopyText";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { AlertTriangle, CheckCircle2, Loader2, ShieldCheck } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { getIssuerPresentation } from "./List";

export default (_props: { item: E.CertificateIssuer }) => <></>;

const StateValue = (props: { item: E.CertificateIssuer }) => {
  const p = getIssuerPresentation(props.item);
  return (
    <span className="inline-flex items-center gap-1.5">
      {p.state === E.CertificateIssuer_Status_State.READY ? (
        <CheckCircle2 size={13} className="text-emerald-500" />
      ) : p.state === E.CertificateIssuer_Status_State.PREPARING ? (
        <Loader2 size={13} className="animate-spin text-blue-500" />
      ) : p.state === E.CertificateIssuer_Status_State.NOT_READY ? (
        <AlertTriangle size={13} className="text-red-500" />
      ) : null}
      <span
        className={twMerge(
          "text-sm font-semibold",
          p.state === E.CertificateIssuer_Status_State.READY
            ? "text-emerald-600"
            : p.state === E.CertificateIssuer_Status_State.PREPARING
              ? "text-blue-600"
              : p.state === E.CertificateIssuer_Status_State.NOT_READY
                ? "text-red-600"
                : "text-slate-500",
        )}
      >
        {p.stateLabel}
      </span>
    </span>
  );
};

export const MainInfo = (props: {
  item: E.CertificateIssuer;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getIssuerPresentation(item);

  return {
    items: [
      {
        label: "Operational status",
        value: <StateValue item={item} />,
      },
      {
        label: "Type",
        value: <ResourceListLabel label="Type">{p.type}</ResourceListLabel>,
      },
      {
        label: "Challenge solver",
        value: (
          <ResourceListLabel label="Solver">{p.solver}</ResourceListLabel>
        ),
      },
      ...(p.acme?.server
        ? [
            {
              label: "ACME directory URL",
              value: <CopyText value={p.acme.server} />,
              span: "full" as const,
            },
          ]
        : []),
      ...(p.acme?.email
        ? [
            {
              label: "ACME account email",
              value: <CopyText value={p.acme.email} />,
            },
          ]
        : []),
      ...(p.accountSecret?.name
        ? [
            {
              label: "ACME account secret",
              value: <ResourceListLabel itemRef={p.accountSecret} />,
            },
          ]
        : []),
      {
        label: "Usage",
        span: "full" as const,
        value: (
          <div className="flex items-start gap-2 rounded-lg border border-blue-100 bg-blue-50/60 px-3 py-2.5">
            <ShieldCheck
              size={14}
              className="mt-0.5 shrink-0 text-blue-600"
            />
            <p className="text-[0.7rem] font-semibold leading-relaxed text-blue-800/80">
              Managed Certificates use this issuer to request and renew
              certificates through the configured ACME provider.
              {p.state === E.CertificateIssuer_Status_State.NOT_READY &&
                " The API does not expose a readiness reason; inspect component logs for the underlying failure."}
            </p>
          </div>
        ),
      },
    ],
  };
};
