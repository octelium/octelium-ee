import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { TextInput } from "@mantine/core";
import { Globe2, Mail, ShieldCheck } from "lucide-react";
import * as React from "react";

const cloneForEdit = (item: EnterpriseP.CertificateIssuer) => {
  const next = EnterpriseP.CertificateIssuer.clone(item);
  if (!next.spec) {
    next.spec = EnterpriseP.CertificateIssuer_Spec.create();
  }
  if (next.spec.type.oneofKind !== "acme") {
    next.spec.type = {
      oneofKind: "acme",
      acme: EnterpriseP.CertificateIssuer_Spec_ACME.create({
        solver: {
          type: {
            oneofKind: "dns",
            dns: EnterpriseP.CertificateIssuer_Spec_ACME_Solver_DNS.create(),
          },
        },
      }),
    };
  }
  return next;
};

const isValidURL = (value: string) => {
  if (!value) return true;
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
};

const isValidEmail = (value: string) =>
  !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);

const Edit = (props: {
  item: EnterpriseP.CertificateIssuer;
  onUpdate: (item: EnterpriseP.CertificateIssuer) => void;
}) => {
  const [req, setReq] = React.useState(cloneForEdit(props.item));
  const itemKey = props.item.metadata?.uid || props.item.metadata?.name;

  React.useEffect(() => {
    setReq(cloneForEdit(props.item));
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.CertificateIssuer.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.CertificateIssuer.clone(next));
  };

  if (!req.spec || req.spec.type.oneofKind !== "acme") return null;
  const acme = req.spec.type.acme;

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/50 shadow-[0_1px_3px_rgba(15,23,42,0.035)]">
      <div className="flex items-start gap-3 border-b border-slate-200 bg-white px-4 py-3.5">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
          <ShieldCheck size={17} strokeWidth={2.2} />
        </span>
        <div>
          <h3 className="text-[0.8rem] font-bold text-slate-800">
            ACME configuration
          </h3>
          <p className="mt-0.5 text-[0.68rem] font-semibold leading-relaxed text-slate-400">
            Configure the ACME account used to issue and renew managed cluster
            certificates.
          </p>
        </div>
      </div>

      <div className="grid gap-4 p-4 md:grid-cols-2">
        <TextInput
          required
          type="email"
          label="Account email"
          description="Contact address registered with the ACME provider."
          placeholder="acme@example.com"
          leftSection={<Mail size={14} strokeWidth={2.2} />}
          value={acme.email}
          error={isValidEmail(acme.email) ? undefined : "Enter a valid email address"}
          onChange={(event) => {
            acme.email = event.target.value;
            updateReq();
          }}
        />
        <TextInput
          required
          type="url"
          label="Directory URL"
          description="ACME provider directory endpoint."
          placeholder="https://acme-v02.api.letsencrypt.org/directory"
          leftSection={<Globe2 size={14} strokeWidth={2.2} />}
          value={acme.server}
          error={isValidURL(acme.server) ? undefined : "Enter a valid HTTP or HTTPS URL"}
          onChange={(event) => {
            acme.server = event.target.value;
            updateReq();
          }}
        />

        <div className="rounded-lg border border-slate-200 bg-white px-3 py-2.5 md:col-span-2">
          <div className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400">
            Challenge solver
          </div>
          <div className="mt-1 flex items-center gap-2">
            <span className="rounded-md border border-blue-200 bg-blue-50 px-2 py-1 text-[0.68rem] font-bold text-blue-700">
              DNS-01
            </span>
            <span className="text-[0.68rem] font-semibold text-slate-500">
              Domain ownership is verified using DNS challenge records.
            </span>
          </div>
        </div>
      </div>
    </section>
  );
};

export default Edit;
