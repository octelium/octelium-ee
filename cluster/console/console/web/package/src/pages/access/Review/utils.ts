import * as AccessP from "@/apis/accessv1/accessv1";
import { match } from "ts-pattern";

export const getDecisionMeta = (
  decision?: AccessP.Review_Spec_Decision,
): { label: string; className: string } =>
  match(decision)
    .with(AccessP.Review_Spec_Decision.APPROVE, () => ({
      label: "Approved",
      className: "text-emerald-600",
    }))
    .with(AccessP.Review_Spec_Decision.REJECT, () => ({
      label: "Rejected",
      className: "text-red-500",
    }))
    .otherwise(() => ({ label: "Pending", className: "text-amber-500" }));
