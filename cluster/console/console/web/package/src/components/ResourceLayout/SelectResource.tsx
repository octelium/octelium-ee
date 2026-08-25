import {
  API,
  getResourcePathFromAPIKind,
  printResourceNameWithDisplay,
  Resource,
  ResourceName,
} from "@/utils/pb";
import { getResourceComponentInfo } from "@/pages/utils/resourceRegistry";
import {
  ActionIcon,
  Alert,
  Button,
  Loader,
  Select,
  Tooltip,
} from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { FileText, Plus, Search } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import TimeAgo from "../TimeAgo";
import ResourceYAML from "../ResourceYAML";
import { listResourcesForSelect } from "./listResourcesForSelect";

const SelectResource = (props: {
  api: string;
  kind: string;
  defaultValue?: string;
  description?: string;
  required?: boolean;
  label?: string;
  labelDefault?: boolean;
  clearable?: boolean;
  onChange: (item?: Resource) => void;
}) => {
  const { api, kind } = props;
  const [yamlItem, setYamlItem] = React.useState<Resource>();
  const location = useLocation();
  const navigate = useNavigate();
  const handledCreatedName = React.useRef<string | undefined>(undefined);
  const resourceComponentInfo = getResourceComponentInfo(
    api as API,
    kind as ResourceName,
  );
  const canCreate = !!resourceComponentInfo && !resourceComponentInfo.unCreatable;

  const { isLoading, isError, error, data, refetch } = useQuery({
    queryKey: ["listSelectComponent", api, kind],
    queryFn: () => listResourcesForSelect(api, kind),
  });

  const label = props.labelDefault ? `Select ${kind}` : props.label;
  const rscList = React.useMemo(
    () =>
      (data ?? []).map((item) => ({
        value: item.metadata!.name,
        label: printResourceNameWithDisplay(item),
      })),
    [data],
  );
  const resourcesByName = React.useMemo(
    () =>
      new Map((data ?? []).map((item) => [item.metadata!.name, item])),
    [data],
  );

  const openCreate = React.useCallback(() => {
    const returnState =
      location.state && typeof location.state === "object"
        ? location.state
        : undefined;
    navigate(
      `/${api}/${getResourcePathFromAPIKind({
        api: api as API,
        kind: kind as ResourceName,
      })}/create`,
      {
        state: {
          createInDrawer: true,
          returnTo: location.pathname,
          returnState,
        },
      },
    );
  }, [api, kind, location.pathname, location.state, navigate]);

  const createButton = (
    <div className="flex justify-end">
      <Button
        type="button"
        size="compact-sm"
        variant="subtle"
        color="dark"
        leftSection={<Plus size={14} strokeWidth={2.2} />}
        onClick={openCreate}
      >
        Create a {kind}
      </Button>
    </div>
  );

  const createdResourceName =
    location.state && typeof location.state === "object"
      ? (location.state as { createdResourceName?: string })
          .createdResourceName
      : undefined;

  React.useEffect(() => {
    if (
      !createdResourceName ||
      handledCreatedName.current === createdResourceName
    ) {
      return;
    }
    const created = resourcesByName.get(createdResourceName);
    if (!created) return;

    handledCreatedName.current = createdResourceName;
    props.onChange(created);
    const nextState =
      location.state && typeof location.state === "object"
        ? { ...(location.state as Record<string, unknown>) }
        : {};
    delete nextState.createdResourceName;
    navigate(location.pathname, {
      replace: true,
      preventScrollReset: true,
      state: Object.keys(nextState).length > 0 ? nextState : undefined,
    });
  }, [createdResourceName, location.pathname, location.state, navigate, props.onChange, resourcesByName]);

  if (isLoading) {
    return (
      <div className="space-y-1.5">
        <Select
          label={label}
          required={props.required}
          description={props.description}
          data={[]}
          disabled
          placeholder="Loading…"
          rightSection={<Loader size={15} color="gray" />}
        />
        {canCreate && createButton}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-1.5">
        <Alert color="red" title={`Could not load ${kind}s`}>
          <div className="flex flex-col gap-2">
            <span className="text-xs">{error.message}</span>
            <Button size="compact-xs" variant="outline" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        </Alert>
        {canCreate && createButton}
      </div>
    );
  }

  if (!data) {
    return (
      <div className="space-y-1.5">
        <Alert color="red" title={`Could not load ${kind}s`}>
          <Button size="compact-xs" variant="outline" onClick={() => refetch()}>
            Retry
          </Button>
        </Alert>
        {canCreate && createButton}
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      <Select
        label={label}
        required={props.required}
        description={props.description}
        clearable={props.clearable}
        searchable
        data={rscList}
        value={props.defaultValue ?? null}
        disabled={rscList.length === 0}
        leftSection={<Search size={14} strokeWidth={2.1} />}
        maxDropdownHeight={390}
        placeholder={
          rscList.length === 0
            ? `No ${kind} resources found`
            : `Search and select ${kind}…`
        }
        nothingFoundMessage={`No ${kind} resources match your search`}
        comboboxProps={{
          shadow: "md",
          transitionProps: { transition: "pop", duration: 180 },
        }}
        styles={{
          dropdown: {
            padding: 6,
            borderColor: "#e2e8f0",
            borderRadius: 12,
          },
          option: {
            padding: 6,
            borderRadius: 9,
          },
        }}
        renderOption={({ option }) => {
          const item = resourcesByName.get(option.value);
          if (!item) return null;
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
                  <span className="truncate text-[0.78rem] font-bold text-slate-800">
                    {metadata.name}
                  </span>
                  {metadata.displayName && (
                    <span className="truncate text-[0.69rem] font-semibold text-slate-500">
                      {metadata.displayName}
                    </span>
                  )}
                </div>
                {metadata.description && (
                  <span className="mt-0.5 truncate text-[0.67rem] font-medium text-slate-400">
                    {metadata.description}
                  </span>
                )}
                <span className="mt-1 text-[0.64rem] font-semibold text-slate-400">
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
        onChange={(value) => {
          if (!value) {
            props.onChange();
            return;
          }
          props.onChange(resourcesByName.get(value));
        }}
      />

      {canCreate && createButton}

      {yamlItem && (
        <ResourceYAML
          item={yamlItem}
          readOnly
          hideTrigger
          opened
          onClose={() => setYamlItem(undefined)}
        />
      )}
    </div>
  );
};

export default SelectResource;
