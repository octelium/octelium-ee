import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { SSHSessionViewer } from "@/components/SSHRecordingPlayer";
import { getResourceRef } from "@/utils/pb";
import { Terminal } from "lucide-react";
import { Navigate } from "react-router-dom";
import { useContextResource } from "./utils";

const ServiceSSHPage = () => {
  const ctx = useContextResource();

  if (!ctx?.data || ctx.data.kind !== "Service") return null;

  const service = ctx.data as Service;
  if (service.spec?.mode !== Service_Spec_Mode.SSH) {
    return <Navigate to=".." replace />;
  }

  return (
    <div className="w-full space-y-4">
      <header className="flex items-start gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-[0_1px_4px_rgba(15,23,42,0.05)]">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
          <Terminal size={16} strokeWidth={2.2} />
        </span>
        <div className="min-w-0">
          <h2 className="text-sm font-bold text-slate-800">SSH recordings</h2>
          <p className="mt-0.5 text-[0.7rem] font-medium leading-5 text-slate-500">
            Review recorded SSH sessions connected through this Service.
          </p>
        </div>
      </header>
      <SSHSessionViewer serviceRef={getResourceRef(service)} />
    </div>
  );
};

export default ServiceSSHPage;
