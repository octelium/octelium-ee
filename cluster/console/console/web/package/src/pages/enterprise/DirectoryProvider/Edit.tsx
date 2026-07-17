import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import SelectSecret from "@/components/ResourceLayout/SelectSecret";
import { Group, Switch, Tabs, TextInput } from "@mantine/core";
import * as React from "react";
import { match } from "ts-pattern";

const Edit = (props: {
  item: EnterpriseP.DirectoryProvider;
  onUpdate: (item: EnterpriseP.DirectoryProvider) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(
    EnterpriseP.DirectoryProvider.clone(item),
  );
  const updateReq = () => {
    setReq(EnterpriseP.DirectoryProvider.clone(req));
    onUpdate(req);
  };

  return (
    <div>
      <Switch
        label="Is Disabled"
        description="Disable the DirectoryProvider. Disabled DirectoryProviders cannot synchronize Users and Groups"
        checked={req.spec!.isDisabled}
        onChange={(v) => {
          req.spec!.isDisabled = v.currentTarget.checked;
          updateReq();
        }}
      />

      <Tabs
        className="mt-4"
        value={req.spec!.type.oneofKind}
        onChange={(v) => {
          match(v)
            .with("scim", () => {
              req.spec!.type = {
                oneofKind: "scim",
                scim: EnterpriseP.DirectoryProvider_Spec_SCIM.create(),
              };
            })
            .with("googleWorkspace", () => {
              req.spec!.type = {
                oneofKind: "googleWorkspace",
                googleWorkspace:
                  EnterpriseP.DirectoryProvider_Spec_GoogleWorkspace.create({
                    serviceAccount: {
                      type: { oneofKind: "fromSecret", fromSecret: "" },
                    },
                  }),
              };
            })
            .with("keycloak", () => {
              req.spec!.type = {
                oneofKind: "keycloak",
                keycloak: EnterpriseP.DirectoryProvider_Spec_Keycloak.create({
                  clientSecret: {
                    type: { oneofKind: "fromSecret", fromSecret: "" },
                  },
                }),
              };
            })
            .otherwise(() => {});
          updateReq();
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="scim">SCIM</Tabs.Tab>
          <Tabs.Tab value="googleWorkspace">Google Workspace</Tabs.Tab>
          <Tabs.Tab value="keycloak">Keycloak</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="scim">
          <></>
        </Tabs.Panel>

        <Tabs.Panel value="googleWorkspace">
          {match(req.spec!.type)
            .when(
              (x) => x.oneofKind === "googleWorkspace",
              (gw) => (
                <div className="mt-3">
                  <Group grow>
                    <TextInput
                      label="Customer"
                      description="The Google Workspace customer ID"
                      placeholder="C01abc234"
                      value={gw.googleWorkspace.customer}
                      onChange={(v) => {
                        gw.googleWorkspace.customer = v.target.value;
                        updateReq();
                      }}
                    />

                    <TextInput
                      label="Impersonate Subject"
                      description="The subject to impersonate via domain-wide delegation"
                      placeholder="admin@example.com"
                      value={gw.googleWorkspace.impersonateSubject}
                      onChange={(v) => {
                        gw.googleWorkspace.impersonateSubject = v.target.value;
                        updateReq();
                      }}
                    />
                  </Group>

                  {match(gw.googleWorkspace.serviceAccount?.type)
                    .when(
                      (x) => x?.oneofKind === "fromSecret",
                      (x) => (
                        <SelectSecret
                          api="enterprise"
                          label="Service Account Secret"
                          description="Select the Secret holding the service account JSON credentials"
                          defaultValue={x.fromSecret}
                          onChange={(val) => {
                            x.fromSecret = val ?? "";
                            updateReq();
                          }}
                        />
                      ),
                    )
                    .otherwise(() => (
                      <></>
                    ))}

                  <EditItem
                    title="Polling"
                    description="Set the periodic synchronization polling interval"
                    onUnset={() => {
                      gw.googleWorkspace.polling = undefined;
                      updateReq();
                    }}
                    obj={gw.googleWorkspace.polling}
                    onSet={() => {
                      gw.googleWorkspace.polling =
                        EnterpriseP.DirectoryProvider_Spec_GoogleWorkspace_Polling.create();
                      updateReq();
                    }}
                  >
                    {gw.googleWorkspace.polling && (
                      <DurationPicker
                        value={gw.googleWorkspace.polling.interval}
                        title="Interval"
                        onChange={(val) => {
                          gw.googleWorkspace.polling!.interval = val;
                          updateReq();
                        }}
                      />
                    )}
                  </EditItem>
                </div>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="keycloak">
          {match(req.spec!.type)
            .when(
              (x) => x.oneofKind === "keycloak",
              (kc) => (
                <div className="mt-3">
                  <Group grow>
                    <TextInput
                      label="URL"
                      description="The Keycloak base URL"
                      placeholder="https://keycloak.example.com"
                      value={kc.keycloak.url}
                      onChange={(v) => {
                        kc.keycloak.url = v.target.value;
                        updateReq();
                      }}
                    />

                    <TextInput
                      label="Realm"
                      placeholder="master"
                      value={kc.keycloak.realm}
                      onChange={(v) => {
                        kc.keycloak.realm = v.target.value;
                        updateReq();
                      }}
                    />
                  </Group>

                  <Group grow>
                    <TextInput
                      label="Client ID"
                      placeholder="octelium-dirsync"
                      value={kc.keycloak.clientID}
                      onChange={(v) => {
                        kc.keycloak.clientID = v.target.value;
                        updateReq();
                      }}
                    />

                    {match(kc.keycloak.clientSecret?.type)
                      .when(
                        (x) => x?.oneofKind === "fromSecret",
                        (x) => (
                          <SelectSecret
                            api="enterprise"
                            label="Client Secret"
                            description="Select the Secret holding the Keycloak client secret"
                            defaultValue={x.fromSecret}
                            onChange={(val) => {
                              x.fromSecret = val ?? "";
                              updateReq();
                            }}
                          />
                        ),
                      )
                      .otherwise(() => (
                        <></>
                      ))}
                  </Group>

                  <Group grow>
                    <Switch
                      label="Insecure Skip Verify"
                      description="Skip TLS verification of the Keycloak server (insecure)"
                      checked={kc.keycloak.insecureSkipVerify}
                      onChange={(v) => {
                        kc.keycloak.insecureSkipVerify =
                          v.currentTarget.checked;
                        updateReq();
                      }}
                    />
                  </Group>

                  <EditItem
                    title="Polling"
                    description="Set the periodic synchronization polling interval"
                    onUnset={() => {
                      kc.keycloak.polling = undefined;
                      updateReq();
                    }}
                    obj={kc.keycloak.polling}
                    onSet={() => {
                      kc.keycloak.polling =
                        EnterpriseP.DirectoryProvider_Spec_Keycloak_Polling.create();
                      updateReq();
                    }}
                  >
                    {kc.keycloak.polling && (
                      <DurationPicker
                        value={kc.keycloak.polling.interval}
                        title="Interval"
                        onChange={(val) => {
                          kc.keycloak.polling!.interval = val;
                          updateReq();
                        }}
                      />
                    )}
                  </EditItem>
                </div>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>
      </Tabs>
    </div>
  );
};

export default Edit;
