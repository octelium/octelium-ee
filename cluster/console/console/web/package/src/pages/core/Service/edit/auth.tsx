import * as CoreP from "@/apis/corev1/corev1";
import {
  ChoiceGroup,
  Grid,
  isHTTPURL,
  secretName,
  toSecret,
} from "@/pages/enterprise/CollectorExporter/fields";
import * as React from "react";
import {
  joinSummary,
  Note,
  SecretRefField,
  SecretSourceField,
  TagsField,
  TextField,
  Update,
} from "./fields";
import { headerNameError } from "./model";

type Auth = CoreP.Service_Spec_Config_HTTP_Auth;

type Method = NonNullable<Auth["type"]["oneofKind"]> | "none";

const METHODS: { value: Method; label: string; hint: string }[] = [
  { value: "none", label: "None", hint: "Send no upstream credentials" },
  { value: "bearer", label: "Bearer token", hint: "Authorization header" },
  { value: "custom", label: "Custom header", hint: "An API key header" },
  { value: "basic", label: "Basic", hint: "Username and password" },
  { value: "oauth2ClientCredentials", label: "OAuth2", hint: "Client credentials" },
  { value: "sigv4", label: "AWS SigV4", hint: "Signed AWS requests" },
];

const createMethod = (method: Method): Auth["type"] => {
  switch (method) {
    case "basic":
      return {
        oneofKind: "basic",
        basic: CoreP.Service_Spec_Config_HTTP_Auth_Basic.create({
          password: toSecret(""),
        }),
      };
    case "custom":
      return {
        oneofKind: "custom",
        custom: CoreP.Service_Spec_Config_HTTP_Auth_Custom.create({
          value: toSecret(""),
        }),
      };
    case "oauth2ClientCredentials":
      return {
        oneofKind: "oauth2ClientCredentials",
        oauth2ClientCredentials:
          CoreP.Service_Spec_Config_HTTP_Auth_OAuth2ClientCredentials.create({
            clientSecret: toSecret(""),
          }),
      };
    case "sigv4":
      return {
        oneofKind: "sigv4",
        sigv4: CoreP.Service_Spec_Config_HTTP_Auth_Sigv4.create({
          secretAccessKey: toSecret(""),
        }),
      };
    default:
      return {
        oneofKind: "bearer",
        bearer: CoreP.Service_Spec_Config_HTTP_Auth_Bearer.create(toSecret("")),
      };
  }
};

export const authSummary = (auth?: Auth) => {
  const type = auth?.type;
  switch (type?.oneofKind) {
    case "bearer":
      return joinSummary("Bearer token", secretName(type.bearer));
    case "basic":
      return joinSummary("Basic", type.basic.username);
    case "custom":
      return joinSummary("Custom header", type.custom.header);
    case "oauth2ClientCredentials":
      return joinSummary("OAuth2", type.oauth2ClientCredentials.clientID);
    case "sigv4":
      return joinSummary("AWS SigV4", type.sigv4.service, type.sigv4.region);
    default:
      return undefined;
  }
};

export const AuthEditor = (props: {
  holder: { auth?: Auth };
  update: Update;
  note?: React.ReactNode;
  noSigV4?: boolean;
}) => {
  const { holder, update } = props;
  const type = holder.auth?.type;
  const memory = React.useRef<Partial<Record<Method, Auth["type"]>>>({});
  const value: Method = type?.oneofKind ?? "none";

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <ChoiceGroup
        label="Method"
        hideLabel
        cols={3}
        value={value}
        options={METHODS.filter(
          (item) => !props.noSigV4 || item.value !== "sigv4",
        )}
        onChange={(next) => {
          if (type) memory.current[value] = structuredClone(type);
          if (next === "none") {
            holder.auth = undefined;
          } else {
            holder.auth = CoreP.Service_Spec_Config_HTTP_Auth.create({
              type: memory.current[next] ?? createMethod(next),
            });
          }
          update();
        }}
      />
      {props.note && <Note>{props.note}</Note>}
      {type?.oneofKind === "bearer" && (
        <SecretSourceField
          label="Token"
          description="Secret sent as the bearer token."
          value={type.bearer}
          update={update}
        />
      )}
      {type?.oneofKind === "basic" && (
        <Grid>
          <TextField
            of={type.basic}
            name="username"
            update={update}
            label="Username"
            required
          />
          <SecretRefField
            of={type.basic}
            name="password"
            update={update}
            label="Password"
            required
          />
        </Grid>
      )}
      {type?.oneofKind === "custom" && (
        <Grid>
          <TextField
            of={type.custom}
            name="header"
            update={update}
            label="Header"
            placeholder="X-API-Key"
            required
            error={headerNameError}
          />
          <SecretRefField
            of={type.custom}
            name="value"
            update={update}
            label="Value"
            required
          />
        </Grid>
      )}
      {type?.oneofKind === "oauth2ClientCredentials" && (
        <>
          <Grid>
            <TextField
              of={type.oauth2ClientCredentials}
              name="clientID"
              update={update}
              label="Client ID"
              required
            />
            <SecretRefField
              of={type.oauth2ClientCredentials}
              name="clientSecret"
              update={update}
              label="Client secret"
              required
            />
          </Grid>
          <TextField
            of={type.oauth2ClientCredentials}
            name="tokenURL"
            update={update}
            label="Token URL"
            placeholder="https://auth.example.com/oauth/token"
            required
            error={(url) => (isHTTPURL(url) ? undefined : "Enter a full URL")}
          />
          <TagsField
            of={type.oauth2ClientCredentials}
            name="scopes"
            update={update}
            label="Scopes"
            placeholder="Add a scope"
          />
        </>
      )}
      {type?.oneofKind === "sigv4" && (
        <Grid>
          <TextField
            of={type.sigv4}
            name="accessKeyID"
            update={update}
            label="Access key ID"
            required
          />
          <SecretRefField
            of={type.sigv4}
            name="secretAccessKey"
            update={update}
            label="Secret access key"
            required
          />
          <TextField
            of={type.sigv4}
            name="region"
            update={update}
            label="Region"
            placeholder="eu-central-1"
            required
          />
          <TextField
            of={type.sigv4}
            name="service"
            update={update}
            label="AWS service"
            placeholder="bedrock"
            required
          />
        </Grid>
      )}
    </div>
  );
};
