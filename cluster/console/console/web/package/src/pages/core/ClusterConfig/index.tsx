import * as CoreP from "@/apis/corev1/corev1";
import Cond from "@/components/Condition";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import CopyText from "@/components/CopyText";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import { getClientCore } from "@/utils/client";
import { strToNum } from "@/utils/convert";
import { invalidateKey } from "@/utils/pb";
import {
  Alert,
  Button,
  Group,
  Loader,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  Tabs,
  TagsInput,
  TextInput,
} from "@mantine/core";
import { AlertTriangle, Network } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { toast } from "sonner";
import { match } from "ts-pattern";

const Edit = (props: {
  item: CoreP.ClusterConfig;
  onUpdate: (item: CoreP.ClusterConfig) => void;
}) => {
  const { item, onUpdate } = props;
  const cloneForEdit = React.useCallback((value: CoreP.ClusterConfig) => {
    const next = CoreP.ClusterConfig.clone(value);
    if (!next.spec) next.spec = CoreP.ClusterConfig_Spec.create();
    return next;
  }, []);
  const [req, setReq] = React.useState(() => cloneForEdit(item));
  const mmdbAuthTypes = React.useRef<Record<string, unknown>>({});
  const ruleIDs = React.useRef<Record<string, string[]>>({
    post: [],
    authentication: [],
    registration: [],
  });
  const getRuleID = (kind: string, index: number) => {
    const ids = ruleIDs.current[kind];
    while (ids.length <= index) ids.push(`${kind}-${crypto.randomUUID()}`);
    return ids[index];
  };
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    const next = cloneForEdit(item);
    setReq(next);
    mmdbAuthTypes.current = {};
    ruleIDs.current = { post: [], authentication: [], registration: [] };
  }, [cloneForEdit, itemKey]);

  const updateReq = () => {
    const next = CoreP.ClusterConfig.clone(req);
    setReq(next);
    onUpdate(CoreP.ClusterConfig.clone(next));
  };

  return (
    <div className="w-full">
      <div className="w-full py-2">
        <div className="w-full">
          <EditItem
            title="DNS"
            description="Configure the Cluster's private DNS service."
            onUnset={() => {
              req.spec!.dns = undefined;

              updateReq();
            }}
            obj={req.spec!.dns}
            onSet={() => {
              if (!req.spec!.dns) {
                req.spec!.dns = CoreP.ClusterConfig_Spec_DNS.create();
                updateReq();
              }
            }}
          >
            {req.spec!.dns && (
              <>
                <EditItem
                  title="Fallback Zone"
                  description="Configure upstream DNS servers for names not served by the Cluster."
                  onUnset={() => {
                    req.spec!.dns!.fallbackZone = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.dns!.fallbackZone}
                  onSet={() => {
                    if (!req.spec!.dns!.fallbackZone) {
                      req.spec!.dns!.fallbackZone =
                        CoreP.ClusterConfig_Spec_DNS_Zone.create();
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.dns!.fallbackZone && (
                    <Group grow>
                      <TagsInput
                        label="Servers"
                        placeholder="1.1.1.1, tls://8.8.8.8"
                        description="Upstream DNS servers; supports raw DNS addresses and DNS-over-TLS endpoints."
                        value={req.spec!.dns!.fallbackZone!.servers}
                        onChange={(v) => {
                          req.spec!.dns!.fallbackZone!.servers = v;
                          updateReq();
                        }}
                      />

                      <DurationPicker
                        value={req.spec!.dns!.fallbackZone.cacheDuration}
                        title="Cache Duration"
                        description="How long answers from the fallback zone remain cached."
                        onChange={(v) => {
                          req.spec!.dns!.fallbackZone!.cacheDuration = v;
                          updateReq();
                        }}
                      />
                    </Group>
                  )}
                </EditItem>
              </>
            )}
          </EditItem>
          <EditItem
            title="Gateway"
            description="Configure Cluster-wide Gateway options."
            onUnset={() => {
              req.spec!.gateway = undefined;
              updateReq();
            }}
            obj={req.spec!.gateway}
            onSet={() => {
              if (!req.spec!.gateway) {
                req.spec!.gateway = CoreP.ClusterConfig_Spec_Gateway.create();
                updateReq();
              }
            }}
          >
            {req.spec!.gateway && (
              <>
                <DurationPicker
                  value={req.spec!.gateway!.wireguardKeyRotationDuration}
                  title="WireGuard Key Rotation Duration"
                  description="How often Gateway WireGuard keys are rotated."
                  onChange={(v) => {
                    req.spec!.gateway!.wireguardKeyRotationDuration = v;
                    updateReq();
                  }}
                />
              </>
            )}
          </EditItem>

          <EditItem
            title="Authenticator"
            description="Configure Cluster-wide Authenticator and MFA options."
            onUnset={() => {
              req.spec!.authenticator = undefined;
              updateReq();
            }}
            obj={req.spec!.authenticator}
            onSet={() => {
              if (!req.spec!.authenticator) {
                req.spec!.authenticator =
                  CoreP.ClusterConfig_Spec_Authenticator.create();
                updateReq();
              }
            }}
          >
            {req.spec!.authenticator && (
              <div>
                <Group grow>
                  <Switch
                    label="Enable Passkey Login"
                    description="Allow Users to log in directly with supported resident-key Passkeys."
                    checked={req.spec!.authenticator!.enablePasskeyLogin}
                    onChange={(v) => {
                      req.spec!.authenticator!.enablePasskeyLogin =
                        v.target.checked;
                      updateReq();
                    }}
                  />

                  <Select
                    label="Default State"
                    description="Default state assigned to newly registered Authenticators."
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
                      CoreP.Authenticator_Spec_State[
                        req.spec!.authenticator!.defaultState
                      ]
                    }
                    onChange={(v) => {
                      if (!v) {
                        return;
                      }
                      req.spec!.authenticator!.defaultState =
                        CoreP.Authenticator_Spec_State[v as "ACTIVE"];
                      updateReq();
                    }}
                  />
                </Group>

                <EditItem
                  title="FIDO"
                  description="Configure WebAuthn/FIDO registration options."
                  onUnset={() => {
                    req.spec!.authenticator!.fido = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.authenticator.fido}
                  onSet={() => {
                    if (!req.spec!.authenticator!.fido) {
                      req.spec!.authenticator!.fido =
                        CoreP.ClusterConfig_Spec_Authenticator_FIDO.create();
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.authenticator!.fido && (
                    <div>
                      <Group grow>
                        <Select
                          label="Attestation Conveyance Preference"
                          description="Attestation conveyance preference used during WebAuthn registration."
                          data={[
                            {
                              label: "Direct",
                              value:
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference[
                                  CoreP
                                    .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference
                                    .DIRECT
                                ],
                            },
                            {
                              label: "Indirect",
                              value:
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference[
                                  CoreP
                                    .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference
                                    .INDIRECT
                                ],
                            },

                            {
                              label: "None",
                              value:
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference[
                                  CoreP
                                    .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference
                                    .NONE
                                ],
                            },
                            {
                              label: "Enterprise",
                              value:
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference[
                                  CoreP
                                    .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference
                                    .ENTERPRISE
                                ],
                            },
                          ]}
                          value={
                            CoreP
                              .ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference[
                              req.spec!.authenticator!.fido!
                                .attestationConveyancePreference
                            ]
                          }
                          onChange={(v) => {
                            if (!v) {
                              return;
                            }
                            req.spec!.authenticator!.fido!.attestationConveyancePreference =
                              CoreP.ClusterConfig_Spec_Authenticator_FIDO_AttestationConveyancePreference[
                                v as "DIRECT"
                              ];
                            updateReq();
                          }}
                        />
                      </Group>
                    </div>
                  )}
                </EditItem>

                <ItemMessage
                  title="Post-Authentication Rules"
                  obj={req.spec!.authenticator!.postAuthenticationRules}
                  isList
                  onSet={() => {
                    req.spec!.authenticator!.postAuthenticationRules = [
                      CoreP.ClusterConfig_Spec_Authenticator_Rule.create({
                        condition: CoreP.Condition.create({
                          type: { oneofKind: "match", match: "" },
                        }),
                      }),
                    ];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    req.spec!.authenticator!.postAuthenticationRules.push(
                      CoreP.ClusterConfig_Spec_Authenticator_Rule.create({
                        condition: CoreP.Condition.create({
                          type: { oneofKind: "match", match: "" },
                        }),
                      }),
                    );
                    updateReq();
                  }}
                >
                  {req.spec!.authenticator!.postAuthenticationRules &&
                    req.spec!.authenticator!.postAuthenticationRules.map(
                      (rule, ruleIdx) => (
                        <EditItem
                          key={getRuleID("post", ruleIdx)}
                          obj={
                            req.spec!.authenticator!.postAuthenticationRules[
                              ruleIdx
                            ]
                          }
                          onUnset={() => {
                            req.spec!.authenticator!.postAuthenticationRules.splice(
                              ruleIdx,
                              1,
                            );
                            ruleIDs.current.post.splice(ruleIdx, 1);
                            updateReq();
                          }}
                        >
                          <Group grow>
                            <TextInput
                              label="Name"
                              description="Set an optional name for the rule (shown in Logs)"
                              placeholder="my-rule"
                              value={
                                req.spec!.authenticator!
                                  .postAuthenticationRules[ruleIdx].name
                              }
                              onChange={(v) => {
                                req.spec!.authenticator!.postAuthenticationRules[
                                  ruleIdx
                                ].name = v.target.value;
                                updateReq();
                              }}
                            />

                            <Select
                              label="Effect"
                              required
                              description="Set the effect to either ALLOW or DENY"
                              data={[
                                {
                                  label: "Allow",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_Rule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_Rule_Effect
                                        .ALLOW
                                    ],
                                },
                                {
                                  label: "Deny",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_Rule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_Rule_Effect
                                        .DENY
                                    ],
                                },
                              ]}
                              defaultValue={
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_Rule_Effect[
                                  req.spec!.authenticator!
                                    .postAuthenticationRules[ruleIdx].effect
                                ]
                              }
                              onChange={(v) => {
                                req.spec!.authenticator!.postAuthenticationRules[
                                  ruleIdx
                                ].effect =
                                  CoreP.ClusterConfig_Spec_Authenticator_Rule_Effect[
                                    v as "ALLOW"
                                  ];
                                updateReq();
                              }}
                            />
                          </Group>

                          <Cond
                            item={
                              req.spec!.authenticator!.postAuthenticationRules[
                                ruleIdx
                              ].condition ??
                              CoreP.Condition.create({
                                type: { oneofKind: "match", match: "" },
                              })
                            }
                            onChange={(v) => {
                              req.spec!.authenticator!.postAuthenticationRules[
                                ruleIdx
                              ].condition = v;
                              updateReq();
                            }}
                          />
                        </EditItem>
                      ),
                    )}
                </ItemMessage>

                <ItemMessage
                  title="Authentication Enforcement Rules"
                  obj={req.spec!.authenticator!.authenticationEnforcementRules}
                  isList
                  onSet={() => {
                    req.spec!.authenticator!.authenticationEnforcementRules = [
                      CoreP.ClusterConfig_Spec_Authenticator_EnforcementRule.create(
                        {
                          condition: CoreP.Condition.create({
                            type: { oneofKind: "match", match: "" },
                          }),
                        },
                      ),
                    ];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    req.spec!.authenticator!.authenticationEnforcementRules.push(
                      CoreP.ClusterConfig_Spec_Authenticator_EnforcementRule.create(
                        {
                          condition: CoreP.Condition.create({
                            type: { oneofKind: "match", match: "" },
                          }),
                        },
                      ),
                    );
                    updateReq();
                  }}
                >
                  {req.spec!.authenticator!.authenticationEnforcementRules &&
                    req.spec!.authenticator!.authenticationEnforcementRules.map(
                      (rule, ruleIdx) => (
                        <EditItem
                          key={getRuleID("authentication", ruleIdx)}
                          obj={
                            req.spec!.authenticator!
                              .authenticationEnforcementRules[ruleIdx]
                          }
                          onUnset={() => {
                            req.spec!.authenticator!.authenticationEnforcementRules.splice(
                              ruleIdx,
                              1,
                            );
                            ruleIDs.current.authentication.splice(ruleIdx, 1);
                            updateReq();
                          }}
                        >
                          <Group grow>
                            <Select
                              label="Effect"
                              required
                              description="Set the effect to ENFORCE, IGNORE or RECOMMEND"
                              data={[
                                {
                                  label: "Enforce",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect
                                        .ENFORCE
                                    ],
                                },
                                {
                                  label: "Ignore",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect
                                        .IGNORE
                                    ],
                                },
                                {
                                  label: "Recommend",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect
                                        .RECOMMEND
                                    ],
                                },
                              ]}
                              defaultValue={
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                  req.spec!.authenticator!
                                    .authenticationEnforcementRules[ruleIdx]
                                    .effect
                                ]
                              }
                              onChange={(v) => {
                                req.spec!.authenticator!.authenticationEnforcementRules[
                                  ruleIdx
                                ].effect =
                                  CoreP.ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                    v as "ENFORCE"
                                  ];
                                updateReq();
                              }}
                            />
                          </Group>

                          <Cond
                            item={
                              req.spec!.authenticator!
                                .authenticationEnforcementRules[ruleIdx]
                                .condition ??
                              CoreP.Condition.create({
                                type: { oneofKind: "match", match: "" },
                              })
                            }
                            onChange={(v) => {
                              req.spec!.authenticator!.authenticationEnforcementRules[
                                ruleIdx
                              ].condition = v;
                              updateReq();
                            }}
                          />
                        </EditItem>
                      ),
                    )}
                </ItemMessage>

                <ItemMessage
                  title="Registration Enforcement Rules"
                  obj={req.spec!.authenticator!.registrationEnforcementRules}
                  isList
                  onSet={() => {
                    req.spec!.authenticator!.registrationEnforcementRules = [
                      CoreP.ClusterConfig_Spec_Authenticator_EnforcementRule.create(
                        {
                          condition: CoreP.Condition.create({
                            type: { oneofKind: "match", match: "" },
                          }),
                        },
                      ),
                    ];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    req.spec!.authenticator!.registrationEnforcementRules.push(
                      CoreP.ClusterConfig_Spec_Authenticator_EnforcementRule.create(
                        {
                          condition: CoreP.Condition.create({
                            type: { oneofKind: "match", match: "" },
                          }),
                        },
                      ),
                    );
                    updateReq();
                  }}
                >
                  {req.spec!.authenticator!.registrationEnforcementRules &&
                    req.spec!.authenticator!.registrationEnforcementRules.map(
                      (rule, ruleIdx) => (
                        <EditItem
                          key={getRuleID("registration", ruleIdx)}
                          obj={
                            req.spec!.authenticator!
                              .registrationEnforcementRules[ruleIdx]
                          }
                          onUnset={() => {
                            req.spec!.authenticator!.registrationEnforcementRules.splice(
                              ruleIdx,
                              1,
                            );
                            ruleIDs.current.registration.splice(ruleIdx, 1);
                            updateReq();
                          }}
                        >
                          <Group grow>
                            <Select
                              label="Effect"
                              required
                              description="Set the effect to ENFORCE, IGNORE or RECOMMEND"
                              data={[
                                {
                                  label: "Enforce",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect
                                        .ENFORCE
                                    ],
                                },
                                {
                                  label: "Ignore",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect
                                        .IGNORE
                                    ],
                                },
                                {
                                  label: "Recommend",
                                  value:
                                    CoreP
                                      .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                      CoreP
                                        .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect
                                        .RECOMMEND
                                    ],
                                },
                              ]}
                              defaultValue={
                                CoreP
                                  .ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                  req.spec!.authenticator!
                                    .registrationEnforcementRules[ruleIdx]
                                    .effect
                                ]
                              }
                              onChange={(v) => {
                                req.spec!.authenticator!.registrationEnforcementRules[
                                  ruleIdx
                                ].effect =
                                  CoreP.ClusterConfig_Spec_Authenticator_EnforcementRule_Effect[
                                    v as "ENFORCE"
                                  ];
                                updateReq();
                              }}
                            />
                          </Group>

                          <Cond
                            item={
                              req.spec!.authenticator!
                                .registrationEnforcementRules[ruleIdx]
                                .condition ??
                              CoreP.Condition.create({
                                type: { oneofKind: "match", match: "" },
                              })
                            }
                            onChange={(v) => {
                              req.spec!.authenticator!.registrationEnforcementRules[
                                ruleIdx
                              ].condition = v;
                              updateReq();
                            }}
                          />
                        </EditItem>
                      ),
                    )}
                </ItemMessage>
              </div>
            )}
          </EditItem>

          <EditItem
            title="Authorization"
            description="Configure Policies applied to every request in the Cluster."
            onUnset={() => {
              req.spec!.authorization = undefined;
              updateReq();
            }}
            obj={req.spec!.authorization}
            onSet={() => {
              if (!req.spec!.authorization) {
                req.spec!.authorization =
                  CoreP.ClusterConfig_Spec_Authorization.create();
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
                  inlinePolicies={req.spec!.authorization.inlinePolicies}
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
            description="Configure Cluster-wide defaults for Sessions."
            onUnset={() => {
              req.spec!.session = undefined;
              updateReq();
            }}
            obj={req.spec!.session}
            onSet={() => {
              if (!req.spec!.session) {
                req.spec!.session = CoreP.ClusterConfig_Spec_Session.create();
                updateReq();
              }
            }}
          >
            {req.spec!.session && (
              <>
                <EditItem
                  title="Human"
                  description="Session defaults for HUMAN Users."
                  onUnset={() => {
                    req.spec!.session!.human = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.session!.human}
                  onSet={() => {
                    if (!req.spec!.session!.human) {
                      req.spec!.session!.human =
                        CoreP.ClusterConfig_Spec_Session_Human.create();
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.session!.human && (
                    <>
                      <Group grow>
                        <DurationPicker
                          value={req.spec!.session!.human!.accessTokenDuration}
                          title="Access Token Duration"
                          description="How long access tokens issued for human sessions remain valid."
                          onChange={(v) => {
                            req.spec!.session!.human!.accessTokenDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.human!.refreshTokenDuration}
                          title="Refresh Token Duration"
                          description="How long refresh tokens issued for human sessions remain valid."
                          onChange={(v) => {
                            req.spec!.session!.human!.refreshTokenDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.human!.clientDuration}
                          title="Client-base Session Duration"
                          description="How long client-based human sessions remain active before expiring."
                          onChange={(v) => {
                            req.spec!.session!.human!.clientDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.human!.clientlessDuration}
                          title="Clientless Session Duration"
                          description="How long clientless human sessions remain active before expiring."
                          onChange={(v) => {
                            req.spec!.session!.human!.clientlessDuration = v;
                            updateReq();
                          }}
                        />
                      </Group>

                      <Group grow>
                        <NumberInput
                          label="Max Per User"
                          description="Set the max number of Sessions per User"
                          value={req.spec!.session!.human!.maxPerUser}
                          min={1}
                          max={100000}
                          onChange={(v) => {
                            req.spec!.session!.human!.maxPerUser = strToNum(v);
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
                                CoreP.Session_Spec_State[
                                  CoreP.Session_Spec_State.ACTIVE
                                ],
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
                            CoreP.Session_Spec_State[
                              req.spec!.session!.human!.defaultState
                            ]
                          }
                          onChange={(v) => {
                            if (!v) {
                              return;
                            }
                            req.spec!.session!.human!.defaultState =
                              CoreP.Session_Spec_State[v as "ACTIVE"];
                            updateReq();
                          }}
                        />
                      </Group>
                    </>
                  )}
                </EditItem>

                <EditItem
                  title="Workload"
                  description="Session defaults for WORKLOAD Users."
                  onUnset={() => {
                    req.spec!.session!.workload = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.session!.workload}
                  onSet={() => {
                    if (!req.spec!.session!.workload) {
                      req.spec!.session!.workload =
                        CoreP.ClusterConfig_Spec_Session_Workload.create();
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.session!.workload && (
                    <>
                      <Group grow>
                        <DurationPicker
                          value={
                            req.spec!.session!.workload!.accessTokenDuration
                          }
                          title="Access Token Duration"
                          description="How long access tokens issued for workload sessions remain valid."
                          onChange={(v) => {
                            req.spec!.session!.workload!.accessTokenDuration =
                              v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={
                            req.spec!.session!.workload!.refreshTokenDuration
                          }
                          title="Refresh Token Duration"
                          description="How long refresh tokens issued for workload sessions remain valid."
                          onChange={(v) => {
                            req.spec!.session!.workload!.refreshTokenDuration =
                              v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.workload!.clientDuration}
                          title="Client-base Session Duration"
                          description="How long client-based workload sessions remain active before expiring."
                          onChange={(v) => {
                            req.spec!.session!.workload!.clientDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={
                            req.spec!.session!.workload!.clientlessDuration
                          }
                          title="Clientless Session Duration"
                          description="How long clientless workload sessions remain active before expiring."
                          onChange={(v) => {
                            req.spec!.session!.workload!.clientlessDuration = v;
                            updateReq();
                          }}
                        />
                      </Group>

                      <Group grow>
                        <NumberInput
                          label="Max Per User"
                          description="Set the max number of Sessions per User"
                          value={req.spec!.session!.workload!.maxPerUser}
                          min={1}
                          max={100000}
                          onChange={(v) => {
                            req.spec!.session!.workload!.maxPerUser =
                              strToNum(v);
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
                                CoreP.Session_Spec_State[
                                  CoreP.Session_Spec_State.ACTIVE
                                ],
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
                            CoreP.Session_Spec_State[
                              req.spec!.session!.workload!.defaultState
                            ]
                          }
                          onChange={(v) => {
                            if (!v) {
                              return;
                            }
                            req.spec!.session!.workload!.defaultState =
                              CoreP.Session_Spec_State[v as "ACTIVE"];
                            updateReq();
                          }}
                        />
                      </Group>
                    </>
                  )}
                </EditItem>
              </>
            )}
          </EditItem>

          <EditItem
            title="Device"
            description="Configure Cluster-wide defaults for Devices."
            onUnset={() => {
              req.spec!.device = undefined;
              updateReq();
            }}
            obj={req.spec!.device}
            onSet={() => {
              if (!req.spec!.device) {
                req.spec!.device = CoreP.ClusterConfig_Spec_Device.create();
                updateReq();
              }
            }}
          >
            {req.spec!.device && (
              <>
                <EditItem
                  title="Human"
                  description="Device defaults for HUMAN Users."
                  onUnset={() => {
                    req.spec!.device!.human = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.device!.human}
                  onSet={() => {
                    if (!req.spec!.device!.human) {
                      req.spec!.device!.human =
                        CoreP.ClusterConfig_Spec_Device_Human.create();
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.device!.human && (
                    <Group grow>
                      <NumberInput
                        label="Max Per User"
                        description="Set the max number of Devices per User"
                        value={req.spec!.device!.human!.maxPerUser}
                        min={1}
                        max={100000}
                        onChange={(v) => {
                          req.spec!.device!.human!.maxPerUser = strToNum(v);
                          updateReq();
                        }}
                      />

                      <Select
                        label="Default State"
                        description="Set the Device's default state to ACTIVE, PENDING or REJECTED"
                        data={[
                          {
                            label: "Active",
                            value:
                              CoreP.Device_Spec_State[
                                CoreP.Device_Spec_State.ACTIVE
                              ],
                          },
                          {
                            label: "Pending",
                            value:
                              CoreP.Device_Spec_State[
                                CoreP.Device_Spec_State.PENDING
                              ],
                          },
                          {
                            label: "Rejected",
                            value:
                              CoreP.Device_Spec_State[
                                CoreP.Device_Spec_State.REJECTED
                              ],
                          },
                        ]}
                        value={
                          CoreP.Device_Spec_State[
                            req.spec!.device!.human!.defaultState
                          ]
                        }
                        onChange={(v) => {
                          if (!v) {
                            return;
                          }
                          req.spec!.device!.human!.defaultState =
                            CoreP.Device_Spec_State[v as "ACTIVE"];
                          updateReq();
                        }}
                      />
                    </Group>
                  )}
                </EditItem>

                <EditItem
                  title="Workload"
                  description="Device defaults for WORKLOAD Users."
                  onUnset={() => {
                    req.spec!.device!.workload = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.device!.workload}
                  onSet={() => {
                    if (!req.spec!.device!.workload) {
                      req.spec!.device!.workload =
                        CoreP.ClusterConfig_Spec_Device_Workload.create();
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.device!.workload && (
                    <Group grow>
                      <NumberInput
                        label="Max Per User"
                        description="Set the max number of Devices per User"
                        value={req.spec!.device!.workload!.maxPerUser}
                        min={1}
                        max={100000}
                        onChange={(v) => {
                          req.spec!.device!.workload!.maxPerUser = strToNum(v);
                          updateReq();
                        }}
                      />

                      <Select
                        label="Default State"
                        description="Set the Device's default state to ACTIVE, PENDING or REJECTED"
                        data={[
                          {
                            label: "Active",
                            value:
                              CoreP.Device_Spec_State[
                                CoreP.Device_Spec_State.ACTIVE
                              ],
                          },
                          {
                            label: "Pending",
                            value:
                              CoreP.Device_Spec_State[
                                CoreP.Device_Spec_State.PENDING
                              ],
                          },
                          {
                            label: "Rejected",
                            value:
                              CoreP.Device_Spec_State[
                                CoreP.Device_Spec_State.REJECTED
                              ],
                          },
                        ]}
                        value={
                          CoreP.Device_Spec_State[
                            req.spec!.device!.workload!.defaultState
                          ]
                        }
                        onChange={(v) => {
                          if (!v) {
                            return;
                          }
                          req.spec!.device!.workload!.defaultState =
                            CoreP.Device_Spec_State[v as "ACTIVE"];
                          updateReq();
                        }}
                      />
                    </Group>
                  )}
                </EditItem>
              </>
            )}
          </EditItem>

          <EditItem
            title="Ingress"
            description="Configure the internet-facing Ingress and downstream IP detection."
            onUnset={() => {
              req.spec!.ingress = undefined;
              updateReq();
            }}
            obj={req.spec!.ingress}
            onSet={() => {
              if (!req.spec!.ingress) {
                req.spec!.ingress = CoreP.ClusterConfig_Spec_Ingress.create();
                updateReq();
              }
            }}
          >
            {req.spec!.ingress && (
              <div className="space-y-4">
              {req.spec!.ingress.useForwardedForHeader && <Alert color="amber" icon={<AlertTriangle size={15} />} title="Only trust known reverse proxies">An incorrect trusted-hop count can allow clients to spoof forwarding information and their apparent source address.</Alert>}
              <div className="grid gap-4 md:grid-cols-2">
                <Switch
                  label="Use X-Forwarded-For Header"
                  description="Use the X-Forwarded-For header to determine the downstream public IP address. Only trust configured reverse proxies."
                  checked={req.spec!.ingress!.useForwardedForHeader}
                  onChange={(v) => {
                    req.spec!.ingress!.useForwardedForHeader = v.target.checked;

                    updateReq();
                  }}
                />

                <NumberInput
                  label="X-Forwarded-For trusted Hops"
                  description="Number of trusted proxies between Octelium ingress and the downstream client."
                  value={req.spec!.ingress!.xffNumTrustedHops}
                  min={0}
                  max={100}
                  onChange={(v) => {
                    req.spec!.ingress!.xffNumTrustedHops = strToNum(v);
                    updateReq();
                  }}
                />
              </div>
              </div>
            )}
          </EditItem>

          <EditItem
            title="Authentication"
            description="Configure options used during the authentication process."
            onUnset={() => {
              req.spec!.authentication = undefined;
              updateReq();
            }}
            obj={req.spec!.authentication}
            onSet={() => {
              if (!req.spec!.authentication) {
                req.spec!.authentication =
                  CoreP.ClusterConfig_Spec_Authentication.create();
                updateReq();
              }
            }}
          >
            {req.spec!.authentication && (
              <div>
                <EditItem
                  title="Geolocation"
                  description="Resolve client geolocation during authentication for use in Policies."
                  onUnset={() => {
                    req.spec!.authentication!.geolocation = undefined;
                    updateReq();
                  }}
                  obj={req.spec!.authentication!.geolocation}
                  onSet={() => {
                    if (!req.spec!.authentication!.geolocation) {
                      req.spec!.authentication!.geolocation =
                        CoreP.ClusterConfig_Spec_Authentication_Geolocation.create(
                          {
                            type: {
                              oneofKind: "mmdb",
                              mmdb: {
                                type: {
                                  oneofKind: `upstream`,
                                  upstream: {
                                    url: "",
                                  },
                                },
                              },
                            },
                          },
                        );
                      updateReq();
                    }
                  }}
                >
                  {req.spec!.authentication!.geolocation && (
                    <div>
                      {match(req.spec!.authentication!.geolocation.type)
                        .when(
                          (x) => x.oneofKind === `mmdb`,
                          (mmdb) => {
                            return (
                              <div>
                                {match(mmdb.mmdb.type)
                                  .when(
                                    (x) => x.oneofKind === `upstream`,
                                    (upstream) => {
                                      const changeMMDBAuth = (v: string | null) => {
                                        if (!upstream.upstream.auth || !v) return;
                                        const currentKind = upstream.upstream.auth.type.oneofKind;
                                        if (currentKind) mmdbAuthTypes.current[currentKind] = structuredClone(upstream.upstream.auth.type);
                                        const cached = mmdbAuthTypes.current[v] as typeof upstream.upstream.auth.type | undefined;
                                        if (cached) {
                                          upstream.upstream.auth.type = structuredClone(cached);
                                          updateReq();
                                          return;
                                        }
                                        match(v)
                                          .with("bearer", () => { upstream.upstream.auth!.type = { oneofKind: "bearer", bearer: CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Bearer.create({ type: { oneofKind: "fromSecret", fromSecret: "" } }) }; })
                                          .with("basic", () => { upstream.upstream.auth!.type = { oneofKind: "basic", basic: CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Basic.create({ password: { type: { oneofKind: "fromSecret", fromSecret: "" } } }) }; })
                                          .with("custom", () => { upstream.upstream.auth!.type = { oneofKind: "custom", custom: CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Custom.create({ value: { type: { oneofKind: "fromSecret", fromSecret: "" } } }) }; })
                                          .with("query", () => { upstream.upstream.auth!.type = { oneofKind: "query", query: CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Query.create({ value: { type: { oneofKind: "fromSecret", fromSecret: "" } } }) }; })
                                          .otherwise(() => {});
                                        updateReq();
                                      };
                                      return (
                                        <div>
                                          <Group grow>
                                            <TextInput
                                              label="URL"
                                              placeholder="https://mmdb.example/country-db-v1.0.0"
                                              description="URL from which the MaxMind MMDB database is fetched."
                                              value={upstream.upstream.url}
                                              onChange={(v) => {
                                                upstream.upstream.url =
                                                  v.target.value;
                                                updateReq();
                                              }}
                                            />
                                          </Group>

                                          <EditItem
                                            title="Authentication"
                                            description="Credentials used to fetch the MaxMind MMDB database."
                                            onUnset={() => {
                                              upstream.upstream.auth =
                                                undefined;
                                              updateReq();
                                            }}
                                            obj={upstream.upstream.auth}
                                            onSet={() => {
                                              upstream.upstream.auth =
                                                CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth.create(
                                                  {
                                                    type: {
                                                      oneofKind: "bearer",
                                                      bearer: {
                                                        type: {
                                                          oneofKind:
                                                            "fromSecret",
                                                          fromSecret: "",
                                                        },
                                                      },
                                                    },
                                                  },
                                                );
                                              updateReq();
                                            }}
                                          >
                                            {upstream.upstream.auth && (
                                              <Tabs
                                                value={
                                                  upstream.upstream.auth.type
                                                    .oneofKind
                                                }
                                                onChange={changeMMDBAuth}
                                              >
                                                <SegmentedControl
                                                  fullWidth
                                                  value={upstream.upstream.auth.type.oneofKind ?? "bearer"}
                                                  onChange={changeMMDBAuth}
                                                  data={[
                                                    { label: "Bearer", value: "bearer" },
                                                    { label: "Basic", value: "basic" },
                                                    { label: "Custom header", value: "custom" },
                                                    { label: "Query", value: "query" },
                                                  ]}
                                                />
                                                <Tabs.Panel value="bearer">
                                                  {match(
                                                    upstream.upstream.auth.type,
                                                  )
                                                    .when(
                                                      (x) =>
                                                        x.oneofKind ===
                                                        "bearer",
                                                      (bearer) => (
                                                        <SelectResource
                                                          api="core"
                                                          kind="Secret"
                                                          required
                                                          label="Bearer token Secret"
                                                          description="Select the Secret of the bearer token"
                                                          defaultValue={
                                                            bearer.bearer.type
                                                              .oneofKind ===
                                                            "fromSecret"
                                                              ? bearer.bearer
                                                                  .type
                                                                  .fromSecret
                                                              : undefined
                                                          }
                                                          onChange={(val) => {
                                                            match(
                                                              bearer.bearer
                                                                .type,
                                                            ).when(
                                                              (x) =>
                                                                x.oneofKind ===
                                                                "fromSecret",
                                                              (x) => {
                                                                x.fromSecret =
                                                                  val?.metadata?.name ?? "";
                                                              },
                                                            );
                                                            updateReq();
                                                          }}
                                                        />
                                                      ),
                                                    )
                                                    .otherwise(() => (
                                                      <></>
                                                    ))}
                                                </Tabs.Panel>

                                                <Tabs.Panel value="basic">
                                                  {match(
                                                    upstream.upstream.auth.type,
                                                  )
                                                    .when(
                                                      (x) =>
                                                        x.oneofKind === "basic",
                                                      (basic) => (
                                                        <Group grow>
                                                          <TextInput
                                                            label="Username"
                                                            description="Username for HTTP Basic authentication to the MMDB upstream."
                                                            placeholder="user1234"
                                                            value={
                                                              basic.basic
                                                                .username
                                                            }
                                                            onChange={(v) => {
                                                              basic.basic.username =
                                                                v.target.value;
                                                              updateReq();
                                                            }}
                                                          />
                                                          {match(
                                                            basic.basic.password
                                                              ?.type,
                                                          )
                                                            .when(
                                                              (x) =>
                                                                x?.oneofKind ===
                                                                "fromSecret",
                                                              (x) => (
                                                                <SelectResource
                                                                  api="core"
                                                                  kind="Secret"
                                                                  required
                                                                  label="Password Secret"
                                                                  description="Select the Secret of the password"
                                                                  defaultValue={
                                                                    x.fromSecret
                                                                  }
                                                                  onChange={(
                                                                    v,
                                                                  ) => {
                                                                    x.fromSecret =
                                                                      v?.metadata?.name ?? "";
                                                                    updateReq();
                                                                  }}
                                                                />
                                                              ),
                                                            )
                                                            .otherwise(() => (
                                                              <></>
                                                            ))}
                                                        </Group>
                                                      ),
                                                    )
                                                    .otherwise(() => (
                                                      <></>
                                                    ))}
                                                </Tabs.Panel>

                                                <Tabs.Panel value="custom">
                                                  {match(
                                                    upstream.upstream.auth.type,
                                                  )
                                                    .when(
                                                      (x) =>
                                                        x.oneofKind ===
                                                        "custom",
                                                      (custom) => (
                                                        <Group grow>
                                                          <TextInput
                                                            label="Header Name"
                                                            description="HTTP header used to carry the custom MMDB credential."
                                                            placeholder="X-Custom-Auth"
                                                            value={
                                                              custom.custom
                                                                .header
                                                            }
                                                            onChange={(v) => {
                                                              custom.custom.header =
                                                                v.target.value;
                                                              updateReq();
                                                            }}
                                                          />
                                                          {match(
                                                            custom.custom.value
                                                              ?.type,
                                                          )
                                                            .when(
                                                              (x) =>
                                                                x?.oneofKind ===
                                                                "fromSecret",
                                                              (x) => (
                                                                <SelectResource
                                                                  api="core"
                                                                  kind="Secret"
                                                                  required
                                                                  label="Header value Secret"
                                                                  description="Select the Secret of the header value"
                                                                  defaultValue={
                                                                    x.fromSecret
                                                                  }
                                                                  onChange={(
                                                                    v,
                                                                  ) => {
                                                                    x.fromSecret =
                                                                      v?.metadata?.name ?? "";
                                                                    updateReq();
                                                                  }}
                                                                />
                                                              ),
                                                            )
                                                            .otherwise(() => (
                                                              <></>
                                                            ))}
                                                        </Group>
                                                      ),
                                                    )
                                                    .otherwise(() => (
                                                      <></>
                                                    ))}
                                                </Tabs.Panel>

                                                <Tabs.Panel value="query">
                                                  {match(
                                                    upstream.upstream.auth.type,
                                                  )
                                                    .when(
                                                      (x) =>
                                                        x.oneofKind === "query",
                                                      (query) => (
                                                        <Group grow>
                                                          <TextInput
                                                            label="Query Key"
                                                            description="Query parameter used to carry the MMDB credential."
                                                            placeholder="api_key"
                                                            value={
                                                              query.query.key
                                                            }
                                                            onChange={(v) => {
                                                              query.query.key =
                                                                v.target.value;
                                                              updateReq();
                                                            }}
                                                          />
                                                          {match(
                                                            query.query.value
                                                              ?.type,
                                                          )
                                                            .when(
                                                              (x) =>
                                                                x?.oneofKind ===
                                                                "fromSecret",
                                                              (x) => (
                                                                <SelectResource
                                                                  api="core"
                                                                  kind="Secret"
                                                                  required
                                                                  label="Query value Secret"
                                                                  description="Select the Secret of the query value"
                                                                  defaultValue={
                                                                    x.fromSecret
                                                                  }
                                                                  onChange={(
                                                                    v,
                                                                  ) => {
                                                                    x.fromSecret =
                                                                      v?.metadata?.name ?? "";
                                                                    updateReq();
                                                                  }}
                                                                />
                                                              ),
                                                            )
                                                            .otherwise(() => (
                                                              <></>
                                                            ))}
                                                        </Group>
                                                      ),
                                                    )
                                                    .otherwise(() => (
                                                      <></>
                                                    ))}
                                                </Tabs.Panel>
                                              </Tabs>
                                            )}
                                          </EditItem>
                                        </div>
                                      );
                                    },
                                  )
                                  .otherwise(() => (
                                    <></>
                                  ))}
                              </div>
                            );
                          },
                        )
                        .otherwise(() => (
                          <></>
                        ))}
                    </div>
                  )}
                </EditItem>
              </div>
            )}
          </EditItem>
        </div>
      </div>
    </div>
  );
};

export default () => {
  const query = useQuery({
    queryKey: ["core", "clusterconfig"],
    queryFn: async () => getClientCore().getClusterConfig({}),
  });

  if (query.isLoading) return <div className="flex min-h-64 items-center justify-center"><Loader size="sm" color="gray" /></div>;
  if (query.isError) return <div className="mx-auto mt-10 max-w-xl"><Alert color="red" title="Could not load cluster configuration" icon={<AlertTriangle size={16} />}><div className="space-y-3"><div className="text-xs">{query.error.message}</div><Button type="button" size="compact-sm" variant="outline" onClick={() => query.refetch()}>Retry</Button></div></Alert></div>;
  if (!query.data?.response) return <div className="py-16 text-center text-sm font-semibold text-slate-500">Cluster configuration is unavailable.</div>;

  const item = query.data.response;
  const status = item.status;
  const network = status?.network;
  const networkConfig = status?.networkConfig;
  const mode = networkConfig ? CoreP.ClusterConfig_Status_NetworkConfig_Mode[networkConfig.mode] : undefined;

  return (
    <div className="space-y-5">
      {status && <section className="rounded-xl border border-slate-200 bg-white p-4"><div className="mb-3 flex items-center gap-2 text-sm font-bold text-slate-800"><Network size={15} /> Cluster status</div><div className="grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-3 [&>div>:last-child]:font-bold">
        {status.domain && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Domain</div><CopyText value={status.domain} /></div>}
        {mode && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Network mode</div><div className="mt-1 text-sm font-semibold text-slate-700">{mode.replaceAll("_", " ")}</div></div>}
        {network?.clusterNetwork?.v4 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Cluster IPv4 network</div><CopyText value={network.clusterNetwork.v4} /></div>}
        {network?.clusterNetwork?.v6 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Cluster IPv6 network</div><CopyText value={network.clusterNetwork.v6} /></div>}
        {network?.serviceSubnet?.v4 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Service IPv4 subnet</div><CopyText value={network.serviceSubnet.v4} /></div>}
        {network?.serviceSubnet?.v6 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Service IPv6 subnet</div><CopyText value={network.serviceSubnet.v6} /></div>}
        {network?.wgConnSubnet?.v4 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">WireGuard connection subnet</div><CopyText value={network.wgConnSubnet.v4} /></div>}
        {network?.quicConnSubnet?.v4 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">QUIC connection subnet</div><CopyText value={network.quicConnSubnet.v4} /></div>}
        {networkConfig?.v4?.clusterNetwork && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Configured IPv4 network</div><CopyText value={networkConfig.v4.clusterNetwork} /></div>}
        {networkConfig?.v6?.clusterNetwork && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Configured IPv6 network</div><CopyText value={networkConfig.v6.clusterNetwork} /></div>}
        {networkConfig?.wireguard && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">WireGuard</div><div className="mt-1 text-sm font-semibold text-slate-700">Port {networkConfig.wireguard.gatewayPort || "default"} · MTU {networkConfig.wireguard.mtu || "default"}</div></div>}
        {networkConfig?.quicv0 && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">QUICv0</div><div className="mt-1 text-sm font-semibold text-slate-700">{networkConfig.quicv0.enable ? "Enabled" : "Disabled"} · Port {networkConfig.quicv0.gatewayPort || "default"}</div></div>}
        {status.secretManager?.address && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Secret manager</div><div className="flex flex-wrap items-center gap-2"><CopyText value={status.secretManager.address} /><span className="text-xs font-semibold text-slate-500">{status.secretManager.tls ? "TLS" : "No TLS"}</span></div></div>}
        {!!status.device?.probes.length && <div><div className="text-[0.66rem] font-bold uppercase tracking-wide text-slate-400">Device probes</div><div className="mt-1 text-sm font-semibold text-slate-700">{status.device.probes.length.toLocaleString()}</div></div>}
      </div></section>}
      <ResourceEdit item={item} specComponent={({ item, onUpdate }) => <Edit item={item as CoreP.ClusterConfig} onUpdate={(next) => onUpdate(next)} />} noPostUpdateNavigation noPostUpdateToast noMetadata onUpdateDone={() => { invalidateKey(["core", "clusterconfig"]); toast.success("ClusterConfig successfully updated"); }} />
    </div>
  );
};
