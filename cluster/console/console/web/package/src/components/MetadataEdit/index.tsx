import { Metadata } from "@/apis/metav1/metav1";
import { getShortNameFromStr, Resource } from "@/utils/pb";
import {
  ActionIcon,
  Button,
  Collapse,
  SimpleGrid,
  TagsInput,
  Textarea,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  AlignLeft,
  ChevronDown,
  LockKeyhole,
  Plus,
  ShieldCheck,
  Tag,
  Trash2,
  Type,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import Section from "../Section";

interface Row {
  id: string;
  key: string;
  value: string;
}

const createId = () =>
  globalThis.crypto?.randomUUID?.() ??
  `row-${Date.now()}-${Math.random().toString(36).slice(2)}`;

const toRows = (map: Record<string, string> | undefined): Row[] =>
  Object.entries(map ?? {}).map(([key, value]) => ({
    id: createId(),
    key,
    value,
  }));

const toMap = (rows: Row[]): Record<string, string> => {
  const next: Record<string, string> = {};
  for (const row of rows) {
    if (row.key.trim().length === 0) continue;
    next[row.key.trim()] = row.value;
  }
  return next;
};

const KeyValueEditor = (props: {
  resourceUID: string;
  value: Record<string, string> | undefined;
  disabled?: boolean;
  keyPlaceholder: string;
  valuePlaceholder: string;
  onChange: (next: Record<string, string>) => void;
}) => {
  const [rows, setRows] = React.useState<Row[]>(() => toRows(props.value));

  React.useEffect(() => {
    setRows(toRows(props.value));
  }, [props.resourceUID]);

  const apply = (next: Row[]) => {
    setRows(next);
    props.onChange(toMap(next));
  };

  return (
    <div className="flex flex-col gap-2">
      {rows.map((row, index) => (
        <div key={row.id} className="flex items-center gap-2">
          <TextInput
            className="flex-1"
            size="xs"
            aria-label="Key"
            placeholder={props.keyPlaceholder}
            disabled={props.disabled}
            value={row.key}
            onChange={(event) => {
              const next = [...rows];
              next[index] = { ...row, key: event.currentTarget.value };
              apply(next);
            }}
          />
          <TextInput
            className="flex-1"
            size="xs"
            aria-label="Value"
            placeholder={props.valuePlaceholder}
            disabled={props.disabled}
            value={row.value}
            onChange={(event) => {
              const next = [...rows];
              next[index] = { ...row, value: event.currentTarget.value };
              apply(next);
            }}
          />
          <Tooltip label="Remove" withArrow>
            <ActionIcon
              type="button"
              variant="subtle"
              color="red"
              size="sm"
              disabled={props.disabled}
              aria-label={`Remove ${row.key || "entry"}`}
              onClick={() => apply(rows.filter((entry) => entry.id !== row.id))}
            >
              <Trash2 size={13} strokeWidth={2.1} />
            </ActionIcon>
          </Tooltip>
        </div>
      ))}

      <div>
        <Button
          type="button"
          variant="default"
          size="compact-xs"
          disabled={props.disabled}
          leftSection={<Plus size={11} strokeWidth={2.5} />}
          onClick={() =>
            apply([...rows, { id: createId(), key: "", value: "" }])
          }
        >
          Add entry
        </Button>
      </div>
    </div>
  );
};

const MetadataEdit = (props: {
  item: Resource;
  onUpdate: (md: Metadata) => void;
  parentName?: string;
  skipDisplayName?: boolean;
  isUpdateMode?: boolean;
  error?: string;
}) => {
  const req = props.item.metadata ?? Metadata.create();
  const sourceMetadata = props.item.metadata;
  const isSystem = !!sourceMetadata?.isSystem;
  const isNameLocked = isSystem || !!props.isUpdateMode;

  const hasLabelsOrAnnotations =
    Object.keys(req.labels ?? {}).length > 0 ||
    Object.keys(req.annotations ?? {}).length > 0;
  const [moreOpen, setMoreOpen] = React.useState(hasLabelsOrAnnotations);

  const update = (partial: Partial<Metadata>) => {
    const next = Metadata.clone(sourceMetadata ?? Metadata.create());
    Object.assign(next, partial);
    props.onUpdate(next);
  };

  return (
    <div className="flex w-full flex-col gap-4">
      {isSystem && (
        <div className="flex items-start gap-2 rounded-lg border border-blue-200/80 bg-blue-50/70 px-3 py-2">
          <ShieldCheck
            size={15}
            strokeWidth={2.25}
            className="mt-0.5 shrink-0 text-blue-600"
          />
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="text-xs font-semibold text-blue-800">
              System-managed metadata
            </span>
            <span className="text-xs font-normal leading-5 text-blue-700">
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
                : "Lowercase letters, digits and dashes."
          }
          placeholder="my-resource"
          required
          error={isNameLocked ? undefined : props.error}
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
        />
      </SimpleGrid>

      <div className="rounded-lg border border-slate-200/70">
        <button
          type="button"
          onClick={() => setMoreOpen((value) => !value)}
          aria-expanded={moreOpen}
          className="group flex w-full items-center gap-2 px-3.5 py-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <ChevronDown
            size={14}
            strokeWidth={2.25}
            className={twMerge(
              "shrink-0 text-slate-500 transition-transform duration-150",
              !moreOpen && "-rotate-90",
            )}
          />
          <span className="text-body font-semibold text-slate-900">
            Labels &amp; annotations
          </span>
          {hasLabelsOrAnnotations && (
            <span className="rounded-full border border-slate-200 bg-slate-50 px-1.5 py-px text-micro font-medium text-slate-600">
              {Object.keys(req.labels ?? {}).length +
                Object.keys(req.annotations ?? {}).length}
            </span>
          )}
          {!moreOpen && (
            <span className="text-xs font-normal text-slate-500">More</span>
          )}
        </button>

        <Collapse expanded={moreOpen} transitionDuration={150}>
          <div className="flex flex-col gap-4 border-t border-slate-100 px-3.5 py-3.5">
            <Section
              level={2}
              title="Labels"
              description="Key/value pairs used by selectors and Policies."
              obj={req.labels}
              onSet={() => update({ labels: {} })}
              onUnset={() => update({ labels: {} })}
              noDelete
            >
              <KeyValueEditor
                resourceUID={req.uid}
                value={req.labels}
                disabled={isSystem}
                keyPlaceholder="team"
                valuePlaceholder="platform"
                onChange={(labels) => update({ labels })}
              />
            </Section>

            <Section
              level={2}
              title="Annotations"
              description="Free-form metadata that is not used for selection."
              obj={req.annotations}
              onSet={() => update({ annotations: {} })}
              onUnset={() => update({ annotations: {} })}
              noDelete
            >
              <KeyValueEditor
                resourceUID={req.uid}
                value={req.annotations}
                disabled={isSystem}
                keyPlaceholder="octelium.com/owner"
                valuePlaceholder="platform-team"
                onChange={(annotations) => update({ annotations })}
              />
            </Section>
          </div>
        </Collapse>
      </div>
    </div>
  );
};

export default MetadataEdit;
