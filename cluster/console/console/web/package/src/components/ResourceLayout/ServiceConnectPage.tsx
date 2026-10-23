import { Service } from "@/apis/corev1/corev1";
import ServiceConnect from "@/components/ServiceConnect";
import { hasConnect } from "@/utils/pb";
import { Navigate } from "react-router-dom";
import { useContextResource } from "./utils";

const ServiceConnectPage = () => {
  const ctx = useContextResource();

  if (!ctx?.data || ctx.data.kind !== "Service") return null;

  const service = ctx.data as Service;
  if (!hasConnect(service)) {
    return <Navigate to=".." replace />;
  }

  return <ServiceConnect service={service} />;
};

export default ServiceConnectPage;
