import * as CoreP from "@/apis/corev1/corev1";
import { Scope, Update } from "./fields";
import { HTTPForm } from "./http";
import { LLMForm } from "./llm";
import { MCPForm } from "./mcp";
import { Config } from "./model";
import { ModeInfo } from "./modes";
import {
  KubernetesForm,
  MySQLForm,
  PostgresForm,
  RDPForm,
  SOCKS5Form,
  SSHForm,
} from "./protocols";
import { TLSSection } from "./tls";
import { UpstreamSection } from "./upstream";

const SSLMode = CoreP.Service_Spec_Config_Postgres_SSLMode;

export const ProtocolForm = (props: {
  config: Config;
  info: ModeInfo;
  scope: Scope;
  update: Update;
}) => {
  const { config, info, scope, update } = props;
  const type = config.type;
  switch (type.oneofKind) {
    case "http":
      return (
        <HTTPForm
          value={type.http}
          mode={info.mode}
          scope={scope}
          update={update}
        />
      );
    case "mcp":
      return <MCPForm value={type.mcp} scope={scope} update={update} />;
    case "llm":
      return <LLMForm value={type.llm} scope={scope} update={update} />;
    case "ssh":
      return <SSHForm value={type.ssh} update={update} />;
    case "postgres":
      return <PostgresForm value={type.postgres} update={update} />;
    case "mysql":
      return <MySQLForm value={type.mysql} update={update} />;
    case "kubernetes":
      return <KubernetesForm value={type.kubernetes} update={update} />;
    case "socks5":
      return <SOCKS5Form value={type.socks5} update={update} />;
    case "rdp":
      return <RDPForm value={type.rdp} update={update} />;
    default:
      return null;
  }
};

const embeddedOption = (config: Config, update: Update) => {
  const type = config.type;
  if (type.oneofKind === "ssh") {
    return {
      enabled: type.ssh.eSSHMode,
      label: "Connected client",
      hint: "Embedded SSH",
      note: "Each connected Octelium client serves its own embedded SSH server, so there is no upstream to set.",
      onChange: (enabled: boolean) => {
        type.ssh.eSSHMode = enabled;
        update();
      },
    };
  }
  if (type.oneofKind === "socks5") {
    return {
      enabled: type.socks5.isEmbeddedMode,
      label: "Connected client",
      hint: "Embedded SOCKS5",
      note: "Connected Octelium clients act as the SOCKS5 egress, so there is no upstream to set.",
      onChange: (enabled: boolean) => {
        type.socks5.isEmbeddedMode = enabled;
        update();
      },
    };
  }
  return undefined;
};

export const hasUpstreamTLS = (config: Config, info: ModeInfo) => {
  if (info.upstreamTLS === "none") return false;
  const type = config.type;
  return !(
    type.oneofKind === "postgres" && type.postgres.sslMode === SSLMode.DISABLE
  );
};

export const UpstreamForm = (props: {
  config: Config;
  info: ModeInfo;
  update: Update;
}) => {
  const { config, info, update } = props;
  return (
    <>
      <UpstreamSection
        config={config}
        info={info}
        update={update}
        embedded={embeddedOption(config, update)}
      />
      {hasUpstreamTLS(config, info) && (
        <TLSSection
          config={config}
          support={info.upstreamTLS}
          update={update}
        />
      )}
    </>
  );
};

