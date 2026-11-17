import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { Struct } from "@/apis/google/protobuf/struct";
import EditItem from "@/components/EditItem";
import { Select, Switch, Textarea, TextInput } from "@mantine/core";
import * as React from "react";
import { ComputeLimit } from "./fields";

type Image = CordiumP.Workspace_Spec_Image;
type ImageKind = "registry" | "dockerfile" | "git";

const makeImage = (kind: ImageKind): Image => {
  switch (kind) {
    case "dockerfile":
      return CordiumP.Workspace_Spec_Image.create({ type: { oneofKind: "dockerfile", dockerfile: { type: { oneofKind: "inline", inline: "" } } } });
    case "git":
      return CordiumP.Workspace_Spec_Image.create({ type: { oneofKind: "git", git: CordiumP.Workspace_Spec_Image_Git.create() } });
    default:
      return CordiumP.Workspace_Spec_Image.create({ type: { oneofKind: "registry", registry: { url: "" } } });
  }
};

const AgentImage = (props: { value: Image; onChange: (value: Image) => void }) => {
  const memory = React.useRef<Partial<Record<string, Image>>>({});
  const dockerfiles = React.useRef<Partial<Record<string, CordiumP.Workspace_Spec_Image_Dockerfile>>>({});
  const type = props.value.type;
  const update = (mutate: (image: Image) => void) => {
    const next = CordiumP.Workspace_Spec_Image.clone(props.value);
    mutate(next);
    props.onChange(next);
  };

  return (
    <div className="space-y-4">
      <Select
        label="Image source"
        allowDeselect={false}
        data={[{ value: "registry", label: "Container registry" }, { value: "dockerfile", label: "Dockerfile" }, { value: "git", label: "Git repository" }]}
        value={type.oneofKind ?? null}
        onChange={(value) => {
          if (!value) return;
          if (type.oneofKind) memory.current[type.oneofKind] = CordiumP.Workspace_Spec_Image.clone(props.value);
          props.onChange(memory.current[value] ?? makeImage(value as ImageKind));
        }}
      />
      {type.oneofKind === "registry" && (
        <TextInput label="Container image" required description="Public registry image used for agent Workspaces." placeholder="ghcr.io/example/agent:latest" value={type.registry.url} onChange={(event) => update((next) => { if (next.type.oneofKind === "registry") next.type.registry.url = event.currentTarget.value; })} />
      )}
      {type.oneofKind === "dockerfile" && (
        <>
          <Select
            label="Dockerfile source"
            allowDeselect={false}
            data={[{ value: "inline", label: "Inline" }, { value: "url", label: "HTTPS URL" }]}
            value={type.dockerfile.type.oneofKind ?? null}
            onChange={(value) => {
              if (!value) return;
              if (type.dockerfile.type.oneofKind) dockerfiles.current[type.dockerfile.type.oneofKind] = CordiumP.Workspace_Spec_Image_Dockerfile.clone(type.dockerfile);
              update((next) => {
                if (next.type.oneofKind !== "dockerfile") return;
                next.type.dockerfile = dockerfiles.current[value] ?? CordiumP.Workspace_Spec_Image_Dockerfile.create({ type: value === "url" ? { oneofKind: "url", url: "" } : { oneofKind: "inline", inline: "" } });
              });
            }}
          />
          {type.dockerfile.type.oneofKind === "inline" && (
            <Textarea label="Dockerfile" required autosize minRows={5} maxRows={16} styles={{ input: { fontFamily: "var(--font-mono)" } }} value={type.dockerfile.type.inline} onChange={(event) => update((next) => { if (next.type.oneofKind === "dockerfile") next.type.dockerfile.type = { oneofKind: "inline", inline: event.currentTarget.value }; })} />
          )}
          {type.dockerfile.type.oneofKind === "url" && (
            <TextInput label="Dockerfile URL" required placeholder="https://example.com/Dockerfile" value={type.dockerfile.type.url} onChange={(event) => update((next) => { if (next.type.oneofKind === "dockerfile") next.type.dockerfile.type = { oneofKind: "url", url: event.currentTarget.value }; })} />
          )}
        </>
      )}
      {type.oneofKind === "git" && (
        <div className="grid gap-4 md:grid-cols-2">
          <TextInput label="Git repository URL" required placeholder="https://github.com/example/agent.git" value={type.git.url} onChange={(event) => update((next) => { if (next.type.oneofKind === "git") next.type.git.url = event.currentTarget.value; })} />
          <TextInput label="Git checkout" description="Optional branch, tag or commit." value={type.git.checkout} onChange={(event) => update((next) => { if (next.type.oneofKind === "git") next.type.git.checkout = event.currentTarget.value; })} />
          <TextInput label="Dockerfile path" placeholder="Dockerfile" value={type.git.dockerfile} onChange={(event) => update((next) => { if (next.type.oneofKind === "git") next.type.git.dockerfile = event.currentTarget.value; })} />
          <TextInput label="Build context" placeholder="." value={type.git.context} onChange={(event) => update((next) => { if (next.type.oneofKind === "git") next.type.git.context = event.currentTarget.value; })} />
        </div>
      )}
    </div>
  );
};

