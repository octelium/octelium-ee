import * as CoreP from "@/apis/corev1/corev1";
import { CELEditor, OPAEditor } from "@/components/Condition/Editor";
import {
  FormSection,
  Grid,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { Select } from "@mantine/core";
import { Layers, ListChecks, Server, SlidersHorizontal } from "lucide-react";
import * as React from "react";
import { ProtocolForm, hasUpstreamTLS } from "./ConfigForms";
import {
  AddButton,
  FieldText,
  ItemCard,
  joinSummary,
  Note,
  OptionList,
  OptionRow,
  plural,
  Segments,
  TextField,
  Update,
} from "./fields";
import { Config, createConfigType, nameError, Spec } from "./model";
import { ModeInfo } from "./modes";
import { ConditionField } from "./plugins";
import { TLSRow } from "./tls";
import { UpstreamEditor } from "./upstream";

type DynamicConfig = CoreP.Service_Spec_DynamicConfig;
type Rule = CoreP.Service_Spec_DynamicConfig_Rule;

const upstreamSummary = (config: Config) => {
  const type = config.upstream?.type;
  switch (type?.oneofKind) {
    case "url":
      return type.url || "No URL";
    case "container":
      return type.container.image || "Managed container";
    case "loadbalance":
      return plural(type.loadbalance.endpoints.length, "endpoint");
    default:
      return undefined;
  }
};

const NamedConfig = (props: {
  config: Config;
  index: number;
  configs: Config[];
  info: ModeInfo;
  update: Update;
}) => {
  const { config, configs, info, index, update } = props;
  const parents = [
    "default",
    ...configs
      .filter((_, item) => item !== index)
      .map((item) => item.name)
      .filter(Boolean),
  ];
  const duplicate = configs.some(
    (item, itemIndex) => itemIndex !== index && item.name === config.name,
  );

  return (
    <>
      <Grid>
        <TextField
          of={config}
          name="name"
          update={update}
          label="Name"
          placeholder="canary"
          required
          error={(value) =>
            value === "default"
              ? "default is the name of the main Configuration"
              : duplicate
                ? "This name is already used"
                : nameError(value)
          }
        />
        <Select
          label="Inherits from"
          description="Settings left out here come from this Configuration."
          placeholder="Nothing"
          clearable
          data={parents}
          value={config.parent || null}
          onChange={(value) => {
            config.parent = value ?? "";
            update();
          }}
        />
      </Grid>
      <OptionList>
        <OptionRow
          icon={Server}
          title="Upstream"
          description={
            config.parent
              ? "Inherited unless you set one here."
              : "Where the matching requests go."
          }
          summary={upstreamSummary(config)}
          isSet={!!config.upstream}
          onAdd={() => {
            config.upstream = CoreP.Service_Spec_Config_Upstream.create({
              type: { oneofKind: "url", url: "" },
            });
            update();
          }}
          onRemove={() => {
            config.upstream = undefined;
            update();
          }}
        >
          <UpstreamEditor config={config} info={info} update={update} />
        </OptionRow>
        {info.config && (
          <OptionRow
            icon={SlidersHorizontal}
            title={`${info.tab ?? info.label} settings`}
            description={
              config.parent
                ? "Inherited unless you customize them here."
                : "Override the protocol settings for the matching requests."
            }
            summary="Customized"
            isSet={config.type.oneofKind !== undefined}
            onAdd={() => {
              config.type = createConfigType(info.config);
              update();
            }}
            onRemove={() => {
              config.type = { oneofKind: undefined };
              update();
            }}
          >
            <ProtocolForm
              config={config}
              info={info}
              scope="named"
              update={update}
            />
          </OptionRow>
        )}
        {hasUpstreamTLS(config, info) && (
          <TLSRow config={config} support={info.upstreamTLS} update={update} />
        )}
      </OptionList>
    </>
  );
};

const ruleSummary = (rule: Rule) => {
  switch (rule.type.oneofKind) {
    case "configName":
      return rule.type.configName
        ? `Use ${rule.type.configName}`
        : "No Configuration selected";
    case "eval":
      return "Computed with CEL";
    case "opa":
      return "Computed with Rego";
    default:
      return undefined;
  }
};

const conditionText = (rule: Rule) => {
  const type = rule.condition?.type;
  if (type?.oneofKind === "match") return type.match;
  if (type?.oneofKind === "matchAny") return "Every request";
  return undefined;
};

const RuleForm = (props: {
  rule: Rule;
  names: string[];
  update: Update;
}) => {
  const { rule, names, update } = props;
  const type = rule.type;
  return (
    <>
      <ConditionField
        label="Condition"
        description="The first rule that matches a request decides."
        value={rule.condition}
        onChange={(condition) => {
          rule.condition = condition;
          update();
        }}
      />
      <Segments
        label="Then"
        value={type.oneofKind}
        options={[
          { value: "configName", label: "Use a Configuration" },
          { value: "eval", label: "Compute with CEL" },
          { value: "opa", label: "Compute with Rego" },
        ]}
        onChange={(next) => {
          rule.type =
            next === "eval"
              ? { oneofKind: "eval", eval: "" }
              : next === "opa"
                ? { oneofKind: "opa", opa: "" }
                : { oneofKind: "configName", configName: "" };
          update();
        }}
      />
      {type.oneofKind === "configName" && (
        <Select
          label="Configuration"
          required
          placeholder={names.length ? "Select a Configuration" : "Name a Configuration first"}
          data={names}
          value={type.configName || null}
          onChange={(value) => {
            type.configName = value ?? "";
            update();
          }}
        />
      )}
      {type.oneofKind === "eval" && (
        <div className="flex min-w-0 flex-col gap-1">
          <FieldText
            label="CEL expression"
            description="Returns an object in the shape of a Service Configuration."
          />
          <CELEditor
            label="Rule CEL"
            exp={type.eval}
            onChange={(value) => {
              type.eval = value;
              update();
            }}
          />
        </div>
      )}
      {type.oneofKind === "opa" && (
        <div className="flex min-w-0 flex-col gap-1">
          <FieldText
            label="Rego policy"
            description="Sets result to an object in the shape of a Service Configuration."
          />
          <OPAEditor
            label="Rule Rego"
            exp={type.opa}
            onChange={(value) => {
              type.opa = value;
              update();
            }}
          />
        </div>
      )}
    </>
  );
};

export const DynamicForm = (props: {
  spec: Spec;
  info: ModeInfo;
  update: Update;
}) => {
  const { spec, info, update } = props;
  const rows = useListKeys();
  const added = React.useRef(new Set<string>());
  const dynamic: DynamicConfig =
    spec.dynamicConfig ?? CoreP.Service_Spec_DynamicConfig.create();
  const commit = () => {
    spec.dynamicConfig =
      dynamic.configs.length + dynamic.rules.length > 0 ? dynamic : undefined;
    update();
  };
  const names = [
    ...(spec.config ? ["default"] : []),
    ...dynamic.configs.map((item) => item.name).filter(Boolean),
  ];

  return (
    <>
      <Note>
        Rules pick a Configuration for each request, based on who is asking and
        how. When no rule matches, the main Configuration is used.
      </Note>

      <FormSection
        icon={Layers}
        title="Named Configurations"
        description="Alternatives to the main Configuration, such as a canary upstream."
        aside={
          <AddButton
            label="Add Configuration"
            onClick={() => {
              dynamic.configs.push(CoreP.Service_Spec_Config.create());
              added.current.add(
                rows.keyAt("configs", dynamic.configs.length - 1),
              );
              commit();
            }}
          />
        }
      >
        {dynamic.configs.length === 0 ? (
          <Note>No named Configurations yet.</Note>
        ) : (
          <div className="flex min-w-0 flex-col gap-2">
            {dynamic.configs.map((config, index) => {
              const key = rows.keyAt("configs", index);
              return (
                <ItemCard
                  key={key}
                  title={config.name || `Configuration ${index + 1}`}
                  subtitle={joinSummary(
                    config.parent && `Inherits from ${config.parent}`,
                    upstreamSummary(config),
                  )}
                  defaultOpen={added.current.has(key)}
                  removeLabel={`Remove ${config.name || `Configuration ${index + 1}`}`}
                  onRemove={() => {
                    const name = config.name;
                    rows.removeAt("configs", index);
                    dynamic.configs.splice(index, 1);
                    for (const rule of dynamic.rules) {
                      if (
                        rule.type.oneofKind === "configName" &&
                        rule.type.configName === name
                      ) {
                        rule.type.configName = "";
                      }
                    }
                    commit();
                  }}
                >
                  <NamedConfig
                    config={config}
                    index={index}
                    configs={dynamic.configs}
                    info={info}
                    update={commit}
                  />
                </ItemCard>
              );
            })}
          </div>
        )}
      </FormSection>

      <FormSection
        icon={ListChecks}
        title="Rules"
        description="Evaluated in order for every request."
        aside={
          <AddButton
            label="Add rule"
            onClick={() => {
              dynamic.rules.push(
                CoreP.Service_Spec_DynamicConfig_Rule.create({
                  condition: CoreP.Condition.create({
                    type: { oneofKind: "match", match: "" },
                  }),
                  type: { oneofKind: "configName", configName: "" },
                }),
              );
              added.current.add(rows.keyAt("rules", dynamic.rules.length - 1));
              commit();
            }}
          />
        }
      >
        {dynamic.rules.length === 0 ? (
          <Note>No rules yet, so every request uses the main Configuration.</Note>
        ) : (
          <div className="flex min-w-0 flex-col gap-2">
            {dynamic.rules.map((rule, index) => {
              const key = rows.keyAt("rules", index);
              const move = (to: number) => {
                rows.move("rules", index, to);
                const [item] = dynamic.rules.splice(index, 1);
                dynamic.rules.splice(to, 0, item);
                commit();
              };
              return (
                <ItemCard
                  key={key}
                  title={`Rule ${index + 1}`}
                  subtitle={joinSummary(ruleSummary(rule), conditionText(rule))}
                  defaultOpen={added.current.has(key)}
                  removeLabel={`Remove rule ${index + 1}`}
                  onRemove={() => {
                    rows.removeAt("rules", index);
                    dynamic.rules.splice(index, 1);
                    commit();
                  }}
                  onMoveUp={index > 0 ? () => move(index - 1) : undefined}
                  onMoveDown={
                    index < dynamic.rules.length - 1
                      ? () => move(index + 1)
                      : undefined
                  }
                >
                  <RuleForm rule={rule} names={names} update={commit} />
                </ItemCard>
              );
            })}
          </div>
        )}
      </FormSection>
    </>
  );
};
