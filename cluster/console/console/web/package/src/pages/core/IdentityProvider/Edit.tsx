import * as CoreP from "@/apis/corev1/corev1";

import { IdentityProvider_Spec_OIDCIdentityToken } from "@/apis/corev1/corev1";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import TextAreaCustom from "@/components/TextAreaCustom";
import { getDomain } from "@/utils";
import { cloneResource } from "@/utils/pb";
import {
  Group,
  SegmentedControl,
  Switch,
  TagsInput,
  TextInput,
} from "@mantine/core";
import React from "react";
import { match } from "ts-pattern";

type SegmentedTabsContextValue = {
  value: string | null;
  onChange: (value: string) => void;
};

const SegmentedTabsContext = React.createContext<SegmentedTabsContextValue>({
  value: null,
  onChange: () => {},
});

const SegmentedTabsRoot = (props: {
  value?: string | null;
  defaultValue?: string | null;
  onChange?: (value: string | null) => void;
  className?: string;
  children: React.ReactNode;
}) => {
  const [internalValue, setInternalValue] = React.useState(
    props.defaultValue ?? null,
  );
  const value = props.value ?? internalValue;
  const onChange = (next: string) => {
    if (props.value === undefined) setInternalValue(next);
    props.onChange?.(next);
  };

  return (
    <SegmentedTabsContext.Provider value={{ value, onChange }}>
      <div className={props.className}>{props.children}</div>
    </SegmentedTabsContext.Provider>
  );
};

const SegmentedTabsTab = (_props: {
  value: string;
  children: React.ReactNode;
}) => null;

const SegmentedTabsList = (props: {
  className?: string;
  children: React.ReactNode;
}) => {
  const { value, onChange } = React.useContext(SegmentedTabsContext);
  const data = React.Children.toArray(props.children)
    .filter(React.isValidElement)
    .map((child) => {
      const tab = child as React.ReactElement<{
        value: string;
        children: React.ReactNode;
      }>;
      return { value: tab.props.value, label: tab.props.children };
    });

  return (
    <div className={`w-full overflow-x-auto ${props.className ?? ""}`}>
      <SegmentedControl
        size="sm"
        value={value ?? data[0]?.value}
        onChange={onChange}
        data={data}
        fullWidth
        className="w-full"
      />
    </div>
  );
};

const SegmentedTabsPanel = (props: {
  value: string;
  children: React.ReactNode;
}) => {
  const { value } = React.useContext(SegmentedTabsContext);
  return value === props.value ? <>{props.children}</> : null;
};

const Tabs = Object.assign(SegmentedTabsRoot, {
  List: SegmentedTabsList,
  Tab: SegmentedTabsTab,
  Panel: SegmentedTabsPanel,
});

