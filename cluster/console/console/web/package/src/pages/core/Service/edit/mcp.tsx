import * as CoreP from "@/apis/corev1/corev1";
import {
  FormSection,
  Grid,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { Gauge, KeyRound, ShieldCheck } from "lucide-react";
import { AuthEditor } from "./auth";
import {
  joinSummary,
  NumberField,
  OptionList,
  OptionRow,
  Scope,
  TagsField,
  TextField,
  Toggle,
  Update,
} from "./fields";
import { guardrailSummary, GuardrailForm } from "./guardrail";
import {
  CORSRow,
  headerVisibilitySummary,
  HeadersRow,
  PathRow,
  VisibilityHeaders,
  VisibilityRow,
} from "./headers";
import { TransportCard } from "./http";
import { PluginDefinition, PluginsSection, SHARED_PLUGINS } from "./plugins";

const VERSIONS = [
  "2024-11-05",
  "2025-03-26",
  "2025-06-18",
  "2025-11-25",
  "2026-07-28",
];

const endpointError = (value: string) => {
  if (!value.startsWith("/")) return "Start the path with /";
  if (/[?#]/.test(value)) return "Remove the query or fragment";
  if (value.includes("//") || /\/\.\.?(\/|$)/.test(value)) {
    return "Use a canonical path";
  }
  if (value.startsWith("/.well-known/")) return "This path is reserved";
  return undefined;
};

const MCP_PLUGINS: PluginDefinition[] = [
  {
    type: "guardrail",
    label: "Guardrail",
    description: "Inspect tool calls, results and resources",
    icon: ShieldCheck,
    group: "Content safety",
    phased: false,
    summary: guardrailSummary,
    create: () => ({
      oneofKind: "guardrail",
      guardrail: CoreP.Service_Spec_Config_MCP_Plugin_Guardrail.create({
        leg: CoreP.Service_Spec_Config_MCP_Plugin_Guardrail_Leg.REQUEST,
      }),
    }),
  },
  ...SHARED_PLUGINS,
];

export const MCPForm = (props: {
  value: CoreP.Service_Spec_Config_MCP;
  scope: Scope;
  update: Update;
}) => {
  const { value, scope, update } = props;
  const global = scope === "default";
  const { protocol, limits, visibility } = value;

  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Connection"
        description="How clients reach this gateway and how it reaches the MCP server."
      >
        {global && (
          <TextField
            of={value}
            name="endpoint"
            update={update}
            label="Endpoint"
            description="When set, only requests to this exact path are accepted."
            placeholder="/mcp"
            mono
            error={endpointError}
          />
        )}
        <AuthEditor holder={value} update={update} />
      </FormSection>

      <PluginsSection
        holder={value}
        kind="mcp"
        definitions={MCP_PLUGINS}
        scope={scope}
        update={update}
        description="Inspect and shape the JSON-RPC traffic per request."
        renderType={(plugin, updatePlugin) =>
          plugin.type.oneofKind === "guardrail" ? (
            <GuardrailForm
              value={plugin.type.guardrail}
              kind="mcp"
              update={updatePlugin}
            />
          ) : null
        }
      />

      <OptionList title="Protocol and traffic">
        {global && (
          <OptionRow
            icon={ShieldCheck}
            title="Protocol validation"
            description="Restrict the accepted MCP versions and methods."
            summary={
              protocol &&
              (joinSummary(
                protocol.versions.length > 0 && protocol.versions.join(", "),
                protocol.requireVersion && "Version required",
                protocol.rejectUnknownMethods && "Unknown methods rejected",
              ) ||
                "Every version accepted")
            }
            isSet={!!protocol}
            onAdd={() => {
              value.protocol = CoreP.Service_Spec_Config_MCP_Protocol.create();
              update();
            }}
            onRemove={() => {
              value.protocol = undefined;
              update();
            }}
          >
            {protocol && (
              <>
                <TagsField
                  of={protocol}
                  name="versions"
                  update={update}
                  label="Accepted versions"
                  description="Every version is accepted when this is empty."
                  placeholder="2025-11-25"
                  data={VERSIONS}
                />
                <ToggleList>
                  <Toggle
                    of={protocol}
                    name="requireVersion"
                    update={update}
                    label="Require a protocol version"
                    description="Reject requests that declare no version."
                  />
                  <Toggle
                    of={protocol}
                    name="rejectUnknownMethods"
                    update={update}
                    label="Reject unknown methods"
                    description="Keep this off to let MCP extensions reach the server."
                  />
                </ToggleList>
              </>
            )}
          </OptionRow>
        )}
        {global && (
          <OptionRow
            icon={Gauge}
            title="Limits"
            description="Bound how much of the traffic is parsed."
            summary={
              limits &&
              (joinSummary(
                !!limits.maxRequestBytes &&
                  `${limits.maxRequestBytes.toLocaleString()} request bytes`,
                !!limits.maxStreamEventBytes &&
                  `${limits.maxStreamEventBytes.toLocaleString()} event bytes`,
              ) ||
                "Defaults")
            }
            isSet={!!limits}
            onAdd={() => {
              value.limits = CoreP.Service_Spec_Config_MCP_Limits.create();
              update();
            }}
            onRemove={() => {
              value.limits = undefined;
              update();
            }}
          >
            {limits && (
              <Grid>
                <NumberField
                  of={limits}
                  name="maxRequestBytes"
                  update={update}
                  label="Largest request"
                  suffix=" bytes"
                  max={16777216}
                  thousands
                />
                <NumberField
                  of={limits}
                  name="maxStreamEventBytes"
                  update={update}
                  label="Largest stream event"
                  description="Larger events are forwarded without inspection."
                  suffix=" bytes"
                  max={4194304}
                  thousands
                />
              </Grid>
            )}
          </OptionRow>
        )}
        <HeadersRow holder={value} update={update} />
        <PathRow holder={value} update={update} />
        {global && <CORSRow holder={value} update={update} />}
        <VisibilityRow
          isSet={!!visibility}
          summary={
            visibility &&
            (joinSummary(
              !visibility.disableRequestBody && "Request bodies",
              !visibility.disableResponseBody && "Response bodies",
              headerVisibilitySummary(visibility),
            ) ||
              "Bodies not recorded")
          }
          onAdd={() => {
            value.visibility = CoreP.Service_Spec_Config_MCP_Visibility.create();
            update();
          }}
          onRemove={() => {
            value.visibility = undefined;
            update();
          }}
        >
          {visibility && (
            <>
              <ToggleList>
                <Toggle
                  of={visibility}
                  name="disableRequestBody"
                  update={update}
                  label="Do not record request bodies"
                />
                <Toggle
                  of={visibility}
                  name="disableResponseBody"
                  update={update}
                  label="Do not record response bodies"
                />
              </ToggleList>
              <VisibilityHeaders value={visibility} update={update} />
            </>
          )}
        </VisibilityRow>
      </OptionList>

      <TransportCard holder={value} scope={scope} update={update}>
        {global && (
          <Toggle
            of={value}
            name="disableOriginCheck"
            update={update}
            label="Disable Origin validation"
            description="Accept browser requests from any Origin. Clients that send no Origin are always accepted."
          />
        )}
      </TransportCard>
    </>
  );
};
