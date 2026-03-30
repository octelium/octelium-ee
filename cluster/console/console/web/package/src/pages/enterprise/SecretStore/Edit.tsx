import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";

import { Group, Tabs, TextInput } from "@mantine/core";
import { useState } from "react";
import { match } from "ts-pattern";

const Edit = (props: {
  item: EnterpriseP.SecretStore;
  onUpdate: (item: EnterpriseP.SecretStore) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = useState(EnterpriseP.SecretStore.clone(item));
  let [init, _] = useState(EnterpriseP.SecretStore.clone(req));
  const updateReq = () => {
    setReq(EnterpriseP.SecretStore.clone(req));
    onUpdate(req);
  };

  if (!req.spec) {
    return <></>;
  }

  return (
    <div>
      <Tabs
        defaultValue={req.spec!.type.oneofKind}
        onChange={(v) => {
          match(v)
            .with("hashicorpVault", () => {
              match(init.spec?.type)
                .when(
                  (x) => x?.oneofKind === `hashicorpVault`,
                  (x) => {
                    req.spec!.type = x;
                  },
                )
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "hashicorpVault",
                    hashicorpVault:
                      {} as EnterpriseP.SecretStore_Spec_HashicorpVault,
                  };
                });
            })
            .with("kubernetes", () => {
              match(init.spec?.type)
                .when(
                  (x) => x?.oneofKind === `kubernetes`,
                  (x) => {
                    req.spec!.type = x;
                  },
                )
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "kubernetes",
                    kubernetes: {} as EnterpriseP.SecretStore_Spec_Kubernetes,
                  };
                });
            })
            .with("azureKeyVault", () => {
              match(init.spec?.type)
                .when(
                  (x) => x?.oneofKind === `azureKeyVault`,
                  (x) => {
                    req.spec!.type = x;
                  },
                )
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "azureKeyVault",
                    azureKeyVault:
                      {} as EnterpriseP.SecretStore_Spec_AzureKeyVault,
                  };
                });
            })
            .with("googleCloudKeyManagementService", () => {
              match(init.spec?.type)
                .when(
                  (x) => x?.oneofKind === `googleCloudKeyManagementService`,
                  (x) => {
                    req.spec!.type = x;
                  },
                )
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "googleCloudKeyManagementService",
                    googleCloudKeyManagementService:
                      {} as EnterpriseP.SecretStore_Spec_GoogleCloudKeyManagementService,
                  };
                });
            })
            .with("awsKeyManagementService", () => {
              match(init.spec?.type)
                .when(
                  (x) => x?.oneofKind === `awsKeyManagementService`,
                  (x) => {
                    req.spec!.type = x;
                  },
                )
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "awsKeyManagementService",
                    awsKeyManagementService:
                      {} as EnterpriseP.SecretStore_Spec_AWSKeyManagementService,
                  };
                });
            });

          updateReq();
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="kubernetes">Kubernetes</Tabs.Tab>
          <Tabs.Tab value="hashicorpVault">Hashicorp vault</Tabs.Tab>
          <Tabs.Tab value="awsKeyManagementService">AWS KMS</Tabs.Tab>
          <Tabs.Tab value="azureKeyVault">Azure Key Vault</Tabs.Tab>
          <Tabs.Tab value="googleCloudKeyManagementService">
            Google Key Cloud Management
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="hashicorpVault">
          {req.spec!.type.oneofKind === `hashicorpVault` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Address"
                  placeholder={"vault.example.com"}
                  value={req.spec!.type.hashicorpVault.address}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `hashicorpVault`,
                      (x) => {
                        x.hashicorpVault.address = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Key"
                  placeholder={"my-key"}
                  value={req.spec!.type.hashicorpVault.key}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `hashicorpVault`,
                      (x) => {
                        x.hashicorpVault.key = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Role"
                  placeholder={"my-role"}
                  value={req.spec!.type.hashicorpVault.role}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `hashicorpVault`,
                      (x) => {
                        x.hashicorpVault.role = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
              </Group>
            </>
          )}
        </Tabs.Panel>
        <Tabs.Panel value="awsKeyManagementService">
          {req.spec.type.oneofKind === `awsKeyManagementService` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Key ID"
                  placeholder={"ABCDEDF123456"}
                  value={req.spec!.type.awsKeyManagementService.keyID}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `awsKeyManagementService`,
                      (x) => {
                        x.awsKeyManagementService.keyID = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Region"
                  placeholder={"eu-central-1"}
                  value={req.spec!.type.awsKeyManagementService.region}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `awsKeyManagementService`,
                      (x) => {
                        x.awsKeyManagementService.region = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Role ARN"
                  placeholder={"ABCDEDF123456"}
                  value={req.spec!.type.awsKeyManagementService.roleARN}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `awsKeyManagementService`,
                      (x) => {
                        x.awsKeyManagementService.roleARN = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
              </Group>
            </>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="azureKeyVault">
          {req.spec.type.oneofKind === `azureKeyVault` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Client ID"
                  placeholder={"abcdedf123456"}
                  value={req.spec!.type.azureKeyVault.clientID}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `azureKeyVault`,
                      (x) => {
                        x.azureKeyVault.clientID = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Tenant ID"
                  placeholder={"eu-central-1"}
                  value={req.spec!.type.azureKeyVault.tenantID}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `azureKeyVault`,
                      (x) => {
                        x.azureKeyVault.tenantID = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Vault URL"
                  placeholder={"ABCDEDF123456"}
                  value={req.spec!.type.azureKeyVault.vaultURL}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `azureKeyVault`,
                      (x) => {
                        x.azureKeyVault.vaultURL = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
              </Group>
            </>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="googleCloudKeyManagementService">
          {req.spec.type.oneofKind === `googleCloudKeyManagementService` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Location"
                  placeholder={"eu-central-1"}
                  value={
                    req.spec!.type.googleCloudKeyManagementService.location
                  }
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `googleCloudKeyManagementService`,
                      (x) => {
                        x.googleCloudKeyManagementService.location =
                          v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Key"
                  placeholder={"my-key"}
                  value={req.spec!.type.googleCloudKeyManagementService.key}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `googleCloudKeyManagementService`,
                      (x) => {
                        x.googleCloudKeyManagementService.key = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Key Ring"
                  placeholder={"my-key-ring"}
                  value={req.spec!.type.googleCloudKeyManagementService.keyRing}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `googleCloudKeyManagementService`,
                      (x) => {
                        x.googleCloudKeyManagementService.keyRing =
                          v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />

                <TextInput
                  required
                  label="Project"
                  placeholder={"my-project"}
                  value={req.spec!.type.googleCloudKeyManagementService.project}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `googleCloudKeyManagementService`,
                      (x) => {
                        x.googleCloudKeyManagementService.project =
                          v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
              </Group>
            </>
          )}
        </Tabs.Panel>
      </Tabs>
    </div>
  );
};

export default Edit;
