import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import * as React from "react";
import { Group, Switch, Tabs, TextInput } from "@mantine/core";
import { useResourceForm } from "@/pages/utils/form";
import { match } from "ts-pattern";
import SelectSecret from "@/components/ResourceLayout/SelectSecret";

const Edit = (props: {
  item: EnterpriseP.DNSProvider;
  onUpdate: (item: EnterpriseP.DNSProvider) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(EnterpriseP.DNSProvider.clone(item));
  let [init, _] = React.useState(EnterpriseP.DNSProvider.clone(req));
  const updateReq = () => {
    setReq(EnterpriseP.DNSProvider.clone(req));
    onUpdate(req);
  };

  return (
    <div>
      <Tabs
        defaultValue={req.spec!.type.oneofKind}
        onChange={(v) => {
          match(v)
            .with("cloudflare", () => {
              match(init.spec!.type.oneofKind)
                .with(`cloudflare`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "cloudflare",
                    cloudflare: EnterpriseP.DNSProvider_Spec_Cloudflare.create({
                      apiToken: {
                        type: {
                          oneofKind: "fromSecret",
                          fromSecret: "",
                        },
                      },
                    }),
                  };
                });
            })
            .with("aws", () => {
              match(init.spec!.type.oneofKind)
                .with(`aws`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "aws",
                    aws: EnterpriseP.DNSProvider_Spec_AWS.create({
                      secretAccessKey: {
                        type: {
                          oneofKind: "fromSecret",
                          fromSecret: "",
                        },
                      },
                    }),
                  };
                });
            });

          updateReq();
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="cloudflare">Cloudflare</Tabs.Tab>
          <Tabs.Tab value="aws">AWS Route 53</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="cloudflare">
          {req.spec!.type.oneofKind === `cloudflare` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Email"
                  placeholder="cloudflare@example.com"
                  value={req.spec!.type.cloudflare.email}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `cloudflare`,
                      (x) => {
                        x.cloudflare.email = v.target.value;
                        updateReq();
                      }
                    );
                  }}
                />

                <SelectSecret
                  api="enterprise"
                  defaultValue={
                    req.spec!.type.cloudflare.apiToken?.type.oneofKind ===
                    `fromSecret`
                      ? req.spec!.type.cloudflare.apiToken.type.fromSecret
                      : undefined
                  }
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `cloudflare`,
                      (x) => {
                        match(x.cloudflare.apiToken?.type).when(
                          (x) => x?.oneofKind === `fromSecret`,
                          (x) => {
                            x.fromSecret = v ?? "";
                            updateReq();
                          }
                        );
                      }
                    );
                  }}
                />
                <Switch
                  label="Proxied"
                  checked={req.spec?.type.cloudflare.proxied}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `cloudflare`,
                      (x) => {
                        x.cloudflare.proxied = v.target.checked;
                        updateReq();
                      }
                    );
                  }}
                />
              </Group>
            </>
          )}
        </Tabs.Panel>
        <Tabs.Panel value="aws">
          {req.spec!.type.oneofKind === `aws` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Access Key ID"
                  placeholder="ABCDEF123456"
                  value={req.spec!.type.aws.accessKeyID}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `aws`,
                      (x) => {
                        x.aws.accessKeyID = v.target.value;
                        updateReq();
                      }
                    );
                  }}
                />

                <TextInput
                  required
                  label="Region"
                  placeholder="eu-central-1"
                  value={req.spec!.type.aws.region}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `aws`,
                      (x) => {
                        x.aws.region = v.target.value;
                        updateReq();
                      }
                    );
                  }}
                />

                <TextInput
                  label="Assume Role ARN"
                  placeholder="arn:aws:iam::YYYYYYYYYYYY:role/dns-manager"
                  value={req.spec!.type.aws.assumeRoleARN}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `aws`,
                      (x) => {
                        x.aws.assumeRoleARN = v.target.value;
                        updateReq();
                      }
                    );
                  }}
                />

                <SelectSecret
                  api="enterprise"
                  defaultValue={match(req.spec!.type.aws.secretAccessKey?.type)
                    .when(
                      (x) => x?.oneofKind === `fromSecret`,
                      (x) => x.fromSecret
                    )
                    .otherwise(() => undefined)}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `aws`,
                      (x) => {
                        match(x.aws.secretAccessKey?.type).when(
                          (x) => x?.oneofKind === `fromSecret`,
                          (x) => {
                            x.fromSecret = v ?? "";
                            updateReq();
                          }
                        );
                      }
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
