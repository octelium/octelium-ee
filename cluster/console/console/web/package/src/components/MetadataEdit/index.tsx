import { Metadata } from "@/apis/metav1/metav1";
import { getShortNameFromStr, Resource } from "@/utils/pb";
import { SimpleGrid, TagsInput, Textarea, TextInput } from "@mantine/core";
import { AlignLeft, LockKeyhole, ShieldCheck, Tag, Type } from "lucide-react";

const sharedInputStyles = {
  label: {
    fontSize: "0.74rem",
    fontWeight: 700,
    color: "#334155",
    marginBottom: "5px",
  },
  description: {
    fontSize: "0.68rem",
    fontWeight: 600,
    lineHeight: 1.4,
    color: "#94a3b8",
    marginBottom: "6px",
  },
  input: {
    fontSize: "0.8rem",
    fontWeight: 600,
    backgroundColor: "rgba(248, 250, 252, 0.85)",
    border: "1px solid #e2e8f0",
    borderRadius: "8px",
    color: "#1e293b",
    boxShadow: "0 1px 2px rgba(15, 23, 42, 0.03)",
    transition:
      "background-color 500ms ease, border-color 500ms ease, box-shadow 500ms ease",
    "&:focus": {
      backgroundColor: "#ffffff",
      borderColor: "#94a3b8",
      boxShadow: "0 0 0 3px rgba(148, 163, 184, 0.16)",
    },
    "&:disabled": {
      backgroundColor: "#f1f5f9",
      color: "#64748b",
      borderColor: "#e2e8f0",
      opacity: 1,
      cursor: "not-allowed",
    },
  },
  section: {
    color: "#94a3b8",
  },
};

const MetadataEdit = (props: {
  item: Resource;
  onUpdate: (md: Metadata) => void;
  parentName?: string;
  skipDisplayName?: boolean;
  isUpdateMode?: boolean;
}) => {
  const req = props.item.metadata ?? Metadata.create();
  const sourceMetadata = props.item.metadata;
  const isSystem = !!sourceMetadata?.isSystem;
  const isNameLocked = isSystem || !!props.isUpdateMode;

  const update = (partial: Partial<Metadata>) => {
    const next = Metadata.clone(sourceMetadata ?? Metadata.create());
    Object.assign(next, partial);
    props.onUpdate(next);
  };

  return (
    <div className="flex w-full flex-col gap-3">
      {isSystem && (
        <div className="flex items-start gap-2 rounded-lg border border-blue-200/80 bg-blue-50/70 px-3 py-2">
          <ShieldCheck
            size={15}
            strokeWidth={2.25}
            className="mt-0.5 shrink-0 text-blue-600"
          />
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="text-[0.72rem] font-bold text-blue-800">
              System-managed metadata
            </span>
            <span className="text-[0.65rem] font-semibold leading-4 text-blue-600">
              Managed by the Cluster; metadata cannot be changed here.
            </span>
          </div>
        </div>
      )}

      <SimpleGrid
        cols={{ base: 1, sm: props.skipDisplayName ? 1 : 2 }}
        spacing="sm"
        verticalSpacing="sm"
      >
        <TextInput
          value={getShortNameFromStr(req.name)}
          label="Name"
          description={
            props.isUpdateMode
              ? "Names are immutable after creation."
              : props.parentName
                ? `Unique name under ${props.parentName}.`
                : "Must be unique within this API, version, and kind."
          }
          placeholder="my-resource"
          required
          disabled={isNameLocked}
          leftSection={
            isNameLocked ? (
              <LockKeyhole size={13} strokeWidth={2.25} />
            ) : (
              <Tag size={13} strokeWidth={2.25} />
            )
          }
          onChange={(event) => {
            const name = event.currentTarget.value;
            update({
              name: props.parentName ? `${name}.${props.parentName}` : name,
            });
          }}
          styles={sharedInputStyles}
        />

        {!props.skipDisplayName && (
          <TextInput
            value={req.displayName}
            label="Display name"
            description="Optional public name; it does not need to be unique."
            placeholder="My Resource"
            disabled={isSystem}
            leftSection={<Type size={13} strokeWidth={2.25} />}
            onChange={(event) =>
              update({ displayName: event.currentTarget.value })
            }
            styles={sharedInputStyles}
          />
        )}
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm" verticalSpacing="sm">
        <TagsInput
          label="Tags"
          disabled={isSystem}
          placeholder="Add a tag"
          description="Optional tags that classify this resource."
          value={req.tags}
          leftSection={<Tag size={13} strokeWidth={2.25} />}
          onChange={(tags) => update({ tags })}
          clearable
          styles={{
            ...sharedInputStyles,
            pill: {
              height: "22px",
              fontSize: "0.68rem",
              fontWeight: 700,
              backgroundColor: "#f1f5f9",
              color: "#475569",
              border: "1px solid #e2e8f0",
              borderRadius: "6px",
            },
            input: {
              ...sharedInputStyles.input,
              minHeight: "36px",
            },
          }}
        />

        <Textarea
          value={req.description}
          disabled={isSystem}
          label="Description"
          description="Short description of this resource, up to 1000 characters."
          placeholder="Describe this resource…"
          minRows={2}
          autosize
          maxRows={4}
          leftSection={<AlignLeft size={13} strokeWidth={2.25} />}
          onChange={(event) =>
            update({ description: event.currentTarget.value })
          }
          styles={{
            ...sharedInputStyles,
            input: {
              ...sharedInputStyles.input,
              lineHeight: 1.45,
              paddingTop: "8px",
              paddingBottom: "8px",
            },
          }}
        />
      </SimpleGrid>
    </div>
  );
};

export default MetadataEdit;
