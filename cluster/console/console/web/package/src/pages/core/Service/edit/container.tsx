import * as CoreP from "@/apis/corev1/corev1";
import {
  Grid,
  KeyValueList,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { Checkbox, Select, TextInput } from "@mantine/core";
import {
  Activity,
  Cpu,
  HardDrive,
  KeyRound,
  ShieldCheck,
  SquareTerminal,
  Variable,
} from "lucide-react";
import * as React from "react";
import {
  joinSummary,
  ListField,
  listSummary,
  NumberField,
  OptionList,
  OptionRow,
  plural,
  RemoveButton,
  SecretField,
  SecretRefField,
  Segments,
  StringList,
  TagsField,
  TextField,
  Toggle,
  Update,
} from "./fields";
import { isNameValid, nameError } from "./model";

type Container = CoreP.Service_Spec_Config_Upstream_Container;
type Env = CoreP.Service_Spec_Config_Upstream_Container_Env;
type Probe = CoreP.Service_Spec_Config_Upstream_Container_Probe;
type Volume = CoreP.Service_Spec_Config_Upstream_Container_Volume;

const LINUX_CAPABILITIES = [
  "ALL",
  "AUDIT_WRITE",
  "CHOWN",
  "DAC_OVERRIDE",
  "FOWNER",
  "FSETID",
  "KILL",
  "MKNOD",
  "NET_ADMIN",
  "NET_BIND_SERVICE",
  "NET_RAW",
  "SETFCAP",
  "SETGID",
  "SETPCAP",
  "SETUID",
  "SYS_ADMIN",
  "SYS_CHROOT",
  "SYS_TIME",
];

const capabilityError = (values: string[]) =>
  values.find((item) => !/^(ALL|[A-Z][A-Z0-9_]{0,29})$/.test(item))
    ? "Use uppercase capability names, like NET_ADMIN"
    : undefined;

const envKeyError = (value: string) =>
  /^[A-Za-z_][A-Za-z0-9_.-]*$/.test(value) ? undefined : "Invalid name";

const EnvRow = (props: {
  env: Env;
  index: number;
  update: Update;
  onRemove: () => void;
}) => {
  const { env, index, update } = props;
  const memory = React.useRef<Partial<Record<string, Env["type"]>>>({});
  const kind = env.type.oneofKind ?? "value";
  return (
    <div className="flex min-w-0 flex-wrap items-start gap-2">
      <TextInput
        aria-label={`Variable ${index + 1} name`}
        className="min-w-0 flex-[1_1_9rem]"
        classNames={{ input: "font-mono" }}
        placeholder="NAME"
        value={env.name}
        error={env.name ? envKeyError(env.name) : undefined}
        onChange={(event) => {
          env.name = event.currentTarget.value;
          update();
        }}
      />
      <div className="flex h-9 shrink-0 items-center">
        <Segments
          label={`Variable ${index + 1} source`}
          hideLabel
          value={kind}
          options={[
            { value: "value", label: "Value" },
            { value: "fromSecret", label: "Secret" },
            { value: "kubernetesSecretRef", label: "K8s" },
          ]}
          onChange={(next) => {
            memory.current[kind] = structuredClone(env.type);
            env.type =
              memory.current[next] ??
              (next === "fromSecret"
                ? { oneofKind: "fromSecret", fromSecret: "" }
                : next === "kubernetesSecretRef"
                  ? {
                      oneofKind: "kubernetesSecretRef",
                      kubernetesSecretRef:
                        CoreP.Service_Spec_Config_Upstream_Container_Env_KubernetesSecretRef.create(),
                    }
                  : { oneofKind: "value", value: "" });
            update();
          }}
        />
      </div>
      <div className="flex min-w-0 flex-[2_1_13rem] items-start gap-2">
        {env.type.oneofKind === "value" && (
          <TextInput
            aria-label={`Variable ${index + 1} value`}
            className="min-w-0 flex-1"
            placeholder="value"
            value={env.type.value}
            onChange={(event) => {
              env.type = { oneofKind: "value", value: event.currentTarget.value };
              update();
            }}
          />
        )}
        {env.type.oneofKind === "fromSecret" && (
          <div className="-mt-1 min-w-0 flex-1">
            <SecretField
              required
              value={env.type.fromSecret}
              onChange={(name) => {
                env.type = { oneofKind: "fromSecret", fromSecret: name };
                update();
              }}
            />
          </div>
        )}
        {env.type.oneofKind === "kubernetesSecretRef" && (
          <>
            <TextInput
              aria-label={`Variable ${index + 1} Kubernetes Secret`}
              className="min-w-0 flex-1"
              placeholder="k8s-secret"
              value={env.type.kubernetesSecretRef.name}
              onChange={(event) => {
                if (env.type.oneofKind === "kubernetesSecretRef") {
                  env.type.kubernetesSecretRef.name = event.currentTarget.value;
                }
                update();
              }}
            />
            <TextInput
              aria-label={`Variable ${index + 1} Kubernetes Secret key`}
              className="min-w-0 flex-1"
              placeholder="data-key"
              value={env.type.kubernetesSecretRef.key}
              onChange={(event) => {
                if (env.type.oneofKind === "kubernetesSecretRef") {
                  env.type.kubernetesSecretRef.key = event.currentTarget.value;
                }
                update();
              }}
            />
          </>
        )}
        <RemoveButton
          label={`Remove variable ${index + 1}`}
          onClick={props.onRemove}
        />
      </div>
    </div>
  );
};

const EnvList = (props: { container: Container; update: Update }) => {
  const rows = useListKeys();
  const { container, update } = props;
  return (
    <ListField
      label="Variables"
      description="Secrets are read when the container starts. K8s reads a Kubernetes Secret in the octelium namespace."
      addLabel="Add variable"
      count={container.env.length}
      onAdd={() => {
        container.env.push(
          CoreP.Service_Spec_Config_Upstream_Container_Env.create({
            type: { oneofKind: "value", value: "" },
          }),
        );
        update();
      }}
    >
      {container.env.map((env, index) => (
        <EnvRow
          key={rows.keyAt("env", index)}
          env={env}
          index={index}
          update={update}
          onRemove={() => {
            rows.removeAt("env", index);
            container.env.splice(index, 1);
            update();
          }}
        />
      ))}
    </ListField>
  );
};

const ExtendedResources = (props: {
  limit: CoreP.Service_Spec_Config_Upstream_Container_ResourceLimit;
  update: Update;
}) => {
  const entries = React.useRef<{ key: string; value: string }[]>(
    Object.entries(props.limit.ext).map(([key, value]) => ({ key, value })),
  );
  const commit = () => {
    props.limit.ext = Object.fromEntries(
      entries.current
        .filter((entry) => entry.key)
        .map((entry) => [entry.key, entry.value]),
    );
    props.update();
  };
  return (
    <KeyValueList
      label="Extended resources"
      description="Custom limits such as GPUs."
      items={entries.current}
      make={() => ({ key: "", value: "" })}
      addLabel="Add resource"
      keyPlaceholder="nvidia.com/gpu"
      valuePlaceholder="1"
      onUpdate={commit}
    />
  );
};

const VolumeRow = (props: {
  volume: Volume;
  index: number;
  update: Update;
  onRemove: () => void;
}) => {
  const { volume, index, update } = props;
  return (
    <div className="flex min-w-0 flex-wrap items-start gap-2">
      <TextInput
        aria-label={`Volume ${index + 1} name`}
        className="min-w-0 flex-[1_1_9rem]"
        placeholder="volume-name"
        value={volume.name}
        error={volume.name ? nameError(volume.name) : undefined}
        onChange={(event) => {
          volume.name = event.currentTarget.value;
          update();
        }}
      />
      <div className="flex h-9 shrink-0 items-center">
        <Segments
          label={`Volume ${index + 1} type`}
          hideLabel
          value={volume.type.oneofKind}
          options={[
            { value: "emptyDir", label: "Empty dir" },
            { value: "persistentVolumeClaim", label: "PVC" },
          ]}
          onChange={(next) => {
            volume.type =
              next === "persistentVolumeClaim"
                ? {
                    oneofKind: "persistentVolumeClaim",
                    persistentVolumeClaim:
                      CoreP.Service_Spec_Config_Upstream_Container_Volume_PersistentVolumeClaim.create(),
                  }
                : {
                    oneofKind: "emptyDir",
                    emptyDir:
                      CoreP.Service_Spec_Config_Upstream_Container_Volume_EmptyDir.create(),
                  };
            update();
          }}
        />
      </div>
      <div className="flex min-w-0 flex-[2_1_11rem] items-start gap-2">
        {volume.type.oneofKind === "persistentVolumeClaim" && (
          <TextInput
            aria-label={`Volume ${index + 1} claim name`}
            className="min-w-0 flex-1"
            placeholder="claim-name"
            value={volume.type.persistentVolumeClaim.name}
            error={
              volume.type.persistentVolumeClaim.name
                ? nameError(volume.type.persistentVolumeClaim.name)
                : undefined
            }
            onChange={(event) => {
              if (volume.type.oneofKind === "persistentVolumeClaim") {
                volume.type.persistentVolumeClaim.name =
                  event.currentTarget.value;
              }
              update();
            }}
          />
        )}
        {volume.type.oneofKind === "emptyDir" && (
          <TextInput
            aria-label={`Volume ${index + 1} size limit`}
            className="min-w-0 flex-1"
            placeholder="No size limit"
            inputMode="numeric"
            rightSection={<span className="text-micro text-slate-500">MB</span>}
            value={volume.type.emptyDir.sizeLimitMegabytes || ""}
            onChange={(event) => {
              if (volume.type.oneofKind === "emptyDir") {
                volume.type.emptyDir.sizeLimitMegabytes =
                  Number(event.currentTarget.value.replace(/\D/g, "")) || 0;
              }
              update();
            }}
          />
        )}
        <RemoveButton
          label={`Remove volume ${index + 1}`}
          onClick={props.onRemove}
        />
      </div>
    </div>
  );
};

const Volumes = (props: { container: Container; update: Update }) => {
  const rows = useListKeys();
  const { container, update } = props;
  const names = container.volumes.map((item) => item.name).filter(isNameValid);

  return (
    <>
      <ListField
        label="Volumes"
        description="Storage that the container can mount."
        addLabel="Add volume"
        count={container.volumes.length}
        onAdd={() => {
          container.volumes.push(
            CoreP.Service_Spec_Config_Upstream_Container_Volume.create({
              type: {
                oneofKind: "emptyDir",
                emptyDir:
                  CoreP.Service_Spec_Config_Upstream_Container_Volume_EmptyDir.create(),
              },
            }),
          );
          update();
        }}
      >
        {container.volumes.map((volume, index) => (
          <VolumeRow
            key={rows.keyAt("volumes", index)}
            volume={volume}
            index={index}
            update={update}
            onRemove={() => {
              const removed = volume.name;
              rows.removeAt("volumes", index);
              container.volumes.splice(index, 1);
              container.volumeMounts = container.volumeMounts.filter(
                (mount) => mount.name !== removed || !removed,
              );
              update();
            }}
          />
        ))}
      </ListField>
      <ListField
        label="Mounts"
        description="Where each volume appears inside the container."
        addLabel="Add mount"
        count={container.volumeMounts.length}
        onAdd={() => {
          container.volumeMounts.push(
            CoreP.Service_Spec_Config_Upstream_Container_VolumeMount.create({
              name: names[0] ?? "",
            }),
          );
          update();
        }}
      >
        {container.volumeMounts.map((mount, index) => (
          <div
            key={rows.keyAt("mounts", index)}
            className="flex min-w-0 flex-wrap items-start gap-2"
          >
            <Select
              aria-label={`Mount ${index + 1} volume`}
              className="min-w-0 flex-[1_1_8rem]"
              placeholder="Volume"
              data={names}
              value={mount.name || null}
              onChange={(value) => {
                mount.name = value ?? "";
                update();
              }}
            />
            <TextInput
              aria-label={`Mount ${index + 1} path`}
              className="min-w-0 flex-[2_1_9rem]"
              classNames={{ input: "font-mono" }}
              placeholder="/data"
              value={mount.mountPath}
              error={
                mount.mountPath && !mount.mountPath.startsWith("/")
                  ? "Use an absolute path"
                  : undefined
              }
              onChange={(event) => {
                mount.mountPath = event.currentTarget.value;
                update();
              }}
            />
            <TextInput
              aria-label={`Mount ${index + 1} sub path`}
              className="min-w-0 flex-[1_1_7rem]"
              classNames={{ input: "font-mono" }}
              placeholder="Sub path"
              value={mount.subPath}
              error={
                mount.subPath.startsWith("/") ||
                mount.subPath.split("/").includes("..")
                  ? "Use a relative path"
                  : undefined
              }
              onChange={(event) => {
                mount.subPath = event.currentTarget.value;
                update();
              }}
            />
            <div className="flex h-9 shrink-0 items-center gap-2">
              <Checkbox
                size="xs"
                label="Read-only"
                checked={mount.readOnly}
                onChange={(event) => {
                  mount.readOnly = event.currentTarget.checked;
                  update();
                }}
              />
              <RemoveButton
                label={`Remove mount ${index + 1}`}
                onClick={() => {
                  rows.removeAt("mounts", index);
                  container.volumeMounts.splice(index, 1);
                  update();
                }}
              />
            </div>
          </div>
        ))}
      </ListField>
    </>
  );
};

const createProbe = () =>
  CoreP.Service_Spec_Config_Upstream_Container_Probe.create({
    type: {
      oneofKind: "httpGet",
      httpGet: CoreP.Service_Spec_Config_Upstream_Container_Probe_HTTPGet.create(
        { path: "/" },
      ),
    },
  });

const probeSummary = (probe: Probe) => {
  const interval = probe.periodSeconds
    ? `every ${probe.periodSeconds}s`
    : undefined;
  switch (probe.type.oneofKind) {
    case "httpGet":
      return joinSummary(
        `HTTP ${probe.type.httpGet.path} on ${probe.type.httpGet.port || "?"}`,
        interval,
      );
    case "tcpSocket":
      return joinSummary(`TCP on ${probe.type.tcpSocket.port || "?"}`, interval);
    case "grpc":
      return joinSummary(`gRPC on ${probe.type.grpc.port || "?"}`, interval);
    default:
      return undefined;
  }
};

const ProbeRow = (props: {
  title: string;
  description: string;
  liveness?: boolean;
  container: Container;
  field: "livenessProbe" | "readinessProbe";
  update: Update;
}) => {
  const { container, field, update } = props;
  const probe = container[field];
  const memory = React.useRef<Partial<Record<string, Probe["type"]>>>({});

  return (
    <OptionRow
      icon={Activity}
      title={props.title}
      description={props.description}
      summary={probe && probeSummary(probe)}
      isSet={!!probe}
      onAdd={() => {
        const next = createProbe();
        if (next.type.oneofKind === "httpGet") {
          next.type.httpGet.port = container.port;
        }
        container[field] = next;
        update();
      }}
      onRemove={() => {
        container[field] = undefined;
        update();
      }}
    >
      {probe && (
        <>
          <Segments
            label="Check"
            value={probe.type.oneofKind}
            options={[
              { value: "httpGet", label: "HTTP GET" },
              { value: "tcpSocket", label: "TCP socket" },
              { value: "grpc", label: "gRPC" },
            ]}
            onChange={(next) => {
              const kind = probe.type.oneofKind;
              if (kind) memory.current[kind] = structuredClone(probe.type);
              probe.type =
                memory.current[next] ??
                (next === "tcpSocket"
                  ? {
                      oneofKind: "tcpSocket",
                      tcpSocket:
                        CoreP.Service_Spec_Config_Upstream_Container_Probe_TCPSocket.create(
                          { port: container.port },
                        ),
                    }
                  : next === "grpc"
                    ? {
                        oneofKind: "grpc",
                        grpc: CoreP.Service_Spec_Config_Upstream_Container_Probe_GRPC.create(
                          { port: container.port },
                        ),
                      }
                    : {
                        oneofKind: "httpGet",
                        httpGet:
                          CoreP.Service_Spec_Config_Upstream_Container_Probe_HTTPGet.create(
                            { path: "/", port: container.port },
                          ),
                      });
              update();
            }}
          />
          {probe.type.oneofKind === "httpGet" && (
            <Grid>
              <TextField
                of={probe.type.httpGet}
                name="path"
                update={update}
                label="Path"
                placeholder="/healthz"
                required
                mono
                error={(value) =>
                  value.startsWith("/") ? undefined : "Start the path with /"
                }
              />
              <NumberField
                of={probe.type.httpGet}
                name="port"
                update={update}
                label="Port"
                placeholder="8080"
                required
                max={65535}
              />
            </Grid>
          )}
          {probe.type.oneofKind === "tcpSocket" && (
            <NumberField
              of={probe.type.tcpSocket}
              name="port"
              update={update}
              label="Port"
              placeholder="8080"
              required
              max={65535}
            />
          )}
          {probe.type.oneofKind === "grpc" && (
            <NumberField
              of={probe.type.grpc}
              name="port"
              update={update}
              label="Port"
              placeholder="8080"
              required
              max={65535}
            />
          )}
          <div className="grid grid-cols-2 gap-4 @xl:grid-cols-3 @3xl:grid-cols-5">
            <NumberField
              of={probe}
              name="initialDelaySeconds"
              update={update}
              label="Initial delay"
              suffix=" s"
            />
            <NumberField
              of={probe}
              name="periodSeconds"
              update={update}
              label="Period"
              suffix=" s"
            />
            <NumberField
              of={probe}
              name="timeoutSeconds"
              update={update}
              label="Timeout"
              suffix=" s"
            />
            {!props.liveness && (
              <NumberField
                of={probe}
                name="successThreshold"
                update={update}
                label="Successes"
              />
            )}
            <NumberField
              of={probe}
              name="failureThreshold"
              update={update}
              label="Failures"
            />
          </div>
        </>
      )}
    </OptionRow>
  );
};

const resourceSummary = (
  limit: CoreP.Service_Spec_Config_Upstream_Container_ResourceLimit,
) =>
  joinSummary(
    !!limit.cpu?.millicores && `${limit.cpu.millicores}m CPU`,
    !!limit.memory?.megabytes && `${limit.memory.megabytes} MB memory`,
    ...Object.entries(limit.ext).map(([key, value]) => `${value} ${key}`),
  ) || "No limits";

const securitySummary = (
  context: CoreP.Service_Spec_Config_Upstream_Container_SecurityContext,
) =>
  joinSummary(
    context.readOnlyRootFilesystem && "Read-only root",
    !!context.runAsUser && `UID ${context.runAsUser}`,
    !!context.capabilities?.add.length &&
      `+${listSummary(context.capabilities.add, 2)}`,
    !!context.capabilities?.drop.length &&
      `-${listSummary(context.capabilities.drop, 2)}`,
  ) || "Image defaults";

export const ContainerEditor = (props: {
  container: Container;
  update: Update;
}) => {
  const { container, update } = props;
  const credentials = container.credentials;
  const login =
    credentials?.type.oneofKind === "usernamePassword"
      ? credentials.type.usernamePassword
      : undefined;
  const limit = container.resourceLimit;
  const cpu = limit?.cpu ?? { millicores: 0 };
  const memory = limit?.memory ?? { megabytes: 0 };
  const updateLimit = () => {
    if (limit) {
      limit.cpu = cpu;
      limit.memory = memory;
    }
    update();
  };
  const security = container.securityContext;
  const capabilities =
    security?.capabilities ??
    CoreP.Service_Spec_Config_Upstream_Container_SecurityContext_Capabilities.create();
  const updateCapabilities = () => {
    if (security) {
      security.capabilities =
        capabilities.add.length + capabilities.drop.length > 0
          ? capabilities
          : undefined;
    }
    update();
  };
  const command = [...container.command, ...container.args].join(" ");

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <div className="grid grid-cols-2 gap-4 @2xl:grid-cols-[minmax(0,2.4fr)_minmax(0,1fr)_minmax(0,1fr)]">
        <TextField
          of={container}
          name="image"
          update={update}
          label="Image"
          placeholder="ghcr.io/acme/app:1.0.0"
          required
          className="col-span-2 @2xl:col-span-1"
        />
        <NumberField
          of={container}
          name="port"
          update={update}
          label="Port"
          placeholder="8080"
          required
          max={65535}
        />
        <NumberField
          of={container}
          name="replicas"
          update={update}
          label="Replicas"
          placeholder="1"
          max={100}
        />
      </div>

      <OptionList>
        <OptionRow
          icon={Variable}
          title="Environment variables"
          description="Literal values, Octelium Secrets or Kubernetes Secrets."
          summary={listSummary(container.env.map((item) => item.name))}
          isSet={container.env.length > 0}
          onAdd={() => {
            container.env.push(
              CoreP.Service_Spec_Config_Upstream_Container_Env.create({
                type: { oneofKind: "value", value: "" },
              }),
            );
            update();
          }}
          onRemove={() => {
            container.env = [];
            update();
          }}
        >
          <EnvList container={container} update={update} />
        </OptionRow>

        <OptionRow
          icon={SquareTerminal}
          title="Command and arguments"
          description="Override the entrypoint of the image."
          summary={<span className="font-mono">{command}</span>}
          isSet={container.command.length + container.args.length > 0}
          onAdd={() => {
            container.command = [""];
            update();
          }}
          onRemove={() => {
            container.command = [];
            container.args = [];
            update();
          }}
        >
          <StringList
            label="Command"
            description="One item per argv entry. Press Enter to add the next one."
            values={container.command}
            update={update}
            addLabel="Add item"
            itemLabel="Command item"
            placeholder="/bin/sh"
            mono
            numbered
          />
          <StringList
            label="Arguments"
            description="Passed to the command, in this order."
            values={container.args}
            update={update}
            addLabel="Add argument"
            itemLabel="Argument"
            placeholder="--port=8080"
            mono
            numbered
          />
        </OptionRow>

        <OptionRow
          icon={KeyRound}
          title="Registry credentials"
          description="Pull the image from a private registry."
          summary={login && joinSummary(login.username, login.server)}
          isSet={!!login}
          onAdd={() => {
            container.credentials =
              CoreP.Service_Spec_Config_Upstream_Container_Credentials.create({
                type: {
                  oneofKind: "usernamePassword",
                  usernamePassword:
                    CoreP.Service_Spec_Config_Upstream_Container_Credentials_UsernamePassword.create(
                      {
                        password: {
                          type: { oneofKind: "fromSecret", fromSecret: "" },
                        },
                      },
                    ),
                },
              });
            update();
          }}
          onRemove={() => {
            container.credentials = undefined;
            update();
          }}
        >
          {login && (
            <Grid cols={3}>
              <TextField
                of={login}
                name="server"
                update={update}
                label="Server"
                placeholder="ghcr.io"
              />
              <TextField
                of={login}
                name="username"
                update={update}
                label="Username"
                required
              />
              <SecretRefField
                of={login}
                name="password"
                update={update}
                label="Password"
                required
              />
            </Grid>
          )}
        </OptionRow>

        <OptionRow
          icon={Cpu}
          title="Resource limits"
          description="CPU, memory and extended resources such as GPUs."
          summary={limit && resourceSummary(limit)}
          isSet={!!limit}
          onAdd={() => {
            container.resourceLimit =
              CoreP.Service_Spec_Config_Upstream_Container_ResourceLimit.create(
                { cpu: { millicores: 0 }, memory: { megabytes: 0 } },
              );
            update();
          }}
          onRemove={() => {
            container.resourceLimit = undefined;
            update();
          }}
        >
          {limit && (
            <>
              <Grid>
                <NumberField
                  of={cpu}
                  name="millicores"
                  update={updateLimit}
                  label="CPU"
                  placeholder="2000"
                  suffix=" m"
                  thousands
                />
                <NumberField
                  of={memory}
                  name="megabytes"
                  update={updateLimit}
                  label="Memory"
                  placeholder="4096"
                  suffix=" MB"
                  thousands
                />
              </Grid>
              <ExtendedResources limit={limit} update={update} />
            </>
          )}
        </OptionRow>

        <OptionRow
          icon={HardDrive}
          title="Volumes"
          description="Empty directories and persistent volume claims."
          summary={joinSummary(
            plural(container.volumes.length, "volume"),
            plural(container.volumeMounts.length, "mount"),
          )}
          isSet={container.volumes.length > 0}
          onAdd={() => {
            container.volumes.push(
              CoreP.Service_Spec_Config_Upstream_Container_Volume.create({
                type: {
                  oneofKind: "emptyDir",
                  emptyDir:
                    CoreP.Service_Spec_Config_Upstream_Container_Volume_EmptyDir.create(),
                },
              }),
            );
            update();
          }}
          onRemove={() => {
            container.volumes = [];
            container.volumeMounts = [];
            update();
          }}
        >
          <Volumes container={container} update={update} />
        </OptionRow>

        <ProbeRow
          title="Liveness probe"
          description="Restart the container when this check fails."
          liveness
          container={container}
          field="livenessProbe"
          update={update}
        />
        <ProbeRow
          title="Readiness probe"
          description="Send traffic only after this check passes."
          container={container}
          field="readinessProbe"
          update={update}
        />

        <OptionRow
          icon={ShieldCheck}
          title="Security context"
          description="User, root filesystem and Linux capabilities."
          summary={security && securitySummary(security)}
          isSet={!!security}
          onAdd={() => {
            container.securityContext =
              CoreP.Service_Spec_Config_Upstream_Container_SecurityContext.create();
            update();
          }}
          onRemove={() => {
            container.securityContext = undefined;
            update();
          }}
        >
          {security && (
            <>
              <NumberField
                of={security}
                name="runAsUser"
                update={update}
                label="Run as UID"
                placeholder="Image default"
              />
              <ToggleList>
                <Toggle
                  of={security}
                  name="readOnlyRootFilesystem"
                  update={update}
                  label="Read-only root filesystem"
                  description="Block writes outside mounted volumes."
                />
              </ToggleList>
              <Grid>
                <TagsField
                  of={capabilities}
                  name="add"
                  update={updateCapabilities}
                  label="Add capabilities"
                  placeholder="NET_ADMIN"
                  data={LINUX_CAPABILITIES}
                  error={capabilityError}
                />
                <TagsField
                  of={capabilities}
                  name="drop"
                  update={updateCapabilities}
                  label="Drop capabilities"
                  placeholder="ALL"
                  data={LINUX_CAPABILITIES}
                  error={capabilityError}
                />
              </Grid>
            </>
          )}
        </OptionRow>
      </OptionList>
    </div>
  );
};
