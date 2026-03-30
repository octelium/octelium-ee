import * as React from "react";

import { Metadata } from "@/apis/metav1/metav1";
import { getShortNameFromStr, Resource } from "@/utils/pb";
import { Group, TagsInput, Textarea, TextInput } from "@mantine/core";

const MetadataEdit = (props: {
  item: Resource;
  onUpdate: (md: Metadata) => void;
  parentName?: string;
  skipDisplayName?: boolean;
  isUpdateMode?: boolean;
}) => {
  const { isUpdateMode } = props;
  let [req, setReq] = React.useState(
    Metadata.clone(props.item.metadata ?? Metadata.create()),
  );

  const disabled = props.item.metadata?.isSystem;

  return (
    <div className="w-full">
      <Group grow>
        {
          <TextInput
            value={getShortNameFromStr(req.name)}
            label="Name"
            placeholder="my-resource"
            required
            disabled={disabled || isUpdateMode}
            onChange={(v) => {
              const arg = v.target.value as string;
              req!.name = props.parentName ? `${arg}.${props.parentName}` : arg;
              setReq(Metadata.clone(req));
              props.onUpdate(req);
            }}
          />
        }

        {!props.skipDisplayName && (
          <TextInput
            value={req.displayName}
            label="Display Name"
            placeholder="My Resource"
            disabled={disabled}
            onChange={(v) => {
              req.displayName = v.target.value as string;
              setReq(Metadata.clone(req));
              props.onUpdate(req);
            }}
          />
        )}
      </Group>

      <TagsInput
        label="Tags"
        disabled={disabled}
        placeholder="dev, ops, production, sensitive"
        description="Set one or more tags to describe the resource"
        value={req.tags}
        onChange={(v) => {
          req.tags = v;
          setReq(Metadata.clone(req));
          props.onUpdate(req);
        }}
      />

      <Textarea
        value={req.description}
        disabled={disabled}
        label="Description"
        placeholder="This is a short description on what this resource is about..."
        rows={3}
        onChange={(v) => {
          req.description = v.target.value as string;
          setReq(Metadata.clone(req));
          props.onUpdate(req);
        }}
      />
    </div>
  );
};

export default MetadataEdit;
