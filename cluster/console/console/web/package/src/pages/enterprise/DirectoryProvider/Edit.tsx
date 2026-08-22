import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { Alert, SegmentedControl, Switch, TextInput } from "@mantine/core";
import { AlertTriangle, Cloud, KeyRound, Network } from "lucide-react";
import * as React from "react";

type ProviderType = EnterpriseP.DirectoryProvider_Spec["type"];

const createType = (kind: ProviderType["oneofKind"]): ProviderType => {
  if (kind === "googleWorkspace") {
    return {
      oneofKind: "googleWorkspace",
      googleWorkspace:
        EnterpriseP.DirectoryProvider_Spec_GoogleWorkspace.create({
          serviceAccount: {
            type: { oneofKind: "fromSecret", fromSecret: "" },
          },
        }),
    };
  }
  if (kind === "keycloak") {
    return {
      oneofKind: "keycloak",
      keycloak: EnterpriseP.DirectoryProvider_Spec_Keycloak.create({
        clientSecret: {
          type: { oneofKind: "fromSecret", fromSecret: "" },
        },
      }),
    };
  }
  return {
    oneofKind: "scim",
    scim: EnterpriseP.DirectoryProvider_Spec_SCIM.create(),
  };
};

const cloneForEdit = (item: EnterpriseP.DirectoryProvider) => {
  const next = EnterpriseP.DirectoryProvider.clone(item);
  if (!next.spec) {
    next.spec = EnterpriseP.DirectoryProvider_Spec.create();
  }
  if (!next.spec.type.oneofKind) {
    next.spec.type = createType("scim");
  }
  return next;
};

const isValidURL = (value: string) => {
  if (!value) return true;
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
};

const isValidEmail = (value: string) =>
  !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);

