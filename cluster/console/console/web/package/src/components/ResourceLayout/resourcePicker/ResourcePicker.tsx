import ResourceYAML from "@/components/ResourceYAML";
import { Resource } from "@/utils/pb";
import { Alert, Button, Drawer, Loader, TextInput } from "@mantine/core";
import { Check, Plus, Search, X } from "lucide-react";
import * as React from "react";
import { resourceIcon } from "./facts";
import ResourceRow from "./ResourceRow";

const ResourcePicker = (props: {
  opened: boolean;
  onClose: () => void;
  api: string;
  kind: string;
  multiple?: boolean;
  selected: string[];
  items: Resource[];
  isLoading: boolean;
  isError: boolean;
  errorMessage?: string;
  onRetry: () => void;
  search: string;
  onSearchChange: (value: string) => void;
  onPick: (item: Resource) => void;
  onClear?: () => void;
  onCreate?: () => void;
}) => {
  const { items } = props;
  const [peek, setPeek] = React.useState<Resource>();
  const [active, setActive] = React.useState(0);
  const rows = React.useRef<Array<HTMLLIElement | null>>([]);
  const Icon = resourceIcon(props.api, props.kind);

  React.useEffect(() => {
    setActive(0);
  }, [props.search, props.opened]);

  React.useEffect(() => {
    if (active < 0 || active >= items.length) return;
    rows.current[active]?.scrollIntoView({ block: "nearest" });
  }, [active, items.length]);

  const move = (delta: number) => {
    if (items.length === 0) return;
    setActive((current) => {
      const next = current + delta;
      if (next < 0) return items.length - 1;
      if (next >= items.length) return 0;
      return next;
    });
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      move(1);
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      move(-1);
      return;
    }
    if (event.key === "Enter") {
      const item = items[active];
      if (!item) return;
      event.preventDefault();
      props.onPick(item);
    }
  };

  const selectedCount = props.selected.length;

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="min(620px, 100vw)"
      padding={0}
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      transitionProps={{
        transition: "slide-left",
        duration: 240,
        exitDuration: 200,
      }}
      title={
        <div className="flex min-w-0 items-center gap-2">
          <Icon
            size={15}
            strokeWidth={2.2}
            className="shrink-0 text-slate-500"
          />
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            {props.api}
          </span>
          <span className="truncate text-sm font-bold text-slate-900">
            Select {props.kind}
          </span>
        </div>
      }
      styles={{
        header: {
          borderBottom: "1px solid var(--color-slate-200)",
          minHeight: "56px",
          paddingInline: "16px",
        },
        body: {
          height: "calc(100dvh - 56px)",
          padding: 0,
          display: "flex",
          flexDirection: "column",
          backgroundColor: "var(--color-slate-50)",
        },
      }}
    >
      <div className="flex shrink-0 flex-col gap-2 border-b border-slate-200 bg-white px-4 py-3">
        <TextInput
          data-autofocus
          size="sm"
          placeholder={`Search ${props.kind} by name, display name or description…`}
          value={props.search}
          onChange={(event) => props.onSearchChange(event.currentTarget.value)}
          onKeyDown={handleKeyDown}
          leftSection={<Search size={14} strokeWidth={2.1} />}
          rightSection={
            props.isLoading ? (
              <Loader size={14} color="gray" />
            ) : props.search ? (
              <button
                type="button"
                aria-label="Clear the search"
                onClick={() => props.onSearchChange("")}
                className="cursor-pointer text-slate-400 transition-colors duration-150 hover:text-slate-700"
              >
                <X size={14} strokeWidth={2.4} />
              </button>
            ) : null
          }
        />

        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-micro font-normal text-slate-500">
            {props.isLoading
              ? "Loading…"
              : `${items.length} ${items.length === 1 ? "result" : "results"}`}
            {selectedCount > 0 && ` · ${selectedCount} selected`}
          </span>

          <span className="flex items-center gap-2">
            {props.onClear && selectedCount > 0 && (
              <Button
                type="button"
                variant="subtle"
                color="gray"
                size="compact-xs"
                onClick={props.onClear}
              >
                Clear selection
              </Button>
            )}
            <span className="hidden text-micro font-normal text-slate-400 sm:inline">
              ↑↓ to move · ↵ to select
            </span>
          </span>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
        {props.isError ? (
          <Alert color="red" title={`Could not load the ${props.kind} list`}>
            <div className="flex flex-col gap-2">
              <span className="text-xs">{props.errorMessage}</span>
              <Button
                size="compact-xs"
                variant="outline"
                onClick={props.onRetry}
              >
                Try again
              </Button>
            </div>
          </Alert>
        ) : props.isLoading && items.length === 0 ? (
          <div className="flex flex-col gap-2">
            {[0, 1, 2, 3, 4, 5].map((index) => (
              <div
                key={index}
                className="h-[86px] animate-pulse rounded-xl border border-slate-200 bg-white"
              />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="flex min-h-40 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-slate-200 bg-white px-6 py-8 text-center">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-slate-200 bg-slate-50 text-slate-500">
              <Icon size={17} strokeWidth={2.2} />
            </span>
            <span className="text-body font-semibold text-slate-800">
              {props.search
                ? `No ${props.kind} matches “${props.search}”`
                : `No ${props.kind} exists yet`}
            </span>
            {props.onCreate && (
              <Button
                type="button"
                variant="default"
                size="compact-sm"
                leftSection={<Plus size={12} strokeWidth={2.5} />}
                onClick={props.onCreate}
              >
                Create a {props.kind}
              </Button>
            )}
          </div>
        ) : (
          <ul role="listbox" className="flex flex-col gap-2">
            {items.map((item, index) => (
              <div
                key={item.metadata?.uid || item.metadata?.name}
                ref={(node) => {
                  rows.current[index] = node as HTMLLIElement | null;
                }}
              >
                <ResourceRow
                  item={item}
                  api={props.api}
                  kind={props.kind}
                  selected={props.selected.includes(item.metadata!.name)}
                  active={index === active}
                  onPick={() => props.onPick(item)}
                  onPeek={() => setPeek(item)}
                />
              </div>
            ))}
          </ul>
        )}
      </div>

      <footer className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-slate-200 bg-white px-4 py-3">
        {props.onCreate ? (
          <Button
            type="button"
            variant="default"
            size="compact-sm"
            leftSection={<Plus size={12} strokeWidth={2.5} />}
            onClick={props.onCreate}
          >
            Create a {props.kind}
          </Button>
        ) : (
          <span />
        )}

        <Button
          type="button"
          variant={props.multiple ? "filled" : "default"}
          color={props.multiple ? "ink" : undefined}
          size="compact-sm"
          leftSection={
            props.multiple ? <Check size={12} strokeWidth={2.5} /> : undefined
          }
          onClick={props.onClose}
        >
          {props.multiple ? "Done" : "Close"}
        </Button>
      </footer>

      {peek && (
        <ResourceYAML
          item={peek}
          readOnly
          hideTrigger
          opened
          onClose={() => setPeek(undefined)}
        />
      )}
    </Drawer>
  );
};

export default ResourcePicker;
