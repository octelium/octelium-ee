import * as CoreP from "@/apis/corev1/corev1";
import { Condition_Expression as Expression } from "@/apis/enterprisev1/enterprisev1";
import { getOSTypeStr } from "@/pages/core/Device/Main";
import {
  getResourceRef,
  printResourceNameWithDisplay,
  printServiceMode,
} from "@/utils/pb";
import { Select, Switch, TextInput } from "@mantine/core";
import { match } from "ts-pattern";
import SelectResource from "../ResourceLayout/SelectResource";
import TimeAgo from "../TimeAgo";
import TimestampPicker from "../TimestampPicker";

import { ObjectReference } from "@/apis/metav1/metav1";
import { useResourceFromRef } from "../ResourceLayout/utils";
import SelectCountry from "../SelectCountry";

const ResourceValue = (props: { itemRef: ObjectReference }) => {
  const { data } = useResourceFromRef(props.itemRef);
  return <div>{data && `${printResourceNameWithDisplay(data)}`}</div>;
};

export const itemList = [
  {
    type: `user`,
    title: <>User</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `user` && item.type.user.userRef && (
              <ResourceValue itemRef={item.type.user.userRef} />
            )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="User"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "user",
                      user: {
                        userRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `session`,
    title: <>Session</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `session` &&
              item.type.session.sessionRef && (
                <ResourceValue itemRef={item.type.session.sessionRef} />
              )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="Session"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "session",
                      session: {
                        sessionRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `device`,
    title: <>Device</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `device` && item.type.device.deviceRef && (
              <ResourceValue itemRef={item.type.device.deviceRef} />
            )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="Device"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "device",
                      device: {
                        deviceRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `service`,
    title: <>Service</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `service` &&
              item.type.service.serviceRef && (
                <ResourceValue itemRef={item.type.service.serviceRef} />
              )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="Service"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "service",
                      service: {
                        serviceRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `group`,
    title: <>User belongs to Group</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `group` && item.type.group.groupRef && (
              <ResourceValue itemRef={item.type.group.groupRef} />
            )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="Group"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "group",
                      group: {
                        groupRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `namespace`,
    title: <>Namespace</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `namespace` &&
              item.type.namespace.namespaceRef && (
                <ResourceValue itemRef={item.type.namespace.namespaceRef} />
              )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="Namespace"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "namespace",
                      namespace: {
                        namespaceRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationCredential`,
    title: <>Session's Authentication Credential</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `sessionAuthenticationCredential` &&
              item.type.sessionAuthenticationCredential.credentialRef && (
                <ResourceValue
                  itemRef={
                    item.type.sessionAuthenticationCredential.credentialRef
                  }
                />
              )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="Credential"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "sessionAuthenticationCredential",
                      sessionAuthenticationCredential: {
                        credentialRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationIdentityProvider`,
    title: <>Session's Authentication IdentityProvider</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        return (
          <div>
            {item.type.oneofKind === `sessionAuthenticationIdentityProvider` &&
              item.type.sessionAuthenticationIdentityProvider
                .identityProviderRef && (
                <ResourceValue
                  itemRef={
                    item.type.sessionAuthenticationIdentityProvider
                      .identityProviderRef
                  }
                />
              )}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        return (
          <div>
            <SelectResource
              api="core"
              kind="IdentityProvider"
              onChange={(v) => {
                if (!v) {
                  props.onUpdate();
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: "sessionAuthenticationIdentityProvider",
                      sessionAuthenticationIdentityProvider: {
                        identityProviderRef: getResourceRef(v),
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `userType`,
    title: <>User Type</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `userType`) {
          return <></>;
        }

        return (
          <div>
            {match(item.type.userType.type)
              .with(CoreP.User_Spec_Type.HUMAN, () => "Human")
              .with(CoreP.User_Spec_Type.WORKLOAD, () => "Workload")
              .otherwise(() => "")}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
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
              onChange={(v) => {
                if (!v) {
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `userType`,
                      userType: {
                        type: CoreP.User_Spec_Type[v as "HUMAN"],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `deviceOSType`,
    title: <>Device OS Type</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `deviceOSType`) {
          return <></>;
        }

        return <div>{getOSTypeStr(item.type.deviceOSType.osType)}</div>;
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
              data={[
                {
                  label: getOSTypeStr(CoreP.Device_Status_OSType.LINUX),
                  value:
                    CoreP.Device_Status_OSType[
                      CoreP.Device_Status_OSType.LINUX
                    ],
                },
                {
                  label: getOSTypeStr(CoreP.Device_Status_OSType.WINDOWS),
                  value:
                    CoreP.Device_Status_OSType[
                      CoreP.Device_Status_OSType.WINDOWS
                    ],
                },
                {
                  label: getOSTypeStr(CoreP.Device_Status_OSType.MAC),
                  value:
                    CoreP.Device_Status_OSType[CoreP.Device_Status_OSType.MAC],
                },
                {
                  label: getOSTypeStr(CoreP.Device_Status_OSType.ANDROID),
                  value:
                    CoreP.Device_Status_OSType[
                      CoreP.Device_Status_OSType.ANDROID
                    ],
                },
              ]}
              onChange={(v) => {
                if (!v) {
                  return;
                }

                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `deviceOSType`,
                      deviceOSType: {
                        osType: CoreP.Device_Status_OSType[v as "LINUX"],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `serviceMode`,
    title: <>Service Mode</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `serviceMode`) {
          return <></>;
        }

        return <div>{printServiceMode(item.type.serviceMode.mode)}</div>;
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
              required
              data={[
                {
                  label: "HTTP",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.HTTP],
                },
                {
                  label: "TCP",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.TCP],
                },
                {
                  label: "SSH",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.SSH],
                },
                {
                  label: "Web App",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.WEB],
                },
                {
                  label: "Kubernetes",
                  value:
                    CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.KUBERNETES],
                },
                {
                  label: "PostgreSQL",
                  value:
                    CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.POSTGRES],
                },
                {
                  label: "MySQL",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.MYSQL],
                },
                {
                  label: "UDP",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.UDP],
                },
                {
                  label: "gRPC",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.GRPC],
                },
                {
                  label: "DNS",
                  value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.DNS],
                },
              ]}
              // value={CoreP.Service_Spec_Mode[req.spec!.mode]}
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `serviceMode`,
                      serviceMode: {
                        mode: CoreP.Service_Spec_Mode[v as "HTTP"],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionType`,
    title: <>Session Type</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `sessionType`) {
          return <></>;
        }

        return (
          <div>
            {match(item.type.sessionType.type)
              .with(CoreP.Session_Status_Type.CLIENT, () => "Client")
              .with(CoreP.Session_Status_Type.CLIENTLESS, () => "Clientless")
              .otherwise(() => "")}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
              required
              data={[
                {
                  label: "Client",
                  value:
                    CoreP.Session_Status_Type[CoreP.Session_Status_Type.CLIENT],
                },
                {
                  label: "Clientless",
                  value:
                    CoreP.Session_Status_Type[
                      CoreP.Session_Status_Type.CLIENTLESS
                    ],
                },
              ]}
              // value={CoreP.Service_Spec_Mode[req.spec!.mode]}
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionType`,
                      sessionType: {
                        type: CoreP.Session_Status_Type[v as "CLIENT"],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationAAL`,
    title: <>Session Authentication Assurance Level (AAL)</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `sessionAuthenticationAAL`) {
          return <></>;
        }

        return (
          <div>
            {match(item.type.sessionAuthenticationAAL.aal)
              .with(
                CoreP.Session_Status_Authentication_Info_AAL.AAL1,
                () => "AAL1",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_AAL.AAL2,
                () => "AAL2",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_AAL.AAL3,
                () => "AAL3",
              )
              .otherwise(() => "")}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
              required
              data={[
                {
                  label: "AAL1",
                  value:
                    CoreP.Session_Status_Authentication_Info_AAL[
                      CoreP.Session_Status_Authentication_Info_AAL.AAL1
                    ],
                },
                {
                  label: "AAL2",
                  value:
                    CoreP.Session_Status_Authentication_Info_AAL[
                      CoreP.Session_Status_Authentication_Info_AAL.AAL2
                    ],
                },
                {
                  label: "AAL3",
                  value:
                    CoreP.Session_Status_Authentication_Info_AAL[
                      CoreP.Session_Status_Authentication_Info_AAL.AAL3
                    ],
                },
              ]}
              // value={CoreP.Service_Spec_Mode[req.spec!.mode]}
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationAAL`,
                      sessionAuthenticationAAL: {
                        aal: CoreP.Session_Status_Authentication_Info_AAL[
                          v as "AAL1"
                        ],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationType`,
    title: <>Session Authentication Type</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `sessionAuthenticationType`) {
          return <></>;
        }

        return (
          <div>
            {match(item.type.sessionAuthenticationType.type)
              .with(
                CoreP.Session_Status_Authentication_Info_Type.AUTHENTICATOR,
                () => "Authenticator",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.CREDENTIAL,
                () => "Credential",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
                () => "IdentityProvider",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.REFRESH_TOKEN,
                () => "Refresh Token",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.INTERNAL,
                () => "Internal",
              )
              .otherwise(() => "")}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
              required
              data={[
                {
                  label: "Authenticator",
                  value:
                    CoreP.Session_Status_Authentication_Info_Type[
                      CoreP.Session_Status_Authentication_Info_Type
                        .AUTHENTICATOR
                    ],
                },
                {
                  label: "Credential",
                  value:
                    CoreP.Session_Status_Authentication_Info_Type[
                      CoreP.Session_Status_Authentication_Info_Type.CREDENTIAL
                    ],
                },
                {
                  label: "IdentityProvider",
                  value:
                    CoreP.Session_Status_Authentication_Info_Type[
                      CoreP.Session_Status_Authentication_Info_Type
                        .IDENTITY_PROVIDER
                    ],
                },
                {
                  label: "Internal",
                  value:
                    CoreP.Session_Status_Authentication_Info_Type[
                      CoreP.Session_Status_Authentication_Info_Type.INTERNAL
                    ],
                },
                {
                  label: "Refresh Token",
                  value:
                    CoreP.Session_Status_Authentication_Info_Type[
                      CoreP.Session_Status_Authentication_Info_Type
                        .REFRESH_TOKEN
                    ],
                },
              ]}
              // value={CoreP.Service_Spec_Mode[req.spec!.mode]}
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationType`,
                      sessionAuthenticationType: {
                        type: CoreP.Session_Status_Authentication_Info_Type[
                          v as "CREDENTIAL"
                        ],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationCredentialType`,
    title: <>Session's Parent Credential Type</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `sessionAuthenticationCredentialType`) {
          return <></>;
        }

        return (
          <div>
            {match(item.type.sessionAuthenticationCredentialType.type)
              .with(
                CoreP.Credential_Spec_Type.ACCESS_TOKEN,
                () => "Access Token",
              )
              .with(
                CoreP.Credential_Spec_Type.AUTH_TOKEN,
                () => "Authentication Token",
              )
              .with(
                CoreP.Credential_Spec_Type.OAUTH2,
                () => "OAuth2 Client Credentials",
              )

              .otherwise(() => "")}
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Select
              required
              data={[
                {
                  label: "Access Token",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.ACCESS_TOKEN
                    ],
                },
                {
                  label: "Authentication Token",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.AUTH_TOKEN
                    ],
                },
                {
                  label: "OAuth2 Client Credentials",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.OAUTH2
                    ],
                },
              ]}
              // value={CoreP.Service_Spec_Mode[req.spec!.mode]}
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationCredentialType`,
                      sessionAuthenticationCredentialType: {
                        type: CoreP.Credential_Spec_Type[v as "OAUTH2"],
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationGeoipCountryCode`,
    title: <>Session Country</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `sessionAuthenticationGeoipCountryCode`) {
          return <></>;
        }

        return (
          <div>{item.type.sessionAuthenticationGeoipCountryCode.code}</div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <SelectCountry
              val={
                item?.type.oneofKind === `sessionAuthenticationGeoipCountryCode`
                  ? item.type.sessionAuthenticationGeoipCountryCode.code
                  : undefined
              }
              onUpdate={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationGeoipCountryCode`,
                      sessionAuthenticationGeoipCountryCode: {
                        code: v ?? "",
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `timeBefore`,
    title: <>Time Before</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `timeBefore`) {
          return <></>;
        }

        return (
          <div>
            <TimeAgo rfc3339={item.type.timeBefore.timestamp} />
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <TimestampPicker
              value={
                item?.type.oneofKind === `timeBefore`
                  ? item.type.timeBefore.timestamp
                  : undefined
              }
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `timeBefore`,
                      timeBefore: {
                        timestamp: v,
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `timeAfter`,
    title: <>Time After</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `timeAfter`) {
          return <></>;
        }

        return (
          <div>
            <TimeAgo rfc3339={item.type.timeAfter.timestamp} />
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <TimestampPicker
              isFuture
              value={
                item?.type.oneofKind === `timeAfter`
                  ? item.type.timeAfter.timestamp
                  : undefined
              }
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `timeAfter`,
                      timeAfter: {
                        timestamp: v,
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationCredAuthenticatorFIDOHardware`,
    title: <>Hardware-based FIDO</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (
          item.type.oneofKind !==
          `sessionAuthenticationCredAuthenticatorFIDOHardware`
        ) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind ===
                `sessionAuthenticationCredAuthenticatorFIDOHardware`
                  ? props.item.type
                      .sessionAuthenticationCredAuthenticatorFIDOHardware
                      .isHardware
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationCredAuthenticatorFIDOHardware`,
                      sessionAuthenticationCredAuthenticatorFIDOHardware: {
                        isHardware: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationCredAuthenticatorFIDOAttestationVerified`,
    title: <>FIDO Verified Attestation</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (
          item.type.oneofKind !==
          `sessionAuthenticationCredAuthenticatorFIDOAttestationVerified`
        ) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind ===
                `sessionAuthenticationCredAuthenticatorFIDOAttestationVerified`
                  ? props.item.type
                      .sessionAuthenticationCredAuthenticatorFIDOAttestationVerified
                      .isAttestationVerified
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationCredAuthenticatorFIDOAttestationVerified`,
                      sessionAuthenticationCredAuthenticatorFIDOAttestationVerified:
                        {
                          isAttestationVerified: event.currentTarget.checked,
                        },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationCredAuthenticatorFIDOUserVerified`,
    title: <>FIDO User Verified</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (
          item.type.oneofKind !==
          `sessionAuthenticationCredAuthenticatorFIDOUserVerified`
        ) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind ===
                `sessionAuthenticationCredAuthenticatorFIDOUserVerified`
                  ? props.item.type
                      .sessionAuthenticationCredAuthenticatorFIDOUserVerified
                      .isUserVerified
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationCredAuthenticatorFIDOUserVerified`,
                      sessionAuthenticationCredAuthenticatorFIDOUserVerified: {
                        isUserVerified: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `sessionAuthenticationCredAuthenticatorFIDOUserPresent`,
    title: <>FIDO User Present</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (
          item.type.oneofKind !==
          `sessionAuthenticationCredAuthenticatorFIDOUserPresent`
        ) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind ===
                `sessionAuthenticationCredAuthenticatorFIDOUserPresent`
                  ? props.item.type
                      .sessionAuthenticationCredAuthenticatorFIDOUserPresent
                      .isUserPresent
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `sessionAuthenticationCredAuthenticatorFIDOUserPresent`,
                      sessionAuthenticationCredAuthenticatorFIDOUserPresent: {
                        isUserPresent: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `requestHTTPPathExact`,
    title: <>Request HTTP Exact Path</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `requestHTTPPathExact`) {
          return <></>;
        }

        return (
          <div>
            <>{item.type.requestHTTPPathExact.value}</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <TextInput
              required
              label="Exact path"
              placeholder="/api/v1"
              value={
                props.item?.type.oneofKind === `requestHTTPPathExact`
                  ? props.item.type.requestHTTPPathExact.value
                  : undefined
              }
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `requestHTTPPathExact`,
                      requestHTTPPathExact: {
                        value: v.target.value,
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `requestHTTPPathPrefix`,
    title: <>Request HTTP Path Prefix</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `requestHTTPPathPrefix`) {
          return <></>;
        }

        return (
          <div>
            <>{item.type.requestHTTPPathPrefix.value}</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <TextInput
              required
              label="Path Prefix"
              placeholder="/api/v1"
              value={
                props.item?.type.oneofKind === `requestHTTPPathPrefix`
                  ? props.item.type.requestHTTPPathPrefix.value
                  : undefined
              }
              onChange={(v) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `requestHTTPPathPrefix`,
                      requestHTTPPathPrefix: {
                        value: v.target.value,
                      },
                    },
                  }),
                );
              }}
            />
          </div>
        );
      },
    },
  },

  {
    type: `apiServer`,
    title: <>Request to API Server</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `apiServer`) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind === `apiServer`
                  ? props.item.type.apiServer.isAPIServer
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `apiServer`,
                      apiServer: {
                        isAPIServer: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `apiServerCore`,
    title: <>Request to API Server Core API</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `apiServerCore`) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind === `apiServerCore`
                  ? props.item.type.apiServerCore.isAPIServerCore
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `apiServerCore`,
                      apiServerCore: {
                        isAPIServerCore: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `apiServerUser`,
    title: <>Request to API Server User API</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `apiServerUser`) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind === `apiServerUser`
                  ? props.item.type.apiServerUser.isAPIServerUser
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `apiServerUser`,
                      apiServerUser: {
                        isAPIServerUser: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `apiServerEnterprise`,
    title: <>Request to API Server Enterprise API</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `apiServerEnterprise`) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind === `apiServerEnterprise`
                  ? props.item.type.apiServerEnterprise.isAPIServerEnterprise
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `apiServerEnterprise`,
                      apiServerEnterprise: {
                        isAPIServerEnterprise: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },

  {
    type: `apiServerCordium`,
    title: <>Request to API Server Coredium API</>,
    components: {
      Value: (props: { item: Expression }) => {
        const { item } = props;

        if (item.type.oneofKind !== `apiServerCordium`) {
          return <></>;
        }

        return (
          <div>
            <>Enabled</>
          </div>
        );
      },
      Edit: (props: {
        item?: Expression;
        onUpdate: (item?: Expression) => void;
      }) => {
        const { item } = props;

        return (
          <div>
            <Switch
              checked={
                props.item?.type.oneofKind === `apiServerCordium`
                  ? props.item.type.apiServerCordium.isAPIServerCordium
                  : undefined
              }
              onChange={(event) => {
                props.onUpdate(
                  Expression.create({
                    type: {
                      oneofKind: `apiServerCordium`,
                      apiServerCordium: {
                        isAPIServerCordium: event.currentTarget.checked,
                      },
                    },
                  }),
                );
              }}
              size="lg"
            />
          </div>
        );
      },
    },
  },
];

export default itemList;
