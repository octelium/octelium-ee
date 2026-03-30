import * as CoreP from "@/apis/corev1/corev1";
import { InlinePolicy } from "@/apis/corev1/corev1";
import Edit from "@/pages/core/Policy/Edit";
import * as React from "react";
import ItemMessage from "../ItemMessage";

const SelectInlinePolicies = (props: {
  inlinePolicies: InlinePolicy[];
  onUpdate: (inlinePolicies: InlinePolicy[]) => void;
}) => {
  let [req, setReq] = React.useState({
    inlinePolicies: props.inlinePolicies,
  });
  const data = props.inlinePolicies;

  const updateReq = () => {
    setReq(req);

    props.onUpdate(req.inlinePolicies);
  };

  return (
    <div className="w-full">
      <ItemMessage
        title="Inline Policies"
        obj={req.inlinePolicies}
        isList
        onSet={() => {
          req.inlinePolicies = [
            CoreP.InlinePolicy.create({
              spec: {
                rules: [CoreP.Policy_Spec_Rule.create()],
              },
            }),
          ];
          updateReq();
        }}
        onAddListItem={() => {
          req.inlinePolicies.push(
            CoreP.InlinePolicy.create({
              spec: {
                rules: [CoreP.Policy_Spec_Rule.create()],
              },
            }),
          );
          updateReq();
        }}
      >
        {req.inlinePolicies &&
          req.inlinePolicies.map((x, idx) => (
            <div className="w-full">
              <Edit
                item={CoreP.Policy.create({
                  spec: x.spec,
                })}
                onUpdate={(itm) => {
                  req.inlinePolicies[idx].spec = itm.spec;
                  updateReq();
                }}
              />
            </div>
          ))}
      </ItemMessage>
    </div>
  );
};

export default SelectInlinePolicies;
