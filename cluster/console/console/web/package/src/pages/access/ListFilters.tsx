import { useListParams } from "@/components/ResourceLayout/ResourceListToolbar";
import { Button, Checkbox, Popover, Select } from "@mantine/core";
import { ListFilter } from "lucide-react";
import { getAccessListFilters } from "./listFilters";

const AccessListFilters = (props: { kind: string }) => {
  const { searchParams, setParam, removeParam } = useListParams();
  const filters = getAccessListFilters(props.kind);
  const booleans = filters.filter((filter) => !filter.options);
  const active = booleans.filter(
    (filter) => searchParams.get(filter.key) === "true",
  ).length;

  if (filters.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {filters
        .filter((filter) => filter.options)
        .map((filter) => (
          <Select
            key={filter.key}
            w={156}
            clearable
            aria-label={filter.label}
            placeholder={filter.label}
            value={searchParams.get(filter.key)}
            data={filter.options}
            onChange={(value) =>
              value ? setParam(filter.key, value) : removeParam(filter.key)
            }
          />
        ))}
      {booleans.length > 0 && (
        <Popover position="bottom-end" width={300} withinPortal>
          <Popover.Target>
            <Button
              variant="default"
              size="xs"
              leftSection={<ListFilter size={13} />}
            >
              More filters{active > 0 && ` (${active})`}
            </Button>
          </Popover.Target>
          <Popover.Dropdown>
            <div className="flex flex-col gap-3">
              {booleans.map((filter) => (
                <Checkbox
                  key={filter.key}
                  size="xs"
                  label={filter.label}
                  checked={searchParams.get(filter.key) === "true"}
                  onChange={(event) =>
                    event.currentTarget.checked
                      ? setParam(filter.key, "true")
                      : removeParam(filter.key)
                  }
                />
              ))}
            </div>
          </Popover.Dropdown>
        </Popover>
      )}
    </div>
  );
};

export default AccessListFilters;
