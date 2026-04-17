import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import SecretTextAreaCustom from "@/components/TextAreaCustom/SecretTextAreaCustom";
import { cloneResource } from "@/utils/pb";

const Edit = (props: {
  item: CoreP.Secret;
  onUpdate: (item: CoreP.Secret) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Secret>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Secret.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Secret;
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
