import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import { cloneResource } from "@/utils/pb";

import { Group, Select, TextInput } from "@mantine/core";

const Edit = (props: {
  item: CoreP.Authenticator;
  onUpdate: (item: CoreP.Authenticator) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Authenticator>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Authenticator.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Authenticator;
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
          label="State"
          description="Set the state to ACTIVE, PENDING or REJECTED"
          data={[
            {
              label: "Active",
              value: CoreP.Device_Spec_State[CoreP.Device_Spec_State.ACTIVE],
            },
            {
              label: "Pending",
              value: CoreP.Device_Spec_State[CoreP.Device_Spec_State.PENDING],
            },
            {
              label: "Rejected",
              value: CoreP.Device_Spec_State[CoreP.Device_Spec_State.REJECTED],
            },
          ]}
          value={CoreP.Device_Spec_State[req.spec!.state]}
          onChange={(v) => {
            if (!v) {
              return;
            }
            req.spec!.state = CoreP.Authenticator_Spec_State[v as "ACTIVE"];
            updateReq();
          }}
        />

        <TextInput
          label="Display Name"
          placeholder="My PC TPM"
          description="Set the Authenticator display name"
          value={req.spec!.displayName}
          onChange={(v) => {
            req.spec!.displayName = v.target.value;
            updateReq();
          }}
        />
      </Group>
    </div>
  );
};

export default Edit;
