import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";
import { cloneResource } from "@/utils/pb";

import DurationPicker from "@/components/DurationPicker";
import ItemMessage from "@/components/ItemMessage";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import { strToNum } from "@/utils/convert";
import {
  CloseButton,
  Group,
  NumberInput,
  Select,
  Switch,
  TextInput,
} from "@mantine/core";

const Edit = (props: {
  item: CoreP.User;
  onUpdate: (item: CoreP.User) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.User>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.User.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.User;
    setReq(clone);

    props.onUpdate(clone);
  };

  if (!req) {
    return <></>;
  }

  return (
    <div>
      <Group grow>
        <Select
          label="Type"
          required
          description="A User has to be either a Human or a Workload"
          data={[
            {
              label: "Human",
              value: CoreP.User_Spec_Type[CoreP.User_Spec_Type.HUMAN],
            },
            {
              label: "Workload",
              value: CoreP.User_Spec_Type[CoreP.User_Spec_Type.WORKLOAD],
            },
          ]}
          defaultValue={
            CoreP.User_Spec_Type[req.spec!.type] ??
            CoreP.User_Spec_Type[CoreP.User_Spec_Type.HUMAN]
          }
          onChange={(v) => {
            const typ = CoreP.User_Spec_Type[v as "HUMAN" | "WORKLOAD"];
            req.spec!.type = typ;
            if (typ === CoreP.User_Spec_Type.WORKLOAD) {
              req.spec!.email = "";
            }
            updateReq();
          }}
        />

        <TextInput
          label="Email"
          placeholder="john@example.com"
          description="Set the User email"
          value={req.spec?.email}
          disabled={req.spec?.type !== CoreP.User_Spec_Type.HUMAN}
          onChange={(v) => {
            req.spec!.email = v.target.value;
            updateReq();
          }}
        />
      </Group>

      <Group grow>
        <SelectResourceMultiple
          api="core"
          kind="Group"
          label="Groups"
          description="Choose any number of Groups for the User to belong to"
          defaultValue={req.spec!.groups}
          clearable
          onChange={(v) => {
            req.spec!.groups = v?.map((x) => x.metadata!.name) ?? [];
            updateReq();
          }}
        />

        <Switch
          label="Disabled"
          description="Disable/deactivate the User"
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>

      <EditItem
        title="Authentication"
        description="Set Authentication-related Options"
        onUnset={() => {
          req.spec!.authentication = undefined;
          updateReq();
        }}
        obj={req.spec!.authentication}
        onSet={() => {
          if (!req.spec!.authentication) {
            req.spec!.authentication = CoreP.User_Spec_Authentication.create(
              {},
            );
            updateReq();
          }
        }}
      >
        {req.spec!.authentication && (
          <>
            <ItemMessage
              title="Identities"
              obj={req.spec!.authentication!.identities}
              isList
              onSet={() => {
                req.spec!.authentication!.identities = [
                  CoreP.User_Spec_Authentication_Identity.create(),
                ];
                updateReq();
              }}
              onAddListItem={() => {
                req.spec!.authentication?.identities.push(
                  CoreP.User_Spec_Authentication_Identity.create(),
                );
                updateReq();
              }}
            >
              {req.spec!.authentication.identities.map((identity, idx) => (
                <div className="w-full flex" key={idx}>
                  <CloseButton
                    size={"sm"}
                    variant="subtle"
                    onClick={() => {
                      req.spec!.authentication!.identities.splice(idx, 1);
                      updateReq();
                    }}
                  ></CloseButton>
                  <Group className="flex-1" grow>
                    <TextInput
                      required
                      label="Identifier"
                      description="Set the identifier value returned by the IdentityProvider during authentication (e.g. email)"
                      placeholder="linus"
                      value={
                        req.spec!.authentication!.identities[idx].identifier
                      }
                      onChange={(v) => {
                        req.spec!.authentication!.identities[idx].identifier =
                          v.target.value;
                        updateReq();
                      }}
                    />

                    <SelectResource
                      api="core"
                      kind="IdentityProvider"
                      description="Set the corresponding IdentityProvider"
                      labelDefault
                      defaultValue={
                        req.spec!.authentication!.identities[idx]
                          .identityProvider
                      }
                      onChange={(v) => {
                        req.spec!.authentication!.identities[
                          idx
                        ].identityProvider = v?.metadata?.name ?? "";
                        updateReq();
                      }}
                    />
                    {/**
                     <SelectIdentityProvider
                      defaultValue={
                        req.spec!.authentication!.identities[idx]
                          .identityProvider
                      }
                      onChange={(v) => {
                        req.spec!.authentication!.identities[
                          idx
                        ].identityProvider = v ?? "";
                        updateReq();
                      }}
                    />
                     **/}
                  </Group>
                </div>
              ))}
            </ItemMessage>
          </>
        )}
      </EditItem>

      <EditItem
        title="Authorization"
        description="Set the User Policies"
        onUnset={() => {
          req.spec!.authorization = undefined;
          updateReq();
        }}
        obj={req.spec!.authorization}
        onSet={() => {
          if (!req.spec!.authorization) {
            req.spec!.authorization = CoreP.User_Spec_Authorization.create({
              policies: [],
            });
            updateReq();
          }
        }}
      >
        {req.spec!.authorization && (
          <>
            <SelectPolicies
              policies={req.spec!.authorization.policies}
              onUpdate={(v) => {
                if (!v) {
                  req.spec!.authorization!.policies = [];
                } else {
                  req.spec!.authorization!.policies = v;
                }

                updateReq();
              }}
            />
            <SelectInlinePolicies
              inlinePolicies={req.spec!.authorization!.inlinePolicies}
              onUpdate={(v) => {
                req.spec!.authorization!.inlinePolicies = v;
                updateReq();
              }}
            />
          </>
        )}
      </EditItem>

      <EditItem
        title="Session"
        description="Set Session-related Options"
        onUnset={() => {
          req.spec!.session = undefined;
          updateReq();
        }}
        obj={req.spec!.session}
        onSet={() => {
          if (!req.spec!.session) {
            req.spec!.session = CoreP.User_Spec_Session.create({});
            updateReq();
          }
        }}
      >
        {req.spec!.session && (
          <>
            <Group grow>
              <DurationPicker
                value={req.spec!.session!.accessTokenDuration}
                title="Access Token Duration"
                onChange={(v) => {
                  req.spec!.session!.accessTokenDuration = v;
                  updateReq();
                }}
              />

              <DurationPicker
                value={req.spec!.session!.refreshTokenDuration}
                title="Refresh Token Duration"
                onChange={(v) => {
                  req.spec!.session!.refreshTokenDuration = v;
                  updateReq();
                }}
              />

              <DurationPicker
                value={req.spec!.session!.clientDuration}
                title="Session Client-based Duration"
                onChange={(v) => {
                  req.spec!.session!.clientDuration = v;
                  updateReq();
                }}
              />

              <DurationPicker
                value={req.spec!.session!.clientlessDuration}
                title="Session Clientless Duration"
                onChange={(v) => {
                  req.spec!.session!.clientlessDuration = v;
                  updateReq();
                }}
              />
            </Group>

            <Group grow>
              <NumberInput
                label="Max Per User"
                description="Set the max number of Sessions per User"
                defaultValue={req.spec!.session!.maxPerUser}
                min={1}
                max={100000}
                onChange={(v) => {
                  req.spec!.session!.maxPerUser = strToNum(v);
                }}
              />

              <Select
                label="Default State"
                description="Set the Session's default state to ACTIVE, PENDING or REJECTED"
                data={[
                  {
                    label: "Active",
                    value:
                      CoreP.Session_Spec_State[CoreP.Session_Spec_State.ACTIVE],
                  },
                  {
                    label: "Pending",
                    value:
                      CoreP.Session_Spec_State[
                        CoreP.Session_Spec_State.PENDING
                      ],
                  },
                  {
                    label: "Rejected",
                    value:
                      CoreP.Session_Spec_State[
                        CoreP.Session_Spec_State.REJECTED
                      ],
                  },
                ]}
                value={
                  CoreP.Session_Spec_State[req.spec!.session!.defaultState]
                }
                onChange={(v) => {
                  if (!v) {
                    return;
                  }
                  req.spec!.session!.defaultState =
                    CoreP.Session_Spec_State[v as "ACTIVE"];
                  updateReq();
                }}
              />
            </Group>
          </>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
