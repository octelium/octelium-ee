import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import { cloneResource } from "@/utils/pb";

const Edit = (props: {
  item: CoreP.Group;
  onUpdate: (item: CoreP.Group) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Group>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Group.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Group;
    setReq(clone);

    props.onUpdate(clone);
  };

  if (!req) {
    return <></>;
  }

  return (
    <div>
      <EditItem
        title="Authorization"
        description="Set the Group Policies"
        onUnset={() => {
          req.spec!.authorization = undefined;
          updateReq();
        }}
        obj={req.spec!.authorization}
        onSet={() => {
          if (!req.spec!.authorization) {
            req.spec!.authorization = CoreP.Group_Spec_Authorization.create({
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
