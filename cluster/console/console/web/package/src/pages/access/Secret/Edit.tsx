import * as React from "react";

import * as AccessP from "@/apis/accessv1/accessv1";

import SecretTextAreaCustom from "@/components/TextAreaCustom/SecretTextAreaCustom";
import { Alert } from "@mantine/core";
import { KeyRound } from "lucide-react";

const Edit = (props: {
  item: AccessP.Secret;
  onUpdate: (item: AccessP.Secret) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Secret.clone(item));
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    setReq(AccessP.Secret.clone(item));
  }, [itemKey]);

  const updateReq = () => {
    const next = AccessP.Secret.clone(req);
    setReq(next);
    onUpdate(AccessP.Secret.clone(next));
  };

  return (
    <div className="space-y-4">
      <Alert color="blue" icon={<KeyRound size={15} />} title="Write-only value">
        The current value is never returned by the API, so it is always empty
        here. A value is required on every update and it entirely replaces the
        stored one.
      </Alert>

      <SecretTextAreaCustom
        required
        label="Value"
        description="Confidential value that the access Integrations refer to by this Secret's name. It cannot be empty or larger than 512 KiB."
        value={
          req.data?.type.oneofKind === "value" ? req.data.type.value : undefined
        }
        onChange={(v) => {
          req.data = v
            ? AccessP.Secret_Data.create({
                type: { oneofKind: "value", value: v },
              })
            : undefined;
          updateReq();
        }}
      />
    </div>
  );
};

export default Edit;
