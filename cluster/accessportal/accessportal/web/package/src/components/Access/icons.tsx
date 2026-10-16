import * as UserP from "@/apis/userv1/userv1";
import {
  Bot,
  Boxes,
  Cable,
  Database,
  Globe,
  Layers,
  Monitor,
  Server,
  ShieldCheck,
  Terminal,
  type LucideIcon,
  type LucideProps,
} from "lucide-react";

const SERVICE_ICONS: Partial<Record<UserP.Service_Spec_Type, LucideIcon>> = {
  [UserP.Service_Spec_Type.HTTP]: Globe,
  [UserP.Service_Spec_Type.WEB]: Globe,
  [UserP.Service_Spec_Type.DNS]: Globe,
  [UserP.Service_Spec_Type.SSH]: Terminal,
  [UserP.Service_Spec_Type.POSTGRES]: Database,
  [UserP.Service_Spec_Type.MYSQL]: Database,
  [UserP.Service_Spec_Type.KUBERNETES]: Boxes,
  [UserP.Service_Spec_Type.MCP]: Bot,
  [UserP.Service_Spec_Type.LLM]: Bot,
  [UserP.Service_Spec_Type.TCP]: Cable,
  [UserP.Service_Spec_Type.UDP]: Cable,
  [UserP.Service_Spec_Type.GRPC]: Cable,
  [UserP.Service_Spec_Type.SOCKS5]: ShieldCheck,
  [UserP.Service_Spec_Type.RDP_WEB]: Monitor,
};

export const ServiceTypeIcon = ({
  serviceType,
  ...props
}: LucideProps & { serviceType?: UserP.Service_Spec_Type }) => {
  const Icon = serviceType === undefined ? Server : SERVICE_ICONS[serviceType] ?? Server;
  return <Icon {...props} />;
};

export const ResourceIcon = ({
  kind,
  ...props
}: LucideProps & { kind: "Service" | "Catalog" | "Unknown" }) => {
  const Icon = kind === "Catalog" ? Layers : Server;
  return <Icon {...props} />;
};
