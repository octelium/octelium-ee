import { Struct } from "@/apis/google/protobuf/struct";
import * as CoreP from "@/apis/corev1/corev1";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import {
  FormSection,
  Grid,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { Boxes, Braces, Cable, Power } from "lucide-react";
import * as React from "react";
import {
  CodeField,
  jsonError,
  NumberField,
  Toggle,
  Update,
} from "./fields";
import { Spec } from "./model";
import { ModeInfo } from "./modes";

const portHint = (spec: Spec) => {
  const type = spec.config?.upstream?.type;
  if (type?.oneofKind === "container") return "Inferred from the container";
  if (type?.oneofKind === "url" || type?.oneofKind === "loadbalance") {
    return "Inferred from the upstream";
  }
  return "Inferred from the mode";
};

const Attributes = (props: { spec: Spec; update: Update }) => {
  const { spec, update } = props;
  const [text, setText] = React.useState(() =>
    spec.attrs ? JSON.stringify(Struct.toJson(spec.attrs), null, 2) : "",
  );
  const error = jsonError(text, { object: true });
  return (
    <CodeField
      label="Attributes"
      description="A JSON object that Policies can read. Use camelCase keys."
      mode="json"
      value={text}
      error={error}
      onChange={(next) => {
        setText(next);
        if (!next.trim()) {
          spec.attrs = undefined;
          update();
        } else if (!jsonError(next, { object: true })) {
          spec.attrs = Struct.fromJson(JSON.parse(next));
          update();
        }
      }}
    />
  );
};

export const SettingsForm = (props: {
  spec: Spec;
  info: ModeInfo;
  update: Update;
}) => {
  const { spec, info, update } = props;
  const deployment = spec.deployment ?? CoreP.Service_Spec_Deployment.create();

  return (
    <>
      <FormSection
        icon={Cable}
        title="Listener"
        description="Where clients connect to this Service."
      >
        <NumberField
          of={spec}
          name="port"
          update={update}
          label="Port"
          placeholder={portHint(spec)}
          max={65535}
        />
        {info.listenerTLS && (
          <ToggleList>
            <Toggle
              of={spec}
              name="isTLS"
              update={update}
              label="Serve over TLS"
              description="Use the Namespace certificate. TLS has to be enabled on the Namespace."
            />
          </ToggleList>
        )}
      </FormSection>

      <FormSection
        icon={Boxes}
        title="Deployment"
        description="Where the Service runs and how many copies of it run."
      >
        <Grid>
          <SelectResource
            api="core"
            kind="Region"
            clearable
            label="Region"
            description="The default Region when not set."
            defaultValue={spec.region || undefined}
            onChange={(region) => {
              spec.region = region?.metadata?.name ?? "";
              update();
            }}
          />
          <NumberField
            of={deployment}
            name="replicas"
            update={() => {
              spec.deployment = deployment.replicas ? deployment : undefined;
              update();
            }}
            label="Replicas"
            description="How many Service proxies to deploy."
            placeholder="1"
            max={100}
          />
        </Grid>
      </FormSection>

      <FormSection
        icon={Braces}
        title="Attributes"
        description="Custom metadata for your authorization rules."
      >
        <Attributes spec={spec} update={update} />
      </FormSection>

      <FormSection
        icon={Power}
        title="Status"
        description="A disabled Service keeps its configuration but stops accepting requests."
      >
        <ToggleList>
          <Toggle
            of={spec}
            name="isDisabled"
            update={update}
            label="Disable this Service"
          />
        </ToggleList>
      </FormSection>
    </>
  );
};
