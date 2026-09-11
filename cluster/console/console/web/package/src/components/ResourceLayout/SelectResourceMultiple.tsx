import {
  printResourceNameWithDisplay,
  Resource,
} from "@/utils/pb";
import {
  ActionIcon,
  Alert,
  Button,
  Loader,
  MultiSelect,
  Tooltip,
} from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { FileText, Search } from "lucide-react";
import * as React from "react";
import TimeAgo from "../TimeAgo";
import ResourceYAML from "../ResourceYAML";
import { listResourcesForSelect } from "./listResourcesForSelect";

const SelectResourceMultiple = (props: {
  api: string;
  kind: string;
  defaultValue?: string[];
  description?: string;
  required?: boolean;
  label?: string;
  labelDefault?: boolean;
  clearable?: boolean;
  onChange: (itemList?: Resource[]) => void;
}) => {
  const { api, kind } = props;
  const [yamlItem, setYamlItem] = React.useState<Resource>();

  const [search, setSearch] = React.useState("");
  const [debouncedSearch, setDebouncedSearch] = React.useState("");

  React.useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedSearch(search), 250);
    return () => window.clearTimeout(timeout);
  }, [search]);

  const { isLoading, isError, error, data, refetch } = useQuery({
    queryKey: ["listSelectComponent", api, kind, debouncedSearch],
    queryFn: () => listResourcesForSelect(api, kind, debouncedSearch),
    placeholderData: keepPreviousData,
  });

  const label = props.labelDefault ? `Select ${kind}` : props.label;
  const rscList = React.useMemo(() => {
    const options = (data ?? []).map((item) => ({
      value: item.metadata!.name,
      label: printResourceNameWithDisplay(item),
    }));
    for (const selected of props.defaultValue ?? []) {
      if (!options.some((option) => option.value === selected)) {
        options.unshift({ value: selected, label: selected });
      }
    }
    return options;
  }, [data, props.defaultValue]);
  const resourcesByName = React.useMemo(
    () =>
      new Map((data ?? []).map((item) => [item.metadata!.name, item])),
    [data],
  );

  if (isLoading && !data) {
    return (
      <MultiSelect
        label={label}
        required={props.required}
        description={props.description}
        data={[]}
        disabled
        placeholder="Loading…"
        rightSection={<Loader size={15} color="gray" />}
      />
    );
  }

  if (isError) {
    return (
      <Alert color="red" title={`Could not load ${kind}s`}>
        <div className="flex flex-col gap-2">
          <span className="text-xs">{error.message}</span>
          <Button size="compact-xs" variant="outline" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </Alert>
    );
  }

  if (!data) {
    return (
      <Alert color="red" title={`Could not load ${kind}s`}>
        <Button size="compact-xs" variant="outline" onClick={() => refetch()}>
          Retry
        </Button>
      </Alert>
    );
  }

  return (
    <>
      <MultiSelect
        label={label}
        required={props.required}
        description={props.description}
        clearable={props.clearable}
        searchable
        searchValue={search}
        onSearchChange={setSearch}
        filter={({ options }) => options}
        data={rscList}
        disabled={rscList.length === 0 && debouncedSearch.length === 0}
        value={props.defaultValue ?? []}
        leftSection={<Search size={14} strokeWidth={2.1} />}
        maxDropdownHeight={390}
        placeholder={
          rscList.length === 0
            ? `No ${kind} resources found`
            : `Search and select ${kind} resources…`
        }
        nothingFoundMessage={`No ${kind} resources match your search`}
        comboboxProps={{
          shadow: "md",
          transitionProps: { transition: "pop", duration: 180 },
        }}
        styles={{
          dropdown: {
            padding: 6,
            borderColor: "var(--color-slate-200)",
            borderRadius: 12,
          },
          option: {
            padding: 6,
            borderRadius: 9,
          },
        }}
        renderOption={({ option }) => {
          const item = resourcesByName.get(option.value);
          if (!item) {
            return (
              <span className="text-body font-normal text-slate-700">
                {option.label}
              </span>
            );
          }
          const metadata = item.metadata!;

          return (
            <div className="flex h-[58px] min-w-0 flex-1 items-center gap-2.5">
              {metadata.picURL ? (
                <img
                  src={metadata.picURL}
                  alt={metadata.displayName || metadata.name}
                  loading="lazy"
                  className="h-9 w-9 shrink-0 rounded-lg border border-slate-200 bg-white object-cover shadow-sm"
                />
              ) : null}

              <div className="flex min-w-0 flex-1 flex-col justify-center">
                <div className="flex min-w-0 items-baseline gap-2">
                  <span className="truncate text-body font-semibold text-slate-800">
                    {metadata.name}
                  </span>
                  {metadata.displayName && (
                    <span className="truncate text-xs font-normal text-slate-500">
                      {metadata.displayName}
                    </span>
                  )}
                </div>
                {metadata.description && (
                  <span className="mt-0.5 truncate text-xs font-medium text-slate-500">
                    {metadata.description}
                  </span>
                )}
                <span className="mt-1 text-micro font-normal text-slate-500">
                  Created <TimeAgo rfc3339={metadata.createdAt} />
                </span>
              </div>

              <Tooltip label="View YAML" withArrow>
                <ActionIcon
                  type="button"
                  variant="subtle"
                  color="gray"
                  size="sm"
                  aria-label={`View YAML for ${metadata.name}`}
                  onMouseDown={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                  }}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    setYamlItem(item);
                  }}
                >
                  <FileText size={14} strokeWidth={2.1} />
                </ActionIcon>
              </Tooltip>
            </div>
          );
        }}
        onChange={(values) => {
          if (values.length === 0) {
            props.onChange(undefined);
            return;
          }
          props.onChange(
            values
              .map((value) => resourcesByName.get(value))
              .filter((item): item is Resource => item !== undefined),
          );
        }}
      />

      {yamlItem && (
        <ResourceYAML
          item={yamlItem}
          readOnly
          hideTrigger
          opened
          onClose={() => setYamlItem(undefined)}
        />
      )}
    </>
  );
};

export default SelectResourceMultiple;
