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
  ActionIcon,
  NumberInput,
  Select,
  Switch,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { Trash2 } from "lucide-react";

const createIdentityKey = () =>
  globalThis.crypto?.randomUUID?.() ??
  `identity-${Date.now()}-${Math.random().toString(36).slice(2)}`;

const Edit = (props: {
  item: CoreP.User;
  onUpdate: (item: CoreP.User) => void;
}) => {
  const [req, setReq] = React.useState<CoreP.User>(() =>
    CoreP.User.clone(props.item),
  );
  const identityKeys = React.useRef(
    props.item.spec?.authentication?.identities.map(createIdentityKey) ?? [],
  );
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      identityKeys.current =
        data.spec?.authentication?.identities.map(createIdentityKey) ?? [];
      setReq(CoreP.User.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.User;
    setReq(clone);

    props.onUpdate(clone);
  };

  return (
    <div className="space-y-7">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <Select
            label="Type"
            required
            description="Choose whether this principal represents a person or workload"
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
            value={
              CoreP.User_Spec_Type[req.spec!.type] ??
              CoreP.User_Spec_Type[CoreP.User_Spec_Type.HUMAN]
            }
            onChange={(value) => {
              if (!value) return;
              const type = CoreP.User_Spec_Type[value as "HUMAN" | "WORKLOAD"];
              req.spec!.type = type;
              if (type === CoreP.User_Spec_Type.WORKLOAD) req.spec!.email = "";
              updateReq();
            }}
          />

          <TextInput
            label="Email"
            placeholder="john@example.com"
            description="Used as a fallback identity for human users"
            value={req.spec?.email}
            disabled={req.spec?.type !== CoreP.User_Spec_Type.HUMAN}
            onChange={(event) => {
              req.spec!.email = event.currentTarget.value;
              updateReq();
            }}
          />
      </div>

      <div className="grid grid-cols-1 items-start gap-4 md:grid-cols-[minmax(0,1fr)_minmax(220px,0.7fr)]">
          <SelectResourceMultiple
            api="core"
            kind="Group"
            label="Groups"
            description="Choose the Groups this User belongs to"
            defaultValue={req.spec!.groups}
            clearable
            onChange={(value) => {
              req.spec!.groups =
                value?.map((resource) => resource.metadata!.name) ?? [];
              updateReq();
            }}
          />

          <div className="rounded-xl border border-slate-200 bg-slate-50/50 px-3.5 py-3">
            <Switch
              label="Disable user"
              color="red.8"
              checked={req.spec!.isDisabled}
              onChange={(event) => {
                req.spec!.isDisabled = event.currentTarget.checked;
                updateReq();
              }}
            />
          </div>
      </div>

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
          <div className="space-y-5">
            <Select
              label="Authenticator default state"
              description="Initial state assigned to newly registered authenticators"
              placeholder="Use the cluster default"
              clearable
              data={[
                {
                  label: "Active",
                  value:
                    CoreP.Authenticator_Spec_State[
                      CoreP.Authenticator_Spec_State.ACTIVE
                    ],
                },
                {
                  label: "Pending",
                  value:
                    CoreP.Authenticator_Spec_State[
                      CoreP.Authenticator_Spec_State.PENDING
                    ],
                },
                {
                  label: "Rejected",
                  value:
                    CoreP.Authenticator_Spec_State[
                      CoreP.Authenticator_Spec_State.REJECTED
                    ],
                },
              ]}
              value={
                req.spec!.authentication.authenticatorDefaultState ===
                CoreP.Authenticator_Spec_State.STATE_UNKNOWN
                  ? null
                  : CoreP.Authenticator_Spec_State[
                      req.spec!.authentication.authenticatorDefaultState
                    ]
              }
              onChange={(value) => {
                req.spec!.authentication!.authenticatorDefaultState = value
                  ? CoreP.Authenticator_Spec_State[
                      value as "ACTIVE" | "PENDING" | "REJECTED"
                    ]
                  : CoreP.Authenticator_Spec_State.STATE_UNKNOWN;
                updateReq();
              }}
            />

            <ItemMessage
              title="Identities"
              obj={req.spec!.authentication!.identities}
              isList
              onSet={() => {
                req.spec!.authentication!.identities = [
                  CoreP.User_Spec_Authentication_Identity.create(),
                ];
                identityKeys.current = [createIdentityKey()];
                updateReq();
              }}
              onAddListItem={() => {
                req.spec!.authentication?.identities.push(
                  CoreP.User_Spec_Authentication_Identity.create(),
                );
                identityKeys.current.push(createIdentityKey());
                updateReq();
              }}
            >
              <div className="space-y-3">
                {req.spec!.authentication.identities.map((identity, idx) => (
                  <div
                    className="relative rounded-xl border border-slate-200 bg-slate-50/40 p-3.5 pr-12"
                    key={identityKeys.current[idx]}
                  >
                    <Tooltip label="Remove identity" withArrow>
                      <ActionIcon
                        type="button"
                        variant="subtle"
                        color="red"
                        size="sm"
                        className="absolute right-3 top-3"
                        aria-label={`Remove external identity ${idx + 1}`}
                        onClick={() => {
                          req.spec!.authentication!.identities.splice(idx, 1);
                          identityKeys.current.splice(idx, 1);
                          updateReq();
                        }}
                      >
                        <Trash2 size={13} strokeWidth={2.1} />
                      </ActionIcon>
                    </Tooltip>

                    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                    <TextInput
                      required
                      label="Identifier"
                      description="Value returned by the Identity Provider, such as an email or username"
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
                      required
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
                    </div>
                  </div>
                ))}
              </div>
            </ItemMessage>
          </div>
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
          <div className="space-y-5">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
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
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <NumberInput
                label="Max Per User"
                description="Set the max number of Sessions per User"
                value={req.spec!.session!.maxPerUser}
                min={1}
                max={100000}
                onChange={(v) => {
                  req.spec!.session!.maxPerUser = strToNum(v);
                  updateReq();
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
                    CoreP.Session_Spec_State[
                      v as "ACTIVE" | "PENDING" | "REJECTED"
                    ];
                  updateReq();
                }}
              />
            </div>
          </div>
        )}
      </EditItem>

      {req.spec!.type === CoreP.User_Spec_Type.HUMAN && (
        <EditItem
          title="Profile"
          description="Optional human profile and contact information"
          obj={req.spec!.info}
          onSet={() => {
            req.spec!.info = CoreP.User_Spec_Info.create({});
            updateReq();
          }}
          onUnset={() => {
            req.spec!.info = undefined;
            updateReq();
          }}
        >
          {req.spec!.info && (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {[
                ["First name", "firstName"],
                ["Middle name", "middleName"],
                ["Last name", "lastName"],
                ["Phone", "phone"],
                ["Locale", "locale"],
                ["Country", "country"],
                ["Website", "website"],
              ].map(([label, key]) => (
                <TextInput
                  key={key}
                  label={label}
                  value={req.spec!.info![key as keyof CoreP.User_Spec_Info]}
                  onChange={(event) => {
                    req.spec!.info![key as keyof CoreP.User_Spec_Info] =
                      event.currentTarget.value;
                    updateReq();
                  }}
                />
              ))}
            </div>
          )}
        </EditItem>
      )}
    </div>
  );
};

export default Edit;
