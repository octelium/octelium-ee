import * as CoreP from "@/apis/corev1/corev1";
import {
  ChoiceGroup,
  FormSection,
  Grid,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { TextInput } from "@mantine/core";
import { Server } from "lucide-react";
import * as React from "react";
import { ContainerEditor } from "./container";
import {
  ListField,
  Note,
  RemoveButton,
  Update,
  UserField,
} from "./fields";
import { Config, parseUpstreamURL, upstreamURLError } from "./model";
import { ModeInfo } from "./modes";

type Upstream = CoreP.Service_Spec_Config_Upstream;

type UpstreamKind = NonNullable<Upstream["type"]["oneofKind"]> | "embedded";

const createUpstreamType = (kind: UpstreamKind): Upstream["type"] => {
  switch (kind) {
    case "container":
      return {
        oneofKind: "container",
        container: CoreP.Service_Spec_Config_Upstream_Container.create(),
      };
    case "loadbalance":
      return {
        oneofKind: "loadbalance",
        loadbalance: CoreP.Service_Spec_Config_Upstream_Loadbalance.create({
          endpoints: [
            CoreP.Service_Spec_Config_Upstream_Loadbalance_Endpoint.create(),
          ],
        }),
      };
    default:
      return { oneofKind: "url", url: "" };
  }
};

const Endpoints = (props: {
  value: CoreP.Service_Spec_Config_Upstream_Loadbalance;
  info: ModeInfo;
  update: Update;
}) => {
  const rows = useListKeys();
  const { value, update } = props;
  const schemes = new Set(
    value.endpoints.map((item) => parseUpstreamURL(item.url)?.scheme),
  );
  return (
    <ListField
      label="Endpoints"
      description="Requests are spread randomly across these endpoints."
      addLabel="Add endpoint"
      count={value.endpoints.length}
      onAdd={() => {
        value.endpoints.push(
          CoreP.Service_Spec_Config_Upstream_Loadbalance_Endpoint.create(),
        );
        update();
      }}
    >
      {value.endpoints.map((endpoint, index) => (
        <div
          key={rows.keyAt("endpoints", index)}
          className="flex min-w-0 flex-wrap items-start gap-2"
        >
          <TextInput
            aria-label={`Endpoint ${index + 1} URL`}
            className="min-w-0 flex-[3_1_14rem]"
            placeholder={props.info.urlPlaceholder}
            value={endpoint.url}
            error={
              upstreamURLError(endpoint.url) ??
              (schemes.size > 1 && endpoint.url
                ? "Use the same scheme in all endpoints"
                : undefined)
            }
            onChange={(event) => {
              endpoint.url = event.currentTarget.value;
              update();
            }}
          />
          <div className="flex min-w-0 flex-[2_1_11rem] items-start gap-2">
            <div className="-mt-1 min-w-0 flex-1">
              <UserField
                value={endpoint.user}
                onChange={(name) => {
                  endpoint.user = name;
                  update();
                }}
              />
            </div>
            <RemoveButton
              label={`Remove endpoint ${index + 1}`}
              onClick={() => {
                rows.removeAt("endpoints", index);
                value.endpoints.splice(index, 1);
                update();
              }}
            />
          </div>
        </div>
      ))}
    </ListField>
  );
};

export const UpstreamEditor = (props: {
  config: Config;
  info: ModeInfo;
  update: Update;
  embedded?: {
    enabled: boolean;
    label: string;
    hint: string;
    note: string;
    onChange: (enabled: boolean) => void;
  };
}) => {
  const { config, info, update, embedded } = props;
  const upstream = config.upstream;
  const memory = React.useRef<Partial<Record<UpstreamKind, Upstream>>>({});
  const kind = embedded?.enabled ? "embedded" : upstream?.type.oneofKind;

  const change = (next: UpstreamKind) => {
    if (upstream && kind) {
      memory.current[kind] = CoreP.Service_Spec_Config_Upstream.clone(upstream);
    }
    embedded?.onChange(next === "embedded");
    if (next === "embedded") {
      config.upstream = undefined;
    } else {
      config.upstream =
        memory.current[next] ??
        CoreP.Service_Spec_Config_Upstream.create({
          type: createUpstreamType(next),
        });
    }
    update();
  };

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <ChoiceGroup<UpstreamKind>
        label="Source"
        hideLabel
        value={kind}
        options={[
          ...(embedded
            ? [
                {
                  value: "embedded" as const,
                  label: embedded.label,
                  hint: embedded.hint,
                },
              ]
            : []),
          {
            value: "url",
            label: "URL",
            hint: "A single address",
          },
          {
            value: "container",
            label: "Managed container",
            hint: "Deployed by the Cluster",
          },
          {
            value: "loadbalance",
            label: "Load balance",
            hint: "Several addresses",
          },
        ]}
        onChange={change}
      />

      {kind === "embedded" && embedded && <Note>{embedded.note}</Note>}

      {upstream?.type.oneofKind === "url" && (
        <Grid>
          <TextInput
            required
            label="URL"
            placeholder={info.urlPlaceholder}
            value={upstream.type.url}
            error={upstreamURLError(upstream.type.url)}
            onChange={(event) => {
              upstream.type = {
                oneofKind: "url",
                url: event.currentTarget.value,
              };
              update();
            }}
          />
          <UserField
            label="Served by User"
            value={upstream.user}
            onChange={(name) => {
              upstream.user = name;
              update();
            }}
          />
        </Grid>
      )}

      {upstream?.type.oneofKind === "url" &&
        (info.config === "llm" ? (
          <Note>
            The URL is the base URL of the API. Include the version segment,
            like https://openrouter.ai/api/v1.
          </Note>
        ) : info.config === "mcp" ? (
          <Note>
            The URL path replaces the request path. Without a path, the request
            path is proxied as it is.
          </Note>
        ) : (
          upstream.user && (
            <Note>
              Reached through any connected client of {upstream.user} that
              serves this Service.
            </Note>
          )
        ))}

      {upstream?.type.oneofKind === "container" && (
        <ContainerEditor container={upstream.type.container} update={update} />
      )}

      {upstream?.type.oneofKind === "loadbalance" && (
        <Endpoints
          value={upstream.type.loadbalance}
          info={info}
          update={update}
        />
      )}
    </div>
  );
};

export const UpstreamSection = (props: React.ComponentProps<typeof UpstreamEditor>) => (
  <FormSection
    icon={Server}
    title="Upstream"
    description="The resource that this Service protects."
  >
    <UpstreamEditor {...props} />
  </FormSection>
);
