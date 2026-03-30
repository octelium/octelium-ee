import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import { Group, Select } from "@mantine/core";

import EditItem from "@/components/EditItem";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import TimestampPicker from "@/components/TimestampPicker";

const Edit = (props: {
  item: CoreP.Session;
  onUpdate: (item: CoreP.Session) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(CoreP.Session.clone(item));
  const updateReq = () => {
    setReq(CoreP.Session.clone(req));
    onUpdate(req);
  };

  return (
    <div>
      <Group grow>
        <Select
          label="State"
          description="Set the state to ACTIVE, PENDING or REJECTED"
          data={[
            {
              label: "Active",
              value: CoreP.Session_Spec_State[CoreP.Session_Spec_State.ACTIVE],
            },
            {
              label: "Pending",
              value: CoreP.Session_Spec_State[CoreP.Session_Spec_State.PENDING],
            },
            {
              label: "Rejected",
              value:
                CoreP.Session_Spec_State[CoreP.Session_Spec_State.REJECTED],
            },
          ]}
          value={CoreP.Session_Spec_State[req.spec!.state]}
          onChange={(v) => {
            if (!v) {
              return;
            }
            req.spec!.state = CoreP.Session_Spec_State[v as "ACTIVE"];
            updateReq();
          }}
        />

        <TimestampPicker
          label="Expires at"
          description="Set the expiration timestamp of the Session"
          value={req.spec!.expiresAt}
          isFuture
          onChange={(v) => {
            req.spec!.expiresAt = v;
            updateReq();
          }}
        />
      </Group>
      <EditItem
        title="Authorization"
        description="Set the Session Policies"
        onUnset={() => {
          req.spec!.authorization = undefined;
          updateReq();
        }}
        obj={req.spec!.authorization}
        onSet={() => {
          if (!req.spec!.authorization) {
            req.spec!.authorization = CoreP.Session_Spec_Authorization.create({
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