const AgentConfig = (props: {
  value?: Struct;
  onChange: (value?: Struct) => void;
}) => {
  const [text, setText] = React.useState(() => props.value ? JSON.stringify(Struct.toJson(props.value), null, 2) : "");
  const [error, setError] = React.useState<string>();

  return (
    <Textarea
      label="Additional configuration (JSON)"
      description="A JSON object merged into the generated agent config.json."
      placeholder="{}"
      autosize
      minRows={5}
      maxRows={16}
      styles={{ input: { fontFamily: "var(--font-mono)" } }}
      value={text}
      error={error}
      onChange={(event) => {
        const value = event.currentTarget.value;
        setText(value);
        if (!value.trim()) {
          setError(undefined);
          props.onChange(undefined);
          return;
        }
        try {
          const parsed = JSON.parse(value);
          if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
            setError("Enter a JSON object");
            return;
          }
          props.onChange(Struct.fromJson(parsed));
          setError(undefined);
        } catch {
          setError("Enter valid JSON");
        }
      }}
    />
  );
};

const Agent = (props: {
  value: CordiumP.ClusterConfig_Spec_Agent;
  onChange: (value: CordiumP.ClusterConfig_Spec_Agent) => void;
}) => {
  const { value } = props;
  const update = (mutate: (next: CordiumP.ClusterConfig_Spec_Agent) => void) => {
    const next = CordiumP.ClusterConfig_Spec_Agent.clone(value);
    mutate(next);
    props.onChange(next);
  };

  return (
    <div className="space-y-4">
      <Switch label="Disable the Cordium AI agent" description="Prevent Users from initializing their agent. Existing agent Workspaces are kept." checked={value.isDisabled} onChange={(event) => update((next) => { next.isDisabled = event.currentTarget.checked; })} />
      <TextInput label="Agent version" description="Package version or npm dist-tag. Leave empty to use the version bundled with the Cluster." placeholder="Bundled version" value={value.version} onChange={(event) => update((next) => { next.version = event.currentTarget.value; })} />

      <EditItem
        title="Default LLM"
        description="Use an LLM-mode Octelium Service, or leave unconfigured so Users can sign in with their own subscriptions."
        obj={value.llm}
        onSet={() => update((next) => { next.llm = CordiumP.ClusterConfig_Spec_Agent_LLM.create(); })}
        onUnset={() => update((next) => { next.llm = undefined; })}
      >
        {value.llm && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput label="LLM Service" description="Name of an LLM-mode Octelium Service." placeholder="llm.default" value={value.llm.service} onChange={(event) => update((next) => { next.llm!.service = event.currentTarget.value; })} />
            <TextInput label="Model" description="Leave empty to discover the model from the Service." placeholder="Discover from Service" value={value.llm.model} onChange={(event) => update((next) => { next.llm!.model = event.currentTarget.value; })} />
          </div>
        )}
      </EditItem>

      <EditItem
        title="Agent image"
        description="Override the default Cordium image used for agent Workspaces."
        obj={value.image}
        onSet={() => update((next) => { next.image = makeImage("registry"); })}
        onUnset={() => update((next) => { next.image = undefined; })}
      >
        {value.image && <AgentImage value={value.image} onChange={(image) => update((next) => { next.image = image; })} />}
      </EditItem>

      <ComputeLimit title="Agent resources" description="Compute resources allocated to agent Workspaces." value={value.limit} onChange={(limit) => update((next) => { next.limit = limit; })} />

      <AgentConfig value={value.config} onChange={(config) => update((next) => { next.config = config; })} />
    </div>
  );
};

export default Agent;
