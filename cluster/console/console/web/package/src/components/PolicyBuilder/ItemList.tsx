import * as CoreP from "@/apis/corev1/corev1";
import { Condition_Expression as Expression } from "@/apis/enterprisev1/enterprisev1";
import { getOSTypeStr } from "@/pages/core/Device/Main";
import {
  getResourceRef,
  printResourceNameWithDisplay,
  printServiceMode,
} from "@/utils/pb";
import { Select, Switch, TextInput, Group } from "@mantine/core";
import { match } from "ts-pattern";
import SelectResource from "../ResourceLayout/SelectResource";
import TimeAgo from "../TimeAgo";
import TimestampPicker from "../TimestampPicker";
import { ObjectReference } from "@/apis/metav1/metav1";
import { useEffect, useState } from "react";
import { useResourceFromRef } from "../ResourceLayout/utils";
import SelectCountry from "../SelectCountry";

const ResourceValue = (props: { itemRef: ObjectReference }) => {
  const { data } = useResourceFromRef(props.itemRef);
  return <>{data ? printResourceNameWithDisplay(data) : null}</>;
};

type EditProps = { item?: Expression; onUpdate: (item?: Expression) => void };
type ValueProps = { item: Expression };

export type ItemDef = {
  type: string;
  title: string;
  tags: string[];
  components: {
    Value: (props: ValueProps) => React.ReactNode;
    Edit: (props: EditProps) => React.ReactNode;
  };
};

const makeResourceItem = (
  type: string,
  title: string,
  tags: string[],
  api: "core" | "enterprise",
  kind: string,
  refKey: string,
): ItemDef => ({
  type,
  title,
  tags,
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      const ref = (item.type as any)[type][refKey] as ObjectReference | undefined;
      return ref ? <ResourceValue itemRef={ref} /> : null;
    },
    Edit: ({ onUpdate }) => (
      <SelectResource
        api={api}
        kind={kind}
        onChange={(v) => {
          if (!v) { onUpdate(); return; }
          onUpdate(Expression.create({
            type: { oneofKind: type, [type]: { [refKey]: getResourceRef(v) } } as any,
          }));
        }}
      />
    ),
  },
});

const makeEnumItem = <T extends string>(
  type: string,
  title: string,
  tags: string[],
  enumKey: string,
  data: { label: string; value: string }[],
  valueLabel: (val: number) => string,
): ItemDef => ({
  type,
  title,
  tags,
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{valueLabel((item.type as any)[type][enumKey])}</>;
    },
    Edit: ({ onUpdate }) => (
      <Select
        data={data}
        onChange={(v) => {
          if (!v) return;
          onUpdate(Expression.create({
            type: { oneofKind: type, [type]: { [enumKey]: (data.find(d => d.value === v) ? v : v) } } as any,
          }));
        }}
      />
    ),
  },
});

const makeBoolItem = (
  type: string,
  title: string,
  tags: string[],
  boolKey: string,
): ItemDef => ({
  type,
  title,
  tags,
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{(item.type as any)[type][boolKey] ? "Enabled" : "Disabled"}</>;
    },
    Edit: ({ item, onUpdate }) => (
      <Switch
        size="md"
        checked={
          item?.type.oneofKind === type
            ? (item.type as any)[type][boolKey]
            : false
        }
        onChange={(e) =>
          onUpdate(Expression.create({
            type: { oneofKind: type, [type]: { [boolKey]: e.currentTarget.checked } } as any,
          }))
        }
      />
    ),
  },
});

const makeTextItem = (
  type: string,
  title: string,
  tags: string[],
  valueKey: string,
  label: string,
  placeholder: string,
): ItemDef => ({
  type,
  title,
  tags,
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{(item.type as any)[type][valueKey]}</>;
    },
    Edit: ({ item, onUpdate }) => (
      <TextInput
        label={label}
        placeholder={placeholder}
        value={
          item?.type.oneofKind === type
            ? (item.type as any)[type][valueKey]
            : ""
        }
        onChange={(e) =>
          onUpdate(Expression.create({
            type: { oneofKind: type, [type]: { [valueKey]: e.target.value } } as any,
          }))
        }
      />
    ),
  },
});

