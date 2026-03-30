import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";
import { cloneResource } from "@/utils/pb";

import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import { Select } from "@mantine/core";

const Edit = (props: {
  item: CoreP.Device;
  onUpdate: (item: CoreP.Device) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Device>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Device.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Device;
    setReq(clone);

    props.onUpdate(clone);
  };

  if (!req) {
    return <></>;
  }

  return (
    <div>
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
          req.spec!.state = CoreP.Device_Spec_State[v as "ACTIVE"];
          updateReq();
        }}
      />

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
    </div>
  );
};

export default Edit;
