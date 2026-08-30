import * as UserP from "@/apis/userv1/userv1";
import {
  Bot,
  Boxes,
  Cable,
  Database,
  Globe,
  Layers,
  Monitor,
  Network,
  Server,
  ShieldCheck,
  Terminal,
} from "lucide-react";

export const serviceTypeIcon = (type?: UserP.Service_Spec_Type) => {
  switch (type) {
    case UserP.Service_Spec_Type.HTTP:
    case UserP.Service_Spec_Type.WEB:
    case UserP.Service_Spec_Type.DNS:
      return Globe;
    case UserP.Service_Spec_Type.SSH:
      return Terminal;
    case UserP.Service_Spec_Type.POSTGRES:
    case UserP.Service_Spec_Type.MYSQL:
      return Database;
    case UserP.Service_Spec_Type.KUBERNETES:
      return Boxes;
    case UserP.Service_Spec_Type.MCP:
    case UserP.Service_Spec_Type.LLM:
      return Bot;
    case UserP.Service_Spec_Type.TCP:
    case UserP.Service_Spec_Type.UDP:
    case UserP.Service_Spec_Type.GRPC:
      return Cable;
    case UserP.Service_Spec_Type.SOCKS5:
      return ShieldCheck;
    case UserP.Service_Spec_Type.RDP_WEB:
      return Monitor;
    default:
      return Server;
  }
};

export const resourceIcon = (kind: "Service" | "Catalog" | "Unknown") =>
  kind === "Catalog" ? Layers : kind === "Service" ? Network : Server;
