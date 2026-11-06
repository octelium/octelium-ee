import * as CoreP from "@/apis/corev1/corev1";
import {
  Callout,
  ChoiceGroup,
  FormSection,
  Grid,
  secretName,
  toSecret,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { Select, Textarea } from "@mantine/core";
import {
  Database,
  Eye,
  KeyRound,
  ShieldCheck,
  SquareTerminal,
} from "lucide-react";
import * as React from "react";
import {
  Note,
  PEMList,
  SecretField,
  SecretRefField,
  SecretSourceField,
  Segments,
  StringList,
  TextField,
  Toggle,
  Update,
} from "./fields";

const SSLMode = CoreP.Service_Spec_Config_Postgres_SSLMode;
const PostgresAuthorization = CoreP.Service_Spec_Config_Postgres_Authorization_Mode;
const MySQLAuthorization = CoreP.Service_Spec_Config_MySQL_Authorization_Mode;

type SSHAuthKind = "none" | "password" | "privateKey";

export const SSHForm = (props: {
  value: CoreP.Service_Spec_Config_SSH;
  update: Update;
}) => {
  const { value, update } = props;
  const auth = value.auth?.type;
  const hostKey = value.upstreamHostKey?.type;
  const visibility =
    value.visibility ?? CoreP.Service_Spec_Config_SSH_Visibility.create();
  const updateVisibility = () => {
    value.visibility =
      visibility.disableSessionRecording ||
      visibility.enableSessionStdinRecording
        ? visibility
        : undefined;
    update();
  };
  const hostKeyKind =
    hostKey?.oneofKind === "key"
      ? "key"
      : hostKey?.oneofKind === "insecureIgnoreHostKey" &&
          hostKey.insecureIgnoreHostKey
        ? "ignore"
        : "ca";

  return (
    <>
      <FormSection
        icon={SquareTerminal}
        title="Session"
        description="What Users can do once they are connected."
      >
        <TextField
          of={value}
          name="user"
          update={update}
          label="Upstream user"
          description="Overrides the user that clients ask for."
          placeholder="The user requested by the client"
        />
        <ToggleList>
          <Toggle
            of={value}
            name="enableLocalPortForwarding"
            update={update}
            label="Local port forwarding"
            description="Allow direct-tcpip channels."
          />
          <Toggle
            of={value}
            name="enableSubsystem"
            update={update}
            label="Subsystems"
            description="Allow subsystem requests such as SFTP."
          />
        </ToggleList>
      </FormSection>

      {!value.eSSHMode && (
        <FormSection
          icon={KeyRound}
          title="Upstream authentication"
          description="How the Service logs in and verifies the SSH server."
        >
          <ChoiceGroup<SSHAuthKind>
            label="Log in with"
            value={auth?.oneofKind ?? "none"}
            options={[
              {
                value: "none",
                label: "Cluster certificate",
                hint: "Signed by the Cluster SSH CA",
              },
              { value: "password", label: "Password", hint: "From a Secret" },
              {
                value: "privateKey",
                label: "Private key",
                hint: "From a Secret",
              },
            ]}
            onChange={(next) => {
              value.auth =
                next === "none"
                  ? undefined
                  : CoreP.Service_Spec_Config_SSH_Auth.create({
                      type:
                        next === "password"
                          ? { oneofKind: "password", password: toSecret("") }
                          : { oneofKind: "privateKey", privateKey: toSecret("") },
                    });
              update();
            }}
          />
          {auth?.oneofKind === "password" && (
            <SecretSourceField
              label="Password"
              value={auth.password}
              update={update}
            />
          )}
          {auth?.oneofKind === "privateKey" && (
            <SecretSourceField
              label="Private key"
              description="Secret with the PEM-encoded key."
              value={auth.privateKey}
              update={update}
            />
          )}
          <ChoiceGroup
            label="Verify the server with"
            value={hostKeyKind}
            options={[
              { value: "ca", label: "Cluster CA", hint: "Signed host keys" },
              { value: "key", label: "Pinned key", hint: "One known host key" },
              { value: "ignore", label: "Nothing", hint: "Not for production" },
            ]}
            onChange={(next) => {
              value.upstreamHostKey =
                next === "ca"
                  ? undefined
                  : CoreP.Service_Spec_Config_SSH_UpstreamHostKey.create({
                      type:
                        next === "key"
                          ? { oneofKind: "key", key: "" }
                          : {
                              oneofKind: "insecureIgnoreHostKey",
                              insecureIgnoreHostKey: true,
                            },
                    });
              update();
            }}
          />
          {hostKey?.oneofKind === "key" && (
            <Textarea
              label="Host public key"
              classNames={{ input: "font-mono !text-xs" }}
              placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
              autosize
              minRows={2}
              required
              value={hostKey.key}
              onChange={(event) => {
                hostKey.key = event.currentTarget.value;
                update();
              }}
            />
          )}
          {hostKeyKind === "ignore" && (
            <Callout title="The identity of the server is not checked">
              A machine in the middle could impersonate this server.
            </Callout>
          )}
        </FormSection>
      )}

      <FormSection
        icon={Eye}
        title="Visibility"
        description="What the AccessLogs record about each session."
      >
        <ToggleList>
          <Toggle
            of={visibility}
            name="disableSessionRecording"
            update={updateVisibility}
            label="Do not record sessions"
          />
          {!visibility.disableSessionRecording && (
            <Toggle
              of={visibility}
              name="enableSessionStdinRecording"
              update={updateVisibility}
              label="Also record stdin"
              description="Keystrokes can include passwords that users type."
            />
          )}
        </ToggleList>
      </FormSection>
    </>
  );
};

const SSL_MODES = [
  {
    value: SSLMode.SSL_MODE_UNSET,
    label: "Prefer TLS",
    hint: "Uses TLS when the database offers it, without verifying it.",
  },
  {
    value: SSLMode.DISABLE,
    label: "Disable",
    hint: "Connects in plaintext.",
  },
  {
    value: SSLMode.REQUIRE,
    label: "Require",
    hint: "Encrypts the connection without verifying the certificate.",
  },
  {
    value: SSLMode.VERIFY_CA,
    label: "Verify CA",
    hint: "Also checks that a trusted CA signed the certificate.",
  },
  {
    value: SSLMode.VERIFY_FULL,
    label: "Verify full",
    hint: "Also checks the hostname. Recommended for TLS databases.",
  },
];

type DatabaseConfig = {
  user: string;
  database: string;
  auth?: {
    type:
      | { oneofKind: "password"; password: { type: any } }
      | { oneofKind: undefined };
  };
};

const DatabaseFields = (props: {
  value: DatabaseConfig;
  create: (password: ReturnType<typeof toSecret>) => DatabaseConfig["auth"];
  update: Update;
}) => {
  const { value, update } = props;
  const password =
    value.auth?.type.oneofKind === "password"
      ? secretName(value.auth.type.password)
      : "";
  return (
    <Grid cols={3}>
      <TextField
        of={value}
        name="user"
        update={update}
        label="User"
        placeholder="Requested by the client"
      />
      <TextField
        of={value}
        name="database"
        update={update}
        label="Database"
        placeholder="Requested by the client"
      />
      <SecretField
        label="Password"
        value={password}
        onChange={(name) => {
          value.auth = name ? props.create(toSecret(name)) : undefined;
          update();
        }}
      />
    </Grid>
  );
};

const AUTHORIZATION_OPTIONS = [
  { value: "connection", label: "Once per connection" },
  { value: "command", label: "Every command" },
];

export const PostgresForm = (props: {
  value: CoreP.Service_Spec_Config_Postgres;
  update: Update;
}) => {
  const { value, update } = props;
  const sslMode = SSL_MODES.find((item) => item.value === value.sslMode);
  const disableQuery = !!value.visibility?.disableQuery;
  return (
    <>
      <FormSection
        icon={Database}
        title="Connection"
        description="How the Service connects to the database, so that Users never see the password."
      >
        <DatabaseFields
          value={value}
          create={(password) =>
            CoreP.Service_Spec_Config_Postgres_Auth.create({
              type: { oneofKind: "password", password },
            })
          }
          update={update}
        />
        <Select
          label="TLS"
          description={sslMode?.hint}
          allowDeselect={false}
          data={SSL_MODES.map((item) => ({
            value: String(item.value),
            label: item.label,
          }))}
          value={String(value.sslMode)}
          onChange={(next) => {
            value.sslMode = Number(next);
            update();
          }}
        />
      </FormSection>
      <FormSection
        icon={ShieldCheck}
        title="Authorization and logging"
        description="When Policies run and what the AccessLogs record."
      >
        <Segments
          label="Evaluate Policies"
          value={
            value.authorization?.mode === PostgresAuthorization.ALL
              ? "command"
              : "connection"
          }
          options={AUTHORIZATION_OPTIONS}
          onChange={(next) => {
            value.authorization =
              CoreP.Service_Spec_Config_Postgres_Authorization.create({
                mode:
                  next === "command"
                    ? PostgresAuthorization.ALL
                    : PostgresAuthorization.NONE,
              });
            update();
          }}
        />
        <ToggleList>
          <ToggleQuery
            checked={disableQuery}
            onChange={(checked) => {
              value.visibility = checked
                ? CoreP.Service_Spec_Config_Postgres_Visibility.create({
                    disableQuery: true,
                  })
                : undefined;
              update();
            }}
          />
        </ToggleList>
      </FormSection>
    </>
  );
};

const ToggleQuery = (props: {
  checked: boolean;
  onChange: (checked: boolean) => void;
}) => {
  const holder = { disableQuery: props.checked };
  return (
    <Toggle
      of={holder}
      name="disableQuery"
      update={() => props.onChange(holder.disableQuery)}
      label="Do not record statements"
      description="Recommended when statements embed credentials or other sensitive values."
    />
  );
};

export const MySQLForm = (props: {
  value: CoreP.Service_Spec_Config_MySQL;
  update: Update;
}) => {
  const { value, update } = props;
  const disableQuery = !!value.visibility?.disableQuery;
  return (
    <>
      <FormSection
        icon={Database}
        title="Connection"
        description="How the Service connects to the database, so that Users never see the password."
      >
        <DatabaseFields
          value={value}
          create={(password) =>
            CoreP.Service_Spec_Config_MySQL_Auth.create({
              type: { oneofKind: "password", password },
            })
          }
          update={update}
        />
        <ToggleList>
          <Toggle
            of={value}
            name="isTLS"
            update={update}
            label="Connect over TLS"
            description="Required when the database listens over TLS."
          />
        </ToggleList>
      </FormSection>
      <FormSection
        icon={ShieldCheck}
        title="Authorization and logging"
        description="When Policies run and what the AccessLogs record."
      >
        <Segments
          label="Evaluate Policies"
          value={
            value.authorization?.mode === MySQLAuthorization.ALL
              ? "command"
              : "connection"
          }
          options={AUTHORIZATION_OPTIONS}
          onChange={(next) => {
            value.authorization =
              CoreP.Service_Spec_Config_MySQL_Authorization.create({
                mode:
                  next === "command"
                    ? MySQLAuthorization.ALL
                    : MySQLAuthorization.NONE,
              });
            update();
          }}
        />
        <ToggleList>
          <ToggleQuery
            checked={disableQuery}
            onChange={(checked) => {
              value.visibility = checked
                ? CoreP.Service_Spec_Config_MySQL_Visibility.create({
                    disableQuery: true,
                  })
                : undefined;
              update();
            }}
          />
        </ToggleList>
      </FormSection>
    </>
  );
};

type KubernetesType = CoreP.Service_Spec_Config_Kubernetes["type"];

const createKubernetesType = (
  kind: NonNullable<KubernetesType["oneofKind"]>,
): KubernetesType => {
  switch (kind) {
    case "bearerToken":
      return {
        oneofKind: "bearerToken",
        bearerToken: CoreP.Service_Spec_Config_Kubernetes_BearerToken.create({
          type: { oneofKind: "fromSecret", fromSecret: "" },
        }),
      };
    case "clientCertificate":
      return {
        oneofKind: "clientCertificate",
        clientCertificate: CoreP.Service_Spec_Config_ClientCertificate.create({
          type: { oneofKind: "fromSecret", fromSecret: "" },
        }),
      };
    default:
      return {
        oneofKind: "kubeconfig",
        kubeconfig: CoreP.Service_Spec_Config_Kubernetes_Kubeconfig.create({
          type: { oneofKind: "fromSecret", fromSecret: "" },
        }),
      };
  }
};

export const KubernetesForm = (props: {
  value: CoreP.Service_Spec_Config_Kubernetes;
  update: Update;
}) => {
  const { value, update } = props;
  const type = value.type;
  const memory = React.useRef<Record<string, KubernetesType>>({});
  return (
    <FormSection
      icon={KeyRound}
      title="Cluster credentials"
      description="How the Service authenticates to the Kubernetes API server, so that Users never see them."
    >
      <ChoiceGroup
        label="Authenticate with"
        hideLabel
        value={type.oneofKind}
        options={[
          {
            value: "kubeconfig",
            label: "Kubeconfig",
            hint: "A kubeconfig in a Secret",
          },
          {
            value: "bearerToken",
            label: "Bearer token",
            hint: "A ServiceAccount token",
          },
          {
            value: "clientCertificate",
            label: "Client certificate",
            hint: "x509 mTLS",
          },
        ]}
        onChange={(next) => {
          if (type.oneofKind) memory.current[type.oneofKind] = type;
          value.type = memory.current[next] ?? createKubernetesType(next);
          update();
        }}
      />
      {type.oneofKind === "kubeconfig" && (
        <Grid>
          <SecretSourceField
            label="Kubeconfig"
            value={type.kubeconfig}
            update={update}
          />
          <TextField
            of={type.kubeconfig}
            name="context"
            update={update}
            label="Context"
            placeholder="The current context"
          />
        </Grid>
      )}
      {type.oneofKind === "bearerToken" && (
        <>
          <SecretSourceField
            label="Token"
            value={type.bearerToken}
            update={update}
          />
          <PEMList
            label="Trusted CAs"
            description="PEM-encoded CAs that signed the API server certificate."
            values={type.bearerToken.trustedCAs}
            update={update}
          />
        </>
      )}
      {type.oneofKind === "clientCertificate" && (
        <>
          <SecretSourceField
            label="Client certificate"
            description="Secret with the PEM certificate chain and private key."
            value={type.clientCertificate}
            update={update}
          />
          <PEMList
            label="Trusted CAs"
            description="PEM-encoded CAs that signed the API server certificate."
            values={type.clientCertificate.trustedCAs}
            update={update}
          />
        </>
      )}
    </FormSection>
  );
};

export const SOCKS5Form = (props: {
  value: CoreP.Service_Spec_Config_SOCKS5;
  update: Update;
}) => {
  const { value, update } = props;
  const type = value.auth?.type;
  const login =
    type?.oneofKind === "usernamePassword" ? type.usernamePassword : undefined;
  return (
    <FormSection
      icon={KeyRound}
      title="Upstream authentication"
      description="How the Service authenticates to the upstream SOCKS5 proxy."
    >
      {value.isEmbeddedMode ? (
        <Note>
          Connected clients act as the egress, so there is no upstream proxy to
          authenticate to. Clients pick the egress Session with their SOCKS5
          username.
        </Note>
      ) : (
        <>
          <ChoiceGroup
            label="Method"
            hideLabel
            value={login ? "usernamePassword" : "none"}
            options={[
              { value: "none", label: "None", hint: "An open proxy" },
              {
                value: "usernamePassword",
                label: "Username and password",
                hint: "The password comes from a Secret",
              },
            ]}
            onChange={(next) => {
              value.auth =
                next === "none"
                  ? undefined
                  : CoreP.Service_Spec_Config_SOCKS5_Auth.create({
                      type: {
                        oneofKind: "usernamePassword",
                        usernamePassword:
                          CoreP.Service_Spec_Config_SOCKS5_Auth_UsernamePassword.create(
                            { password: toSecret("") },
                          ),
                      },
                    });
              update();
            }}
          />
          {login && (
            <Grid>
              <TextField
                of={login}
                name="username"
                update={update}
                label="Username"
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
        </>
      )}
    </FormSection>
  );
};

export const RDPForm = (props: {
  value: CoreP.Service_Spec_Config_RDP;
  update: Update;
}) => {
  const { value, update } = props;
  const auth = value.auth ?? CoreP.Service_Spec_Config_RDP_Auth.create();
  const updateAuth = () => {
    value.auth =
      auth.user || auth.domain || secretName(auth.password) ? auth : undefined;
    update();
  };
  const pins = value.upstreamTLS?.pinnedCertSHA256;
  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Upstream authentication"
        description="Credentials for Network Level Authentication. Without them, Users log in themselves."
      >
        <Grid cols={3}>
          <TextField
            of={auth}
            name="user"
            update={updateAuth}
            label="User"
            placeholder="administrator"
          />
          <TextField
            of={auth}
            name="domain"
            update={updateAuth}
            label="Domain"
            placeholder="Local account"
          />
          <SecretRefField
            of={auth}
            name="password"
            update={updateAuth}
            label="Password"
          />
        </Grid>
      </FormSection>
      <FormSection
        icon={ShieldCheck}
        title="Server certificate"
        description="Windows RDP servers usually present self-signed certificates."
      >
        <ChoiceGroup
          label="Verification"
          hideLabel
          value={pins?.length ? "pin" : "any"}
          options={[
            { value: "any", label: "Accept any", hint: "No verification" },
            {
              value: "pin",
              label: "Pin fingerprints",
              hint: "Only known certificates",
            },
          ]}
          onChange={(next) => {
            value.upstreamTLS =
              next === "any"
                ? undefined
                : CoreP.Service_Spec_Config_RDP_UpstreamTLS.create({
                    pinnedCertSHA256: [""],
                  });
            update();
          }}
        />
        {!!pins?.length && (
          <StringList
            label="SHA-256 fingerprints"
            description="Add a second one while rotating the certificate."
            values={pins}
            update={update}
            addLabel="Add fingerprint"
            itemLabel="Fingerprint"
            placeholder="ab:cd:ef:12:34:..."
            mono
            error={(item) =>
              /^[0-9a-fA-F]{64}$/.test(
                item.replace(/^sha256\//, "").replace(/:/g, "").trim(),
              )
                ? undefined
                : "Enter a 64-character SHA-256 hex fingerprint"
            }
          />
        )}
      </FormSection>
    </>
  );
};

