import * as React from "react";

import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";

import SecretTextAreaCustom from "@/components/TextAreaCustom/SecretTextAreaCustom";
import { cloneResource } from "@/utils/pb";

const Edit = (props: {
  item: EnterpriseP.Secret;
  onUpdate: (item: EnterpriseP.Secret) => void;
}) => {
  let [req, setReq] = React.useState<EnterpriseP.Secret>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(EnterpriseP.Secret.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as EnterpriseP.Secret;
    setReq(clone);

    props.onUpdate(clone);
  };

  if (!req) {
    return <></>;
  }

  return (
    <div>
      <SecretTextAreaCustom
        value={
          req.data?.type.oneofKind === `value` ? req.data.type.value : undefined
        }
        onChange={(v) => {
          req.data = {
            type: {
              oneofKind: `value`,
              value: v ?? "",
            },
          };
          updateReq();
        }}
      />
    </div>
  );
};

export default Edit;
