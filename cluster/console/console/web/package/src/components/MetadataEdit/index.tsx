import { Metadata } from "@/apis/metav1/metav1";
import { getShortNameFromStr, Resource } from "@/utils/pb";
import { Group, TagsInput, Textarea, TextInput } from "@mantine/core";
import * as React from "react";

const sharedInputStyles = {
  label: {
    fontSize: "0.72rem",
    fontWeight: 700,
    fontFamily: "Ubuntu, sans-serif",
    textTransform: "uppercase" as const,
    letterSpacing: "0.05em",
    color: "#475569",
    marginBottom: "4px",
  },
  description: {
    fontSize: "0.7rem",
    fontWeight: 600,
    fontFamily: "Ubuntu, sans-serif",
    color: "#94a3b8",
    marginBottom: "4px",
  },
  input: {
    fontSize: "0.82rem",
    fontWeight: 600,
    fontFamily: "Ubuntu, sans-serif",
    backgroundColor: "#ffffff",
    border: "1px solid #e2e8f0",
    borderRadius: "6px",
    color: "#1e293b",
    boxShadow: "0 1px 3px rgba(15,23,42,0.05)",
    "&:focus": {
      borderColor: "#94a3b8",
      boxShadow: "0 0 0 2px rgba(148,163,184,0.2)",
    },
    "&:disabled": {
      backgroundColor: "#f8fafc",
      color: "#94a3b8",
      borderColor: "#e2e8f0",
      cursor: "not-allowed",
    },
  },
};

const MetadataEdit = (props: {
  item: Resource;
  onUpdate: (md: Metadata) => void;
  parentName?: string;
  skipDisplayName?: boolean;
  isUpdateMode?: boolean;
}) => {
  const { isUpdateMode } = props;
  const [req, setReq] = React.useState(
    Metadata.clone(props.item.metadata ?? Metadata.create()),
  );

  const disabled = props.item.metadata?.isSystem;

  const update = (partial: Partial<typeof req>) => {
    Object.assign(req, partial);
    const cloned = Metadata.clone(req);
    setReq(cloned);
    props.onUpdate(cloned);
  };

  return (
    <div className="w-full flex flex-col gap-4">
      <Group grow align="flex-start" gap="md">
        <TextInput
          value={getShortNameFromStr(req.name)}
          label="Name"
          placeholder="my-resource"
          required
          disabled={disabled || isUpdateMode}
          onChange={(v) => {
            const arg = v.target.value;
            update({
              name: props.parentName ? `${arg}.${props.parentName}` : arg,
            });
          }}
          styles={sharedInputStyles}
        />

        {!props.skipDisplayName && (
          <TextInput
            value={req.displayName}
            label="Display Name"
            placeholder="My Resource"
            disabled={disabled}
            onChange={(v) => update({ displayName: v.target.value })}
            styles={sharedInputStyles}
          />
        )}
      </Group>

      <TagsInput
        label="Tags"
        disabled={disabled}
        placeholder="dev, ops, production, sensitive"
        description="One or more tags to describe this resource"
        value={req.tags}
        onChange={(v) => update({ tags: v })}
        styles={{
          ...sharedInputStyles,
          pill: {
            fontSize: "0.7rem",
            fontWeight: 700,
            fontFamily: "Ubuntu, sans-serif",
            backgroundColor: "#f1f5f9",
            color: "#334155",
            border: "1px solid #e2e8f0",
          },
          input: {
            ...sharedInputStyles.input,
            minHeight: "36px",
          },
        }}
      />

      <Textarea
        value={req.description}
        disabled={disabled}
        label="Description"
        placeholder="A short description of what this resource is for..."
        rows={3}
        onChange={(v) => update({ description: v.target.value })}
        styles={{
          ...sharedInputStyles,
          input: {
            ...sharedInputStyles.input,
            resize: "vertical" as const,
            lineHeight: "1.6",
          },
        }}
      />
    </div>
  );
};

export default MetadataEdit;
