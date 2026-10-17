import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import LLMPlayground from "@/components/LLMPlayground";
import MCPPlayground from "@/components/MCPPlayground";
import { hasPlayground } from "@/utils/pb";
import { Navigate } from "react-router-dom";
import { useContextResource } from "./utils";

const ServicePlaygroundPage = () => {
  const ctx = useContextResource();

  if (!ctx?.data || ctx.data.kind !== "Service") return null;

  const service = ctx.data as Service;
  if (!hasPlayground(service)) {
    return <Navigate to=".." replace />;
  }

  const key = service.metadata?.uid ?? service.metadata?.name;

  return service.spec?.mode === Service_Spec_Mode.LLM ? (
    <LLMPlayground key={key} service={service} />
  ) : (
    <MCPPlayground key={key} service={service} />
  );
};

export default ServicePlaygroundPage;