export const itemList: ItemDef[] = [
  makeResourceItem("user", "User", ["identity", "user"], "core", "User", "userRef"),
  makeResourceItem("session", "Session", ["session", "access"], "core", "Session", "sessionRef"),
  makeResourceItem("device", "Device", ["device", "endpoint"], "core", "Device", "deviceRef"),
  makeResourceItem("service", "Service", ["service", "resource"], "core", "Service", "serviceRef"),
  makeResourceItem("group", "User belongs to group", ["identity", "group", "user"], "core", "Group", "groupRef"),
  makeResourceItem("namespace", "Namespace", ["resource", "namespace"], "core", "Namespace", "namespaceRef"),
  makeResourceItem(
    "sessionAuthenticationCredential",
    "Session credential",
    ["session", "authentication", "credential"],
    "core", "Credential", "credentialRef",
  ),
  makeResourceItem(
    "sessionAuthenticationIdentityProvider",
    "Session identity provider",
    ["session", "authentication", "identity provider", "sso"],
    "core", "IdentityProvider", "identityProviderRef",
  ),

  {
    type: "userType",
    title: "User type",
    tags: ["identity", "user", "type"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "userType") return null;
        return <>{match(item.type.userType.type)
          .with(CoreP.User_Spec_Type.HUMAN, () => "Human")
          .with(CoreP.User_Spec_Type.WORKLOAD, () => "Workload")
          .otherwise(() => "")}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={[
            { label: "Human", value: CoreP.User_Spec_Type[CoreP.User_Spec_Type.HUMAN] },
            { label: "Workload", value: CoreP.User_Spec_Type[CoreP.User_Spec_Type.WORKLOAD] },
          ]}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: { oneofKind: "userType", userType: { type: CoreP.User_Spec_Type[v as "HUMAN"] } },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "deviceOSType",
    title: "Device OS type",
    tags: ["device", "os", "platform", "endpoint"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "deviceOSType") return null;
        return <>{getOSTypeStr(item.type.deviceOSType.osType)}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={[
            CoreP.Device_Status_OSType.LINUX,
            CoreP.Device_Status_OSType.WINDOWS,
            CoreP.Device_Status_OSType.MAC,
            CoreP.Device_Status_OSType.ANDROID,
          ].map((v) => ({
            label: getOSTypeStr(v),
            value: CoreP.Device_Status_OSType[v],
          }))}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: { oneofKind: "deviceOSType", deviceOSType: { osType: CoreP.Device_Status_OSType[v as "LINUX"] } },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "serviceMode",
    title: "Service mode",
    tags: ["service", "protocol", "mode", "http", "tcp", "ssh"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "serviceMode") return null;
        return <>{printServiceMode(item.type.serviceMode.mode)}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={[
            CoreP.Service_Spec_Mode.HTTP,
            CoreP.Service_Spec_Mode.TCP,
            CoreP.Service_Spec_Mode.SSH,
            CoreP.Service_Spec_Mode.WEB,
            CoreP.Service_Spec_Mode.KUBERNETES,
            CoreP.Service_Spec_Mode.POSTGRES,
            CoreP.Service_Spec_Mode.MYSQL,
            CoreP.Service_Spec_Mode.UDP,
            CoreP.Service_Spec_Mode.GRPC,
            CoreP.Service_Spec_Mode.DNS,
          ].map((v) => ({
            label: printServiceMode(v),
            value: CoreP.Service_Spec_Mode[v],
          }))}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: { oneofKind: "serviceMode", serviceMode: { mode: CoreP.Service_Spec_Mode[v as "HTTP"] } },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "sessionType",
    title: "Session type",
    tags: ["session", "type", "client", "clientless"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionType") return null;
        return <>{match(item.type.sessionType.type)
          .with(CoreP.Session_Status_Type.CLIENT, () => "Client")
          .with(CoreP.Session_Status_Type.CLIENTLESS, () => "Clientless")
          .otherwise(() => "")}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={[
            { label: "Client", value: CoreP.Session_Status_Type[CoreP.Session_Status_Type.CLIENT] },
            { label: "Clientless", value: CoreP.Session_Status_Type[CoreP.Session_Status_Type.CLIENTLESS] },
          ]}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: { oneofKind: "sessionType", sessionType: { type: CoreP.Session_Status_Type[v as "CLIENT"] } },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationAAL",
    title: "Session authentication assurance level (AAL)",
    tags: ["session", "authentication", "aal", "assurance", "mfa"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationAAL") return null;
        return <>{match(item.type.sessionAuthenticationAAL.aal)
          .with(CoreP.Session_Status_Authentication_Info_AAL.AAL1, () => "AAL1")
          .with(CoreP.Session_Status_Authentication_Info_AAL.AAL2, () => "AAL2")
          .with(CoreP.Session_Status_Authentication_Info_AAL.AAL3, () => "AAL3")
          .otherwise(() => "")}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={["AAL1", "AAL2", "AAL3"].map((v) => ({ label: v, value: v }))}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: {
                oneofKind: "sessionAuthenticationAAL",
                sessionAuthenticationAAL: {
                  aal: CoreP.Session_Status_Authentication_Info_AAL[v as "AAL1"],
                },
              },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationType",
    title: "Session authentication type",
    tags: ["session", "authentication", "type", "credential", "oauth", "idp"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationType") return null;
        return <>{match(item.type.sessionAuthenticationType.type)
          .with(CoreP.Session_Status_Authentication_Info_Type.AUTHENTICATOR, () => "Authenticator")
          .with(CoreP.Session_Status_Authentication_Info_Type.CREDENTIAL, () => "Credential")
          .with(CoreP.Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER, () => "Identity Provider")
          .with(CoreP.Session_Status_Authentication_Info_Type.REFRESH_TOKEN, () => "Refresh Token")
          .with(CoreP.Session_Status_Authentication_Info_Type.INTERNAL, () => "Internal")
          .otherwise(() => "")}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={[
            { label: "Authenticator", value: CoreP.Session_Status_Authentication_Info_Type[CoreP.Session_Status_Authentication_Info_Type.AUTHENTICATOR] },
            { label: "Credential", value: CoreP.Session_Status_Authentication_Info_Type[CoreP.Session_Status_Authentication_Info_Type.CREDENTIAL] },
            { label: "Identity Provider", value: CoreP.Session_Status_Authentication_Info_Type[CoreP.Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER] },
            { label: "Internal", value: CoreP.Session_Status_Authentication_Info_Type[CoreP.Session_Status_Authentication_Info_Type.INTERNAL] },
            { label: "Refresh Token", value: CoreP.Session_Status_Authentication_Info_Type[CoreP.Session_Status_Authentication_Info_Type.REFRESH_TOKEN] },
          ]}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: {
                oneofKind: "sessionAuthenticationType",
                sessionAuthenticationType: {
                  type: CoreP.Session_Status_Authentication_Info_Type[v as "CREDENTIAL"],
                },
              },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationCredentialType",
    title: "Session credential type",
    tags: ["session", "authentication", "credential", "oauth2", "token"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationCredentialType") return null;
        return <>{match(item.type.sessionAuthenticationCredentialType.type)
          .with(CoreP.Credential_Spec_Type.ACCESS_TOKEN, () => "Access Token")
          .with(CoreP.Credential_Spec_Type.AUTH_TOKEN, () => "Authentication Token")
          .with(CoreP.Credential_Spec_Type.OAUTH2, () => "OAuth2 Client Credentials")
          .otherwise(() => "")}</>;
      },
      Edit: ({ onUpdate }) => (
        <Select
          data={[
            { label: "Access Token", value: CoreP.Credential_Spec_Type[CoreP.Credential_Spec_Type.ACCESS_TOKEN] },
            { label: "Authentication Token", value: CoreP.Credential_Spec_Type[CoreP.Credential_Spec_Type.AUTH_TOKEN] },
            { label: "OAuth2 Client Credentials", value: CoreP.Credential_Spec_Type[CoreP.Credential_Spec_Type.OAUTH2] },
          ]}
          onChange={(v) => {
            if (!v) return;
            onUpdate(Expression.create({
              type: {
                oneofKind: "sessionAuthenticationCredentialType",
                sessionAuthenticationCredentialType: {
                  type: CoreP.Credential_Spec_Type[v as "OAUTH2"],
                },
              },
            }));
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationGeoipCountryCode",
    title: "Session country",
    tags: ["session", "geo", "country", "location", "ip"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationGeoipCountryCode") return null;
        return <>{item.type.sessionAuthenticationGeoipCountryCode.code}</>;
      },
      Edit: ({ item, onUpdate }) => (
        <SelectCountry
          val={
            item?.type.oneofKind === "sessionAuthenticationGeoipCountryCode"
              ? item.type.sessionAuthenticationGeoipCountryCode.code
              : undefined
          }
          onUpdate={(v) =>
            onUpdate(Expression.create({
              type: {
                oneofKind: "sessionAuthenticationGeoipCountryCode",
                sessionAuthenticationGeoipCountryCode: { code: v ?? "" },
              },
            }))
          }
        />
      ),
    },
  },

  {
    type: "timeBefore",
    title: "Time before",
    tags: ["time", "schedule", "temporal"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "timeBefore") return null;
        return <TimeAgo rfc3339={item.type.timeBefore.timestamp} />;
      },
      Edit: ({ item, onUpdate }) => (
        <TimestampPicker
          value={item?.type.oneofKind === "timeBefore" ? item.type.timeBefore.timestamp : undefined}
          onChange={(v) =>
            onUpdate(Expression.create({
              type: { oneofKind: "timeBefore", timeBefore: { timestamp: v } },
            }))
          }
        />
      ),
    },
  },

  {
    type: "timeAfter",
    title: "Time after",
    tags: ["time", "schedule", "temporal"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "timeAfter") return null;
        return <TimeAgo rfc3339={item.type.timeAfter.timestamp} />;
      },
      Edit: ({ item, onUpdate }) => (
        <TimestampPicker
          isFuture
          value={item?.type.oneofKind === "timeAfter" ? item.type.timeAfter.timestamp : undefined}
          onChange={(v) =>
            onUpdate(Expression.create({
              type: { oneofKind: "timeAfter", timeAfter: { timestamp: v } },
            }))
          }
        />
      ),
    },
  },

  makeBoolItem("sessionAuthenticationCredAuthenticatorFIDOHardware", "Hardware-based FIDO", ["fido", "hardware", "authenticator", "webauthn"], "isHardware"),
  makeBoolItem("sessionAuthenticationCredAuthenticatorFIDOAttestationVerified", "FIDO verified attestation", ["fido", "attestation", "webauthn", "security"], "isAttestationVerified"),
  makeBoolItem("sessionAuthenticationCredAuthenticatorFIDOUserVerified", "FIDO user verified", ["fido", "user", "verification", "webauthn"], "isUserVerified"),
  makeBoolItem("sessionAuthenticationCredAuthenticatorFIDOUserPresent", "FIDO user present", ["fido", "user", "presence", "webauthn"], "isUserPresent"),
  makeBoolItem("apiServer", "Request to API server", ["api", "server", "request"], "isAPIServer"),
  makeBoolItem("apiServerCore", "Request to core API", ["api", "server", "core"], "isAPIServerCore"),
  makeBoolItem("apiServerUser", "Request to user API", ["api", "server", "user"], "isAPIServerUser"),
  makeBoolItem("apiServerEnterprise", "Request to enterprise API", ["api", "server", "enterprise"], "isAPIServerEnterprise"),
  makeBoolItem("apiServerCordium", "Request to Coredium API", ["api", "server", "coredium"], "isAPIServerCordium"),

  makeTextItem("requestHTTPPathExact", "Request HTTP exact path", ["http", "request", "path", "url"], "value", "Exact path", "/api/v1"),
  makeTextItem("requestHTTPPathPrefix", "Request HTTP path prefix", ["http", "request", "path", "prefix", "url"], "value", "Path prefix", "/api/v1"),
  makeTextItem("requestHTTPHasHeader", "Request HTTP header exists", ["http", "request", "header"], "value", "Header name", "User-Agent"),
  makeTextItem("requestIP", "Request IP address", ["network", "ip", "request"], "value", "IP address", "1.2.3.4"),
  makeTextItem("requestIPInRange", "Request IP address range", ["network", "ip", "cidr", "range"], "value", "CIDR range", "1.2.3.0/24"),

  {
    type: "requestHTTPHeaderValue",
    title: "Request HTTP header value",
    tags: ["http", "request", "header", "value"],
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "requestHTTPHeaderValue") return null;
        const { header, value } = item.type.requestHTTPHeaderValue;
        return <>{`${header} = ${value}`}</>;
      },
      Edit: ({ item, onUpdate }) => {
        const [header, setHeader] = useState(
          item?.type.oneofKind === "requestHTTPHeaderValue"
            ? item.type.requestHTTPHeaderValue.header
            : "",
        );
        const [val, setVal] = useState(
          item?.type.oneofKind === "requestHTTPHeaderValue"
            ? item.type.requestHTTPHeaderValue.value
            : "",
        );

        useEffect(() => {
          onUpdate(Expression.create({
            type: {
              oneofKind: "requestHTTPHeaderValue",
              requestHTTPHeaderValue: { header, value: val },
            },
          }));
        }, [header, val]);

        return (
          <Group grow>
            <TextInput label="Header" placeholder="User-Agent" value={header} onChange={(e) => setHeader(e.target.value)} />
            <TextInput label="Value" placeholder="Mozilla/5.0" value={val} onChange={(e) => setVal(e.target.value)} />
          </Group>
        );
      },
    },
  },
];

export default itemList;