const Edit = (props: {
  item: EnterpriseP.DirectoryProvider;
  onUpdate: (item: EnterpriseP.DirectoryProvider) => void;
}) => {
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const configurations = React.useRef<Partial<Record<string, ProviderType>>>({
    [req.spec!.type.oneofKind!]: structuredClone(req.spec!.type),
  });
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;

  React.useEffect(() => {
    const next = cloneForEdit(props.item);
    setReq(next);
    configurations.current = {
      [next.spec?.type.oneofKind ?? "scim"]: structuredClone(
        next.spec?.type ?? createType("scim"),
      ),
    };
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.DirectoryProvider.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.DirectoryProvider.clone(next));
  };

  if (!req.spec) return null;
  const type = req.spec.type;

  const changeType = (value: string) => {
    if (!req.spec || !["scim", "googleWorkspace", "keycloak"].includes(value))
      return;
    const currentKind = req.spec.type.oneofKind;
    if (currentKind) {
      configurations.current[currentKind] = structuredClone(req.spec.type);
    }
    req.spec.type = structuredClone(
      configurations.current[value] ??
        createType(value as ProviderType["oneofKind"]),
    );
    updateReq();
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="text-[0.72rem] font-bold text-slate-700">Provider status</div>
          <div className="mt-0.5 text-[0.67rem] font-semibold text-slate-400">
            Disabled providers cannot synchronize users or groups.
          </div>
        </div>
        <Switch
          label="Disabled"
          description="Disable synchronization for this provider."
          checked={req.spec.isDisabled}
          onChange={(event) => {
            req.spec!.isDisabled = event.currentTarget.checked;
            updateReq();
          }}
        />
      </div>

      <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-3">
        <div className="mb-2 text-[0.72rem] font-bold text-slate-700">
          Directory source
        </div>
        <SegmentedControl
          fullWidth
          value={type.oneofKind ?? "scim"}
          onChange={changeType}
          data={[
            { label: "SCIM", value: "scim" },
            { label: "Google Workspace", value: "googleWorkspace" },
            { label: "Keycloak", value: "keycloak" },
          ]}
        />
      </div>

      {type.oneofKind === "scim" && (
        <Alert color="blue" icon={<Network size={15} />} title="SCIM push provisioning">
          Configure your identity platform with the SCIM endpoint and bearer
          token shown on this provider’s main page.
        </Alert>
      )}

      {type.oneofKind === "googleWorkspace" && (
        <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 text-[0.75rem] font-bold text-slate-800">
            <Cloud size={15} /> Google Workspace
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput
              required
              label="Customer ID"
              description="Google Workspace customer identifier."
              placeholder="C01abc234"
              value={type.googleWorkspace.customer}
              onChange={(event) => {
                type.googleWorkspace.customer = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              required
              type="email"
              label="Impersonated administrator"
              description="Administrator used for domain-wide delegation."
              placeholder="admin@example.com"
              value={type.googleWorkspace.impersonateSubject}
              error={
                isValidEmail(type.googleWorkspace.impersonateSubject)
                  ? undefined
                  : "Enter a valid email address"
              }
              onChange={(event) => {
                type.googleWorkspace.impersonateSubject = event.target.value;
                updateReq();
              }}
            />
            <div className="md:col-span-2">
              <SelectResource
                api="enterprise"
                kind="Secret"
                required
                label="Service account Secret"
                description="Secret containing the Google service-account JSON credentials."
                defaultValue={
                  type.googleWorkspace.serviceAccount?.type.oneofKind ===
                  "fromSecret"
                    ? type.googleWorkspace.serviceAccount.type.fromSecret
                    : undefined
                }
                onChange={(secret) => {
                  type.googleWorkspace.serviceAccount = {
                    type: {
                      oneofKind: "fromSecret",
                      fromSecret: secret?.metadata?.name ?? "",
                    },
                  };
                  updateReq();
                }}
              />
            </div>
          </div>
          <EditItem
            title="Polling"
            description="Periodically synchronize the Google directory."
            obj={type.googleWorkspace.polling}
            onUnset={() => {
              type.googleWorkspace.polling = undefined;
              updateReq();
            }}
            onSet={() => {
              type.googleWorkspace.polling =
                EnterpriseP.DirectoryProvider_Spec_GoogleWorkspace_Polling.create();
              updateReq();
            }}
          >
            {type.googleWorkspace.polling && (
              <DurationPicker
                value={type.googleWorkspace.polling.interval}
                title="Polling interval"
                description="How often the Google Workspace directory is synchronized."
                onChange={(value) => {
                  type.googleWorkspace.polling!.interval = value;
                  updateReq();
                }}
              />
            )}
          </EditItem>
        </section>
      )}

      {type.oneofKind === "keycloak" && (
        <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 text-[0.75rem] font-bold text-slate-800">
            <KeyRound size={15} /> Keycloak
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput
              required
              type="url"
              label="Server URL"
              description="Base URL of the Keycloak deployment."
              placeholder="https://keycloak.example.com"
              value={type.keycloak.url}
              error={isValidURL(type.keycloak.url) ? undefined : "Enter a valid HTTP or HTTPS URL"}
              onChange={(event) => {
                type.keycloak.url = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              required
              label="Realm"
              description="Keycloak realm whose users and groups are synchronized."
              placeholder="master"
              value={type.keycloak.realm}
              onChange={(event) => {
                type.keycloak.realm = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              required
              label="Client ID"
              description="Keycloak client used to access the Admin API."
              placeholder="octelium-dirsync"
              value={type.keycloak.clientID}
              onChange={(event) => {
                type.keycloak.clientID = event.target.value;
                updateReq();
              }}
            />
            <SelectResource
              api="enterprise"
              kind="Secret"
              required
              label="Client Secret"
              description="Secret containing the Keycloak OAuth client secret."
              defaultValue={
                type.keycloak.clientSecret?.type.oneofKind === "fromSecret"
                  ? type.keycloak.clientSecret.type.fromSecret
                  : undefined
              }
              onChange={(secret) => {
                type.keycloak.clientSecret = {
                  type: {
                    oneofKind: "fromSecret",
                    fromSecret: secret?.metadata?.name ?? "",
                  },
                };
                updateReq();
              }}
            />
          </div>

          {type.keycloak.insecureSkipVerify && (
            <Alert color="red" icon={<AlertTriangle size={15} />} title="TLS verification is disabled">
              The Keycloak server certificate will not be verified. Use this
              only for temporary diagnostics in a trusted environment.
            </Alert>
          )}
          <Switch
            label="Skip TLS verification"
            description="Skip Keycloak certificate-chain and hostname verification. Avoid this in production."
            checked={type.keycloak.insecureSkipVerify}
            onChange={(event) => {
              type.keycloak.insecureSkipVerify = event.currentTarget.checked;
              updateReq();
            }}
          />

          <EditItem
            title="Polling"
            description="Periodically synchronize the Keycloak directory."
            obj={type.keycloak.polling}
            onUnset={() => {
              type.keycloak.polling = undefined;
              updateReq();
            }}
            onSet={() => {
              type.keycloak.polling =
                EnterpriseP.DirectoryProvider_Spec_Keycloak_Polling.create();
              updateReq();
            }}
          >
            {type.keycloak.polling && (
              <DurationPicker
                value={type.keycloak.polling.interval}
                title="Polling interval"
                description="How often the Keycloak directory is synchronized."
                onChange={(value) => {
                  type.keycloak.polling!.interval = value;
                  updateReq();
                }}
              />
            )}
          </EditItem>
        </section>
      )}
    </div>
  );
};

export default Edit;
