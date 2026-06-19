import * as CoreP from "@/apis/corev1/corev1";
import Cond from "@/components/Condition";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectSecret from "@/components/ResourceLayout/SelectSecret";
import { getClientCore } from "@/utils/client";
import { strToNum } from "@/utils/convert";
import { invalidateKey } from "@/utils/pb";
import {
  Group,
  NumberInput,
  Select,
  Switch,
  Tabs,
  TagsInput,
  TextInput,
} from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { toast } from "sonner";
import { match } from "ts-pattern";

const Edit = (props: {
  item: CoreP.ClusterConfig;
  onUpdate: (item: CoreP.ClusterConfig) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(CoreP.ClusterConfig.clone(item));
  const updateReq = () => {
    setReq(CoreP.ClusterConfig.clone(req));
    onUpdate(req);
  };

  return (
    <div className="w-full">
      <div className="w-full my-12">
        <div className="w-full">
          <EditItem
            title="DNS"
            description="Set the Cluster's DNS-related Configuration"
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
                  description="Set the fallback zone DNS servers for the Cluster's private DNS server Service"
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
                        description="Set one or more Fallback DNS servers. Both raw DNS and DNS over TLS servers are supported"
                        value={req.spec!.dns!.fallbackZone!.servers}
                        onChange={(v) => {
                          req.spec!.dns!.fallbackZone!.servers = v;
                          updateReq();
                        }}
                      />

                      <DurationPicker
                        value={req.spec!.dns!.fallbackZone.cacheDuration}
                        title="Cache Duration"
                        description="Set the cache duration of the DNS server"
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
            description="Set the Cluster's Gateway-related Configuration"
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
                  description="Set the duration for the Gateways to periodically rotate their keys"
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
            description="Set the Global Authenticator-related Configuration"
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
                    description="Enabled the ability for Users to login with Passkeys"
                    checked={req.spec!.authenticator!.enablePasskeyLogin}
                    onChange={(v) => {
                      req.spec!.authenticator!.enablePasskeyLogin =
                        v.target.checked;
                      updateReq();
                    }}
                  />

                  <Select
                    label="Default State"
                    description="Set the default Authenticator state to ACTIVE, PENDING or REJECTED"
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
                  description="Set the Global FIDO-related Configuration"
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
                          description="Set attestation conveyance preference to DIRECT, INDIRECT, NONE or ENTERPRISE"
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
                      (rule: any, ruleIdx: number) => (
                        <EditItem
                          key={`${ruleIdx}`}
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
                      (rule: any, ruleIdx: number) => (
                        <EditItem
                          key={`${ruleIdx}`}
                          obj={
                            req.spec!.authenticator!
                              .authenticationEnforcementRules[ruleIdx]
                          }
                          onUnset={() => {
                            req.spec!.authenticator!.authenticationEnforcementRules.splice(
                              ruleIdx,
                              1,
                            );
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
                      (rule: any, ruleIdx: number) => (
                        <EditItem
                          key={`${ruleIdx}`}
                          obj={
                            req.spec!.authenticator!
                              .registrationEnforcementRules[ruleIdx]
                          }
                          onUnset={() => {
                            req.spec!.authenticator!.registrationEnforcementRules.splice(
                              ruleIdx,
                              1,
                            );
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
            description="Set the Global Authorization-related Configuration"
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
            description="Set the Cluster's Session-related Configuration"
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
                  description="Set Human Session-related Configuration"
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
                          onChange={(v) => {
                            req.spec!.session!.human!.accessTokenDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.human!.refreshTokenDuration}
                          title="Refresh Token Duration"
                          onChange={(v) => {
                            req.spec!.session!.human!.refreshTokenDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.human!.clientDuration}
                          title="Client-base Session Duration"
                          onChange={(v) => {
                            req.spec!.session!.human!.clientDuration = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.human!.clientlessDuration}
                          title="Clientless Session Duration"
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
                  description="Set Workload Session-related Configuration"
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
                          onChange={(v) => {
                            req.spec!.session!.workload!.refreshTokenDuration =
                              v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={req.spec!.session!.workload!.clientDuration}
                          title="Client-base Session Duration"
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
            description="Set the Cluster's Device-related Configuration"
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
                  description="Set Human Device-related Configuration"
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
                  description="Set Workload Device-related Configuration"
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
            description="Set the Cluster's Ingress-related Configuration"
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
              <Group grow>
                <Switch
                  label="Use X-Forwarded-For Header"
                  description="Trust and enable the use of the X-Forwarded-For header to obtain the downstream IP address"
                  checked={req.spec!.ingress!.useForwardedForHeader}
                  onChange={(v) => {
                    req.spec!.ingress!.useForwardedForHeader = v.target.checked;

                    updateReq();
                  }}
                />

                <NumberInput
                  label="X-Forwarded-For trusted Hops"
                  description="Set the number of trusted hops between Octelium ingress an the downstream"
                  value={req.spec!.ingress!.xffNumTrustedHops}
                  min={0}
                  max={100}
                  onChange={(v) => {
                    req.spec!.ingress!.xffNumTrustedHops = strToNum(v);
                    updateReq();
                  }}
                />
              </Group>
            )}
          </EditItem>

          <EditItem
            title="Authentication"
            description="Set the authentication-related Configuration"
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
                  description="Set geolocation-related Configuration"
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
                                <Select
                                  label="MMDB Source"
                                  description="Set the MMDB source"
                                  data={[
                                    {
                                      label: "Upstream URL",
                                      value: "upstream",
                                    },
                                    {
                                      label: "From Config",
                                      value: "fromConfig",
                                    },
                                  ]}
                                  value={mmdb.mmdb.type.oneofKind}
                                  onChange={(v) => {
                                    mmdb.mmdb.type = match(v)
                                      .with("fromConfig", () => ({
                                        oneofKind: "fromConfig" as const,
                                        fromConfig: "",
                                      }))
                                      .otherwise(() => ({
                                        oneofKind: "upstream" as const,
                                        upstream:
                                          CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream.create(
                                            { url: "" },
                                          ),
                                      }));
                                    updateReq();
                                  }}
                                />

                                {match(mmdb.mmdb.type)
                                  .when(
                                    (x) => x.oneofKind === `fromConfig`,
                                    (fromConfig) => {
                                      return (
                                        <Group grow>
                                          <TextInput
                                            label="From Config"
                                            placeholder="my-mmdb-config"
                                            description="Set the name of the config containing the MMDB"
                                            value={fromConfig.fromConfig}
                                            onChange={(v) => {
                                              fromConfig.fromConfig =
                                                v.target.value;
                                              updateReq();
                                            }}
                                          />
                                        </Group>
                                      );
                                    },
                                  )
                                  .when(
                                    (x) => x.oneofKind === `upstream`,
                                    (upstream) => {
                                      return (
                                        <div>
                                          <Group grow>
                                            <TextInput
                                              label="URL"
                                              placeholder="https://mmdb.example/country-db-v1.0.0"
                                              description="Set the MMDB URL"
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
                                            description="Set the MMDB upstream authentication"
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
                                                defaultValue={
                                                  upstream.upstream.auth.type
                                                    .oneofKind
                                                }
                                                onChange={(v) => {
                                                  match(v)
                                                    .with("bearer", () => {
                                                      upstream.upstream.auth!.type =
                                                        {
                                                          oneofKind: "bearer",
                                                          bearer:
                                                            CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Bearer.create(
                                                              {
                                                                type: {
                                                                  oneofKind:
                                                                    "fromSecret",
                                                                  fromSecret:
                                                                    "",
                                                                },
                                                              },
                                                            ),
                                                        };
                                                    })
                                                    .with("basic", () => {
                                                      upstream.upstream.auth!.type =
                                                        {
                                                          oneofKind: "basic",
                                                          basic:
                                                            CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Basic.create(
                                                              {
                                                                password: {
                                                                  type: {
                                                                    oneofKind:
                                                                      "fromSecret",
                                                                    fromSecret:
                                                                      "",
                                                                  },
                                                                },
                                                              },
                                                            ),
                                                        };
                                                    })
                                                    .with("custom", () => {
                                                      upstream.upstream.auth!.type =
                                                        {
                                                          oneofKind: "custom",
                                                          custom:
                                                            CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Custom.create(
                                                              {
                                                                value: {
                                                                  type: {
                                                                    oneofKind:
                                                                      "fromSecret",
                                                                    fromSecret:
                                                                      "",
                                                                  },
                                                                },
                                                              },
                                                            ),
                                                        };
                                                    })
                                                    .with("query", () => {
                                                      upstream.upstream.auth!.type =
                                                        {
                                                          oneofKind: "query",
                                                          query:
                                                            CoreP.ClusterConfig_Spec_Authentication_Geolocation_MMDB_Upstream_Auth_Query.create(
                                                              {
                                                                value: {
                                                                  type: {
                                                                    oneofKind:
                                                                      "fromSecret",
                                                                    fromSecret:
                                                                      "",
                                                                  },
                                                                },
                                                              },
                                                            ),
                                                        };
                                                    })
                                                    .otherwise(() => {});
                                                  updateReq();
                                                }}
                                              >
                                                <Tabs.List>
                                                  <Tabs.Tab value="bearer">
                                                    Bearer
                                                  </Tabs.Tab>
                                                  <Tabs.Tab value="basic">
                                                    Basic
                                                  </Tabs.Tab>
                                                  <Tabs.Tab value="custom">
                                                    Custom Header
                                                  </Tabs.Tab>
                                                  <Tabs.Tab value="query">
                                                    Query
                                                  </Tabs.Tab>
                                                </Tabs.List>

                                                <Tabs.Panel value="bearer">
                                                  {match(
                                                    upstream.upstream.auth.type,
                                                  )
                                                    .when(
                                                      (x) =>
                                                        x.oneofKind ===
                                                        "bearer",
                                                      (bearer) => (
                                                        <SelectSecret
                                                          api="core"
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
                                                                  val ?? "";
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
                                                                <SelectSecret
                                                                  api="core"
                                                                  label="Password Secret"
                                                                  description="Select the Secret of the password"
                                                                  defaultValue={
                                                                    x.fromSecret
                                                                  }
                                                                  onChange={(
                                                                    v,
                                                                  ) => {
                                                                    x.fromSecret =
                                                                      v ?? "";
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
                                                                <SelectSecret
                                                                  api="core"
                                                                  label="Header value Secret"
                                                                  description="Select the Secret of the header value"
                                                                  defaultValue={
                                                                    x.fromSecret
                                                                  }
                                                                  onChange={(
                                                                    v,
                                                                  ) => {
                                                                    x.fromSecret =
                                                                      v ?? "";
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
                                                                <SelectSecret
                                                                  api="core"
                                                                  label="Query value Secret"
                                                                  description="Select the Secret of the query value"
                                                                  defaultValue={
                                                                    x.fromSecret
                                                                  }
                                                                  onChange={(
                                                                    v,
                                                                  ) => {
                                                                    x.fromSecret =
                                                                      v ?? "";
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
  const { isSuccess, data } = useQuery({
    queryKey: ["core", "clusterconfig"],
    queryFn: async () => {
      return await getClientCore().getClusterConfig({});
    },
  });

  if (!isSuccess) {
    return <></>;
  }

  if (!data) {
    return <></>;
  }

  return (
    <div>
      {data && data.response && (
        <ResourceEdit
          item={data.response}
          // @ts-ignore
          specComponent={Edit}
          noPostUpdateNavigation
          noPostUpdateToast
          noMetadata
          onUpdateDone={(v) => {
            invalidateKey(["core", "clusterconfig"]);
            toast.success("ClusterConfig successfully updated");
          }}
        />
      )}
    </div>
  );
};
