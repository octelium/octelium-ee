import { Dimension } from "@/apis/visibilityv1/llm/vllmv1";
import LLMVisibility from "@/components/LLMVisibility";
import type { LLMScope } from "@/components/LLMVisibility/utils";
import PageWrap from "@/components/PageWrap";
import { getResourceRef, hasLLMVisibility, Resource } from "@/utils/pb";
import { BrainCircuit } from "lucide-react";
import { Navigate } from "react-router-dom";
import { match } from "ts-pattern";
import { useContextResource } from "./utils";

type PrincipalKind = "user" | "session" | "service" | "policy";

export const ResourceLLM = (props: { resource: Resource }) => {
  const { resource } = props;

  if (!hasLLMVisibility(resource)) {
    return <Navigate to=".." replace />;
  }

  const scope: LLMScope = {};
  let hidden: Dimension[] = [];
  let principalKinds: PrincipalKind[] = [];

  if (
    !match(resource.kind)
      .with("User", () => {
        scope.userRef = getResourceRef(resource);
        hidden = [Dimension.USER];
        principalKinds = ["session", "service", "policy"];
        return true;
      })
      .with("Session", () => {
        scope.sessionRef = getResourceRef(resource);
        hidden = [Dimension.USER, Dimension.SESSION, Dimension.DEVICE];
        principalKinds = ["service", "policy"];
        return true;
      })
      .with("Device", () => {
        scope.deviceRef = getResourceRef(resource);
        hidden = [Dimension.DEVICE];
        principalKinds = ["user", "session", "service"];
        return true;
      })
      .with("Service", () => {
        scope.serviceRef = getResourceRef(resource);
        hidden = [Dimension.SERVICE, Dimension.NAMESPACE];
        principalKinds = ["user", "session", "policy"];
        return true;
      })
      .with("Namespace", () => {
        scope.namespaceRef = getResourceRef(resource);
        hidden = [Dimension.NAMESPACE];
        principalKinds = ["user", "session", "service"];
        return true;
      })
      .with("Region", () => {
        scope.regionRef = getResourceRef(resource);
        hidden = [Dimension.REGION];
        principalKinds = ["user", "session", "service"];
        return true;
      })
      .otherwise(() => false)
  ) {
    return <></>;
  }

  const displayName =
    resource.metadata?.displayName || resource.metadata?.name || "Resource";

  return (
    <div className="flex w-full flex-col gap-4">
      <header className="flex flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_2px_10px_rgba(15,23,42,0.04)] sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
            <BrainCircuit size={18} strokeWidth={2.2} />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-base font-bold text-slate-900">
              LLM visibility
            </h1>
            <p className="mt-0.5 truncate text-[0.72rem] font-semibold text-slate-500">
              Inference activity, tokens, guardrails and cost controls scoped to
              this resource.
            </p>
            <span className="mt-2 inline-flex max-w-full truncate rounded-md bg-slate-100 px-2 py-1 text-[0.65rem] font-bold text-slate-600">
              {resource.kind} · {displayName}
            </span>
          </div>
        </div>
        <span className="inline-flex w-fit shrink-0 items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-[0.64rem] font-bold uppercase tracking-[0.06em] text-slate-500">
          Resource scope
        </span>
      </header>

      <LLMVisibility
        scope={scope}
        hideDimensions={hidden}
        principalKinds={principalKinds}
      />
    </div>
  );
};

const ResourceItemLLMPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceLLM resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemLLMPage;
