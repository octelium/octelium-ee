import * as React from "react";

import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";

import { cloneResource } from "@/utils/pb";
import { FileInput, Textarea } from "@mantine/core";

const Edit = (props: {
  item: EnterpriseP.Secret;
  onUpdate: (item: EnterpriseP.Secret) => void;
}) => {
  let [req, setReq] = React.useState<EnterpriseP.Secret>(props.item);
  let [val, setVal] = React.useState("");
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
      <Textarea
        rows={5}
        label="Set Secret Value"
        placeholder="TopSecret123456"
        required
        value={val}
        onChange={(v) => {
          if (!v) {
            req.data = undefined;
          } else if (v.target.value.length === 0) {
            req.data = undefined;
          } else {
            req.data = {
              type: {
                oneofKind: "value",
                value: v.target.value,
              },
            };
          }
          setVal(v.target.value);
          updateReq();
        }}
      />
      <div className="flex items-end justify-end my-4">
        <FileInput
          label="choose Secret content from a file"
          placeholder="Click to choose select a file"
          accept="text/*"
          onChange={(file) => {
            if (!file) {
              return;
            }

            const reader = new FileReader();

            reader.onload = (e: ProgressEvent<FileReader>) => {
              const result = e.target?.result;

              if (!(result instanceof ArrayBuffer)) {
                return;
              }

              const decoder = new TextDecoder("utf-8", { fatal: true });
              try {
                const value = decoder.decode(result);
                req.data = {
                  type: {
                    oneofKind: "value",
                    value,
                  },
                };

                setVal(value);

                updateReq();
              } catch {
                const bytes = new Uint8Array(result);

                req.data = {
                  type: {
                    oneofKind: "valueBytes",
                    valueBytes: bytes,
                  },
                };

                updateReq();
              }
            };

            reader.onerror = () => {};

            reader.readAsArrayBuffer(file);
          }}
        />
      </div>
    </div>
  );
};

export default Edit;