const Edit = (props: {
  item: CoreP.IdentityProvider;
  onUpdate: (item: CoreP.IdentityProvider) => void;
}) => {
  let [req, setReq] = React.useState(props.item);
  let [init, _] = React.useState(cloneResource(req) as CoreP.IdentityProvider);
  const updateReq = () => {
    setReq(CoreP.IdentityProvider.clone(req));
    props.onUpdate(req);
  };

  return (
    <div>
      <Group grow>
        <TextInput
          label="Display Name"
          description="Public name shown to Users on the sign-in screen."
          placeholder="Log in with MyIDP"
          value={req.spec!.displayName}
          onChange={(v) => {
            req.spec!.displayName = v.target.value;
            updateReq();
          }}
        />

        <Switch
          label="Disable User Email as identity"
          description="Require an explicit User identity instead of matching the email returned by this provider."
          checked={req.spec!.disableEmailAsIdentity}
          onChange={(v) => {
            req.spec!.disableEmailAsIdentity = v.target.checked;
            updateReq();
          }}
        />
        <Switch
          label="Disabled"
          description="Disable this IdentityProvider so it cannot be used for authentication."
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>

      <Tabs
        className="mt-8"
        defaultValue={req.spec!.type.oneofKind}
        onChange={(v) => {
          match(v)
            .with("github", () => {
              match(init.spec!.type.oneofKind)
                .with("github", () => {
                  req.spec!.type = init!.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "github",
                    github: {
                      clientID: "",
                      clientSecret: {
                        type: { oneofKind: "fromSecret", fromSecret: "" },
                      },
                    },
                  };
                });
              updateReq();
            })
            .with("oidc", () => {
              match(init.spec!.type.oneofKind)
                .with("oidc", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "oidc",
                    oidc: CoreP.IdentityProvider_Spec_OIDC.create({
                      clientID: "",
                      clientSecret: {
                        type: { oneofKind: "fromSecret", fromSecret: "" },
                      },
                    }),
                  };
                });
              updateReq();
            })
            .with("saml", () => {
              match(init.spec!.type.oneofKind)
                .with("saml", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "saml",
                    saml: {
                      metadataType: {
                        oneofKind: "metadataURL",
                        metadataURL: "",
                      },
                    } as CoreP.IdentityProvider_Spec_SAML,
                  };
                });
              updateReq();
            })
            .with("oidcIdentityToken", () => {
              match(init.spec!.type.oneofKind)
                .with("oidcIdentityToken", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "oidcIdentityToken",
                    oidcIdentityToken:
                      {} as CoreP.IdentityProvider_Spec_OIDCIdentityToken,
                  };
                });
              updateReq();
            });
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="github">GitHub</Tabs.Tab>
          <Tabs.Tab value="oidc">OpenID Connect</Tabs.Tab>
          <Tabs.Tab value="saml">SAML 2.0</Tabs.Tab>
          <Tabs.Tab value="oidcIdentityToken">OIDC Identity Token</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="github">
          {req.spec!.type.oneofKind === "github" && (
            <Group grow>
              <TextInput
                required
                label="Client ID"
                description="GitHub OAuth2 application client ID."
                placeholder="abcdefgh"
                value={req.spec!.type.github.clientID}
                onChange={(v) => {
                  let f = req.spec!.type as {
                    oneofKind: "github";
                    github: CoreP.IdentityProvider_Spec_Github;
                  };
                  f.github.clientID = v.target.value;
                  updateReq();
                }}
              />
              <SelectResource
                api="core"
                kind="Secret"
                label="Client Secret"
                description="Secret containing the GitHub OAuth2 client secret."
                defaultValue={
                  req.spec!.type.github.clientSecret?.type.oneofKind ===
                  "fromSecret"
                    ? req.spec!.type.github.clientSecret.type.fromSecret
                    : undefined
                }
                onChange={(v) => {
                  let f = req.spec!.type as {
                    oneofKind: "github";
                    github: CoreP.IdentityProvider_Spec_Github;
                  };
                  let g = f.github.clientSecret!.type as {
                    oneofKind: "fromSecret";
                    fromSecret: string;
                  };
                  g.fromSecret = v?.metadata?.name ?? "";
                  updateReq();
                }}
              />
            </Group>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="oidc">
          {req.spec!.type.oneofKind === "oidc" && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Client ID"
                  description="OIDC OAuth2 application client ID."
                  placeholder="abcdefgh"
                  value={req.spec!.type.oidc.clientID}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    f.oidc.clientID = v.target.value;
                    updateReq();
                  }}
                />
                <SelectResource
                  api="core"
                  kind="Secret"
                  label="Client Secret"
                  description="Secret containing the OIDC client secret."
                  defaultValue={
                    req.spec!.type.oidc.clientSecret?.type.oneofKind ===
                    "fromSecret"
                      ? req.spec!.type.oidc.clientSecret.type.fromSecret
                      : undefined
                  }
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    let g = f.oidc.clientSecret!.type as {
                      oneofKind: "fromSecret";
                      fromSecret: string;
                    };
                    g.fromSecret = v?.metadata?.name ?? "";
                    updateReq();
                  }}
                />
              </Group>
              <Group grow>
                <TextInput
                  required
                  label="Issuer URL"
                  description="Issuer URL used to discover the OIDC configuration and signing keys."
                  placeholder="https://accounts.google.com"
                  value={req.spec!.type.oidc.issuerURL}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    f.oidc.issuerURL = v.target.value;
                    updateReq();
                  }}
                />
                <TagsInput
                  label="Scopes"
                  placeholder="email, scope-1, scope-2"
                  description="Additional scopes beyond openid; defaults to profile and email when omitted."
                  value={req.spec!.type.oidc.scopes}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    f.oidc.scopes = v;
                    updateReq();
                  }}
                />
              </Group>
              <Group grow>
                <TextInput
                  label="Identifier Claim"
                  description="Claim whose value identifies the User; email is used by default."
                  placeholder="email"
                  value={req.spec!.type.oidc.identifierClaim}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    f.oidc.identifierClaim = v.target.value;
                    updateReq();
                  }}
                />
                <Switch
                  label="Check email verified"
                  description="Accept only tokens whose email_verified claim is true."
                  checked={req.spec!.type.oidc.checkEmailVerified}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    f.oidc.checkEmailVerified = v.target.checked;
                    updateReq();
                  }}
                />
                <Switch
                  label="Use UserInfo endpoint"
                  description="Fetch additional claims from the provider's UserInfo endpoint."
                  checked={req.spec!.type.oidc.useUserInfoEndpoint}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidc";
                      oidc: CoreP.IdentityProvider_Spec_OIDC;
                    };
                    f.oidc.useUserInfoEndpoint = v.target.checked;
                    updateReq();
                  }}
                />
              </Group>
            </>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="saml">
          {req.spec!.type.oneofKind === "saml" && (
            <>
              <Group grow>
                <TextInput
                  label="Entity ID"
                  description="SAML entity ID; the Cluster domain is used when omitted."
                  placeholder="https://accounts.google.com"
                  value={req.spec!.type.saml.entityID}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "saml";
                      saml: CoreP.IdentityProvider_Spec_SAML;
                    };
                    f.saml.entityID = v.target.value;
                    updateReq();
                  }}
                />
                <TextInput
                  label="Identifier Attribute"
                  description="SAML attribute used as the User identifier; email address is used by default."
                  placeholder="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
                  value={req.spec!.type.saml.identifierAttribute}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "saml";
                      saml: CoreP.IdentityProvider_Spec_SAML;
                    };
                    f.saml.identifierAttribute = v.target.value;
                    updateReq();
                  }}
                />
                <Switch
                  label="Force Authn"
                  description="Force re-authentication even when the User has an active SSO session."
                  checked={req.spec!.type.saml.forceAuthn}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "saml";
                      saml: CoreP.IdentityProvider_Spec_SAML;
                    };
                    f.saml.forceAuthn = v.target.checked;
                    updateReq();
                  }}
                />
              </Group>

              <Tabs
                defaultValue={req.spec!.type.saml.metadataType.oneofKind}
                onChange={(v) => {
                  match(v)
                    .with("metadataURL", () => {
                      match(init.spec?.type)
                        .when(
                          (x) => x?.oneofKind === "saml",
                          (x) => {
                            match(x.saml.metadataType)
                              .when(
                                (x) => x.oneofKind === "metadataURL",
                                (x) => {
                                  match(req.spec!.type).when(
                                    (t) => t.oneofKind === "saml",
                                    (t) => {
                                      t.saml.metadataType = {
                                        oneofKind: "metadataURL",
                                        metadataURL: x.metadataURL,
                                      };
                                      updateReq();
                                    },
                                  );
                                },
                              )
                              .otherwise(() => {
                                match(req.spec!.type).when(
                                  (t) => t.oneofKind === "saml",
                                  (t) => {
                                    t.saml.metadataType = {
                                      oneofKind: "metadataURL",
                                      metadataURL: "",
                                    };
                                    updateReq();
                                  },
                                );
                              });
                          },
                        )
                        .otherwise(() => {
                          match(req.spec!.type).when(
                            (t) => t.oneofKind === "saml",
                            (t) => {
                              t.saml.metadataType = {
                                oneofKind: "metadataURL",
                                metadataURL: "",
                              };
                              updateReq();
                            },
                          );
                        });
                    })
                    .with("metadata", () => {
                      match(init.spec?.type)
                        .when(
                          (x) => x?.oneofKind === "saml",
                          (x) => {
                            match(x.saml.metadataType)
                              .when(
                                (x) => x.oneofKind === "metadata",
                                (x) => {
                                  match(req.spec!.type).when(
                                    (t) => t.oneofKind === "saml",
                                    (t) => {
                                      t.saml.metadataType = {
                                        oneofKind: "metadata",
                                        metadata: x.metadata,
                                      };
                                      updateReq();
                                    },
                                  );
                                },
                              )
                              .otherwise(() => {
                                match(req.spec!.type).when(
                                  (t) => t.oneofKind === "saml",
                                  (t) => {
                                    t.saml.metadataType = {
                                      oneofKind: "metadata",
                                      metadata: "",
                                    };
                                    updateReq();
                                  },
                                );
                              });
                          },
                        )
                        .otherwise(() => {
                          match(req.spec!.type).when(
                            (t) => t.oneofKind === "saml",
                            (t) => {
                              t.saml.metadataType = {
                                oneofKind: "metadata",
                                metadata: "",
                              };
                              updateReq();
                            },
                          );
                        });
                    });
                }}
              >
                <Tabs.List>
                  <Tabs.Tab value="metadataURL">Metadata URL</Tabs.Tab>
                  <Tabs.Tab value="metadata">Metadata Content</Tabs.Tab>
                </Tabs.List>

                <Tabs.Panel value="metadataURL">
                  {req.spec!.type.saml.metadataType.oneofKind ===
                    "metadataURL" && (
                    <TextInput
                      required
                      label="Metadata URL"
                      description="The provider's SAML 2.0 metadata URL"
                      placeholder="https://saml.example.com/metadata.xml"
                      value={req.spec!.type.saml.metadataType.metadataURL}
                      onChange={(v) => {
                        let f = req.spec!.type as {
                          oneofKind: "saml";
                          saml: CoreP.IdentityProvider_Spec_SAML;
                        };
                        let g = f.saml.metadataType as {
                          oneofKind: "metadataURL";
                          metadataURL: string;
                        };
                        g.metadataURL = v.target.value;
                        updateReq();
                      }}
                    />
                  )}
                </Tabs.Panel>

                <Tabs.Panel value="metadata">
                  {req.spec!.type.saml.metadataType.oneofKind ===
                    "metadata" && (
                    <TextAreaCustom
                      label="Metadata"
                      required
                      description="Inline SAML 2.0 metadata XML used to configure the provider."
                      placeholder={`<?xml version="1.0" encoding="UTF-8"?><md:EntityDescriptor ...>`}
                      rows={8}
                      value={req.spec!.type.saml.metadataType.metadata}
                      onChange={(v) => {
                        let f = req.spec!.type as {
                          oneofKind: "saml";
                          saml: CoreP.IdentityProvider_Spec_SAML;
                        };
                        let g = f.saml.metadataType as {
                          oneofKind: "metadata";
                          metadata: string;
                        };
                        g.metadata = v ?? "";
                        updateReq();
                      }}
                    />
                  )}
                </Tabs.Panel>
              </Tabs>
            </>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="oidcIdentityToken">
          {req.spec!.type.oneofKind === "oidcIdentityToken" && (
            <>
              <Group grow>
                <TextInput
                  label="Issuer"
                  description="Expected value of the identity token's iss claim."
                  placeholder="http://example.com"
                  value={req.spec!.type.oidcIdentityToken.issuer}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidcIdentityToken";
                      oidcIdentityToken: CoreP.IdentityProvider_Spec_OIDCIdentityToken;
                    };
                    f.oidcIdentityToken.issuer = v.target.value;
                    updateReq();
                  }}
                />
                <TextInput
                  label="Audience"
                  description={`Expected value of the identity token's aud claim; defaults to https://${getDomain()}.`}
                  placeholder="my-custom-aud"
                  value={req.spec!.type.oidcIdentityToken.audience}
                  onChange={(v) => {
                    let f = req.spec!.type as {
                      oneofKind: "oidcIdentityToken";
                      oidcIdentityToken: CoreP.IdentityProvider_Spec_OIDCIdentityToken;
                    };
                    f.oidcIdentityToken.audience = v.target.value;
                    updateReq();
                  }}
                />
              </Group>

              <Tabs
                defaultValue={
                  req.spec!.type.oidcIdentityToken.type.oneofKind ?? "issuerURL"
                }
                onChange={(v) => {
                  match(v)
                    .with("issuerURL", () => {
                      let f = req.spec!.type as {
                        oneofKind: "oidcIdentityToken";
                        oidcIdentityToken: IdentityProvider_Spec_OIDCIdentityToken;
                      };
                      match(init.spec!.type)
                        .when(
                          (x) =>
                            x.oneofKind === "oidcIdentityToken" &&
                            x.oidcIdentityToken.type.oneofKind === "issuerURL",
                          (x) => {
                            f.oidcIdentityToken.type = (
                              x as any
                            ).oidcIdentityToken.type;
                          },
                        )
                        .otherwise(() => {
                          f.oidcIdentityToken.type = {
                            oneofKind: "issuerURL",
                            issuerURL: "",
                          };
                        });
                      updateReq();
                    })
                    .with("jwksURL", () => {
                      let f = req.spec!.type as {
                        oneofKind: "oidcIdentityToken";
                        oidcIdentityToken: IdentityProvider_Spec_OIDCIdentityToken;
                      };
                      match(init.spec!.type)
                        .when(
                          (x) =>
                            x.oneofKind === "oidcIdentityToken" &&
                            x.oidcIdentityToken.type.oneofKind === "jwksURL",
                          (x) => {
                            f.oidcIdentityToken.type = (
                              x as any
                            ).oidcIdentityToken.type;
                          },
                        )
                        .otherwise(() => {
                          f.oidcIdentityToken.type = {
                            oneofKind: "jwksURL",
                            jwksURL: "",
                          };
                        });
                      updateReq();
                    })
                    .with("jwksContent", () => {
                      let f = req.spec!.type as {
                        oneofKind: "oidcIdentityToken";
                        oidcIdentityToken: IdentityProvider_Spec_OIDCIdentityToken;
                      };
                      match(init.spec!.type)
                        .when(
                          (x) =>
                            x.oneofKind === "oidcIdentityToken" &&
                            x.oidcIdentityToken.type.oneofKind ===
                              "jwksContent",
                          (x) => {
                            f.oidcIdentityToken.type = (
                              x as any
                            ).oidcIdentityToken.type;
                          },
                        )
                        .otherwise(() => {
                          f.oidcIdentityToken.type = {
                            oneofKind: "jwksContent",
                            jwksContent: "",
                          };
                        });
                      updateReq();
                    });
                }}
              >
                <Tabs.List>
                  <Tabs.Tab value="issuerURL">Issuer URL</Tabs.Tab>
                  <Tabs.Tab value="jwksURL">JWKS URL</Tabs.Tab>
                  <Tabs.Tab value="jwksContent">JWKS Content</Tabs.Tab>
                </Tabs.List>

                <Tabs.Panel value="issuerURL">
                  {req.spec!.type.oidcIdentityToken.type.oneofKind ===
                    "issuerURL" && (
                    <TextInput
                      label="Issuer URL"
                      description="Issuer URL from which the OIDC configuration and JWKS are discovered."
                      placeholder="https://accounts.google.com"
                      value={req.spec!.type.oidcIdentityToken.type.issuerURL}
                      onChange={(v) => {
                        let f = req.spec!.type as {
                          oneofKind: "oidcIdentityToken";
                          oidcIdentityToken: CoreP.IdentityProvider_Spec_OIDCIdentityToken;
                        };
                        let g = f.oidcIdentityToken.type as {
                          oneofKind: "issuerURL";
                          issuerURL: string;
                        };
                        g.issuerURL = v.target.value;
                        updateReq();
                      }}
                    />
                  )}
                </Tabs.Panel>

                <Tabs.Panel value="jwksURL">
                  {req.spec!.type.oidcIdentityToken.type.oneofKind ===
                    "jwksURL" && (
                    <TextInput
                      label="JWKS URL"
                      description="URL of the issuer's JSON Web Key Set."
                      placeholder="https://www.googleapis.com/oauth2/v3/certs"
                      value={req.spec!.type.oidcIdentityToken.type.jwksURL}
                      onChange={(v) => {
                        let f = req.spec!.type as {
                          oneofKind: "oidcIdentityToken";
                          oidcIdentityToken: CoreP.IdentityProvider_Spec_OIDCIdentityToken;
                        };
                        let g = f.oidcIdentityToken.type as {
                          oneofKind: "jwksURL";
                          jwksURL: string;
                        };
                        g.jwksURL = v.target.value;
                        updateReq();
                      }}
                    />
                  )}
                </Tabs.Panel>

                <Tabs.Panel value="jwksContent">
                  {req.spec!.type.oidcIdentityToken.type.oneofKind ===
                    "jwksContent" && (
                    <TextAreaCustom
                      label="JWKS Content"
                      description="Inline JSON Web Key Set used to validate identity tokens."
                      placeholder={`{\n  "keys": [\n    {\n      "use": "sig",\n      "kty": "RSA",\n      ...\n    }\n  ]\n}`}
                      value={req.spec!.type.oidcIdentityToken.type.jwksContent}
                      onChange={(v) => {
                        let f = req.spec!.type as {
                          oneofKind: "oidcIdentityToken";
                          oidcIdentityToken: CoreP.IdentityProvider_Spec_OIDCIdentityToken;
                        };
                        let g = f.oidcIdentityToken.type as {
                          oneofKind: "jwksContent";
                          jwksContent: string;
                        };
                        g.jwksContent = v ?? "";
                        updateReq();
                      }}
                    />
                  )}
                </Tabs.Panel>
              </Tabs>
            </>
          )}
        </Tabs.Panel>
      </Tabs>
    </div>
  );
};

export default Edit;
