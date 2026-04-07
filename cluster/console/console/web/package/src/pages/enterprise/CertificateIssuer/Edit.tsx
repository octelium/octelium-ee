import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";

import { Group, Tabs, TextInput } from "@mantine/core";
import { useState } from "react";
import { match } from "ts-pattern";

const Edit = (props: {
  item: EnterpriseP.CertificateIssuer;
  onUpdate: (item: EnterpriseP.CertificateIssuer) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = useState(EnterpriseP.CertificateIssuer.clone(item));

  const updateReq = () => {
    setReq(EnterpriseP.CertificateIssuer.clone(req));
    onUpdate(req);
  };

  if (!req.spec) {
    return <></>;
  }

  return (
    <div>
      <Tabs
        defaultValue={req.spec.type.oneofKind}
        onChange={(v) => {
          match(v).with("acme", () => {
            req.spec!.type = {
              oneofKind: "acme",
              acme: EnterpriseP.CertificateIssuer_Spec_ACME.create({
                email: "",
                server: "",
                solver: {
                  type: {
                    oneofKind: "dns",
                    dns: EnterpriseP.CertificateIssuer_Spec_ACME_Solver_DNS.create(),
                  },
                },
              }),
            };
          });
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="acme">ACME</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="acme">
          {req.spec.type.oneofKind === `acme` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="ACME Email"
                  placeholder="acme@example.com"
                  value={req.spec.type.acme.email}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `acme`,
                      (x) => {
                        x.acme.email = v.target.value;
                      },
                    );
                    updateReq();
                  }}
                />
                <TextInput
                  required
                  label="ACME Server URL"
                  placeholder="https://acme-v02.api.letsencrypt.org/directory"
                  value={req.spec.type.acme.server}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `acme`,
                      (x) => {
                        x.acme.server = v.target.value;
                      },
                    );
                    updateReq();
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
