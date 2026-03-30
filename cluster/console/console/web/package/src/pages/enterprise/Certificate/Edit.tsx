import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Group, Select } from "@mantine/core";
import * as React from "react";

const Edit = (props: {
  item: EnterpriseP.Certificate;
  onUpdate: (item: EnterpriseP.Certificate) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(EnterpriseP.Certificate.clone(item));
  let [init, _] = React.useState(EnterpriseP.Certificate.clone(req));
  const updateReq = () => {
    setReq(EnterpriseP.Certificate.clone(req));
    onUpdate(req);
  };

  if (!req.spec) {
    return <></>;
  }

  return (
    <div>
      <Group grow>
        <Select
          label="Mode"
          required
          description="Set the Certificate mode to MANAGED OR MANUAL"
          data={[
            {
              label: "Managed",
              value:
                EnterpriseP.Certificate_Spec_Mode[
                  EnterpriseP.Certificate_Spec_Mode.MANAGED
                ],
            },
            {
              label: "Manual",
              value:
                EnterpriseP.Certificate_Spec_Mode[
                  EnterpriseP.Certificate_Spec_Mode.MANUAL
                ],
            },
          ]}
          defaultValue={
            EnterpriseP.Certificate_Spec_Mode[req.spec!.mode] ??
            EnterpriseP.Certificate_Spec_Mode[
              EnterpriseP.Certificate_Spec_Mode.MANAGED
            ]
          }
          onChange={(v) => {
            req.spec!.mode = EnterpriseP.Certificate_Spec_Mode[v as "MANAGED"];
            updateReq();
          }}
        />
      </Group>
    </div>
  );
};

export default Edit;
