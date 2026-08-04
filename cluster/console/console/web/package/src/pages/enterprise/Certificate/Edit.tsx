import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { SegmentedControl } from "@mantine/core";
import * as React from "react";

const Edit = (props: {
  item: EnterpriseP.Certificate;
  onUpdate: (item: EnterpriseP.Certificate) => void;
}) => {
  const [req, setReq] = React.useState(
    EnterpriseP.Certificate.clone(props.item),
  );
  const itemKey = props.item.metadata?.uid || props.item.metadata?.name;

  React.useEffect(() => {
    setReq(EnterpriseP.Certificate.clone(props.item));
  }, [itemKey]);

  const updateMode = (value: string) => {
    if (!req.spec) return;
    const mode = EnterpriseP.Certificate_Spec_Mode[
      value as keyof typeof EnterpriseP.Certificate_Spec_Mode
    ] as EnterpriseP.Certificate_Spec_Mode;
    if (typeof mode !== "number") return;
    req.spec.mode = mode;
    const next = EnterpriseP.Certificate.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.Certificate.clone(next));
  };

  if (!req.spec) return null;

  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-3">
      <div className="mb-2.5">
        <div className="text-[0.72rem] font-bold text-slate-700">
          Certificate management mode
        </div>
        <div className="mt-0.5 text-[0.67rem] font-semibold text-slate-400">
          Managed certificates are issued by the cluster. Manual certificates
          are supplied and rotated by an administrator.
        </div>
      </div>
      <SegmentedControl
        fullWidth
        value={
          EnterpriseP.Certificate_Spec_Mode[req.spec.mode] ?? "MANAGED"
        }
        onChange={updateMode}
        data={[
          { label: "Managed", value: "MANAGED" },
          { label: "Manual", value: "MANUAL" },
        ]}
      />
    </div>
  );
};

export default Edit;
