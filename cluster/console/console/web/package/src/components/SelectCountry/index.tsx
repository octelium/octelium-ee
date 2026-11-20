import {
  MultiSelect,
  Select,
  type MultiSelectProps,
  type SelectProps,
} from "@mantine/core";
import countries from "i18n-iso-countries";
import enLocale from "i18n-iso-countries/langs/en.json";
import { CountryFlag } from "./CountryFlag";

export { CountryFlag };

countries.registerLocale(enLocale);

const countryOptions = Object.entries(
  countries.getNames("en", { select: "official" }),
)
  .map(([value, label]) => ({ value, label }))
  .sort((a, b) => a.label.localeCompare(b.label));

const renderOption: SelectProps["renderOption"] = ({ option }) => (
  <span className="flex items-center gap-2">
    <CountryFlag code={option.value} />
    <span className="min-w-0 truncate">{option.label}</span>
    <span className="ml-auto text-micro font-semibold uppercase text-slate-500">
      {option.value}
    </span>
  </span>
);

const renderMultiOption: MultiSelectProps["renderOption"] = ({ option }) => (
  <span className="flex items-center gap-2">
    <CountryFlag code={option.value} />
    <span className="min-w-0 truncate">{option.label}</span>
    <span className="ml-auto text-micro font-semibold uppercase text-slate-500">
      {option.value}
    </span>
  </span>
);

const SelectCountry = (props: {
  val?: string;
  values?: string[];
  multiple?: boolean;
  label?: string;
  description?: string;
  onUpdate: (value?: string | string[]) => void;
}) => {
  if (props.multiple) {
    return (
      <MultiSelect
        label={props.label ?? "Allowed countries"}
        description={
          props.label
            ? props.description
            : "Add one or more ISO 3166-1 alpha-2 country codes."
        }
        placeholder="Search and add countries"
        searchable
        clearable
        data={countryOptions}
        value={props.values ?? []}
        renderOption={renderMultiOption}
        onChange={(value) => props.onUpdate(value)}
        maxDropdownHeight={320}
      />
    );
  }

  return (
    <Select
      label="Country"
      description="Select an ISO 3166-1 alpha-2 country code."
      placeholder="Search for a country"
      searchable
      clearable
      data={countryOptions}
      value={props.val ?? null}
      renderOption={renderOption}
      leftSection={<CountryFlag code={props.val} />}
      onChange={(value) => props.onUpdate(value ?? undefined)}
      maxDropdownHeight={320}
    />
  );
};

export default SelectCountry;

export const CountryFilter = (props: {
  value?: string;
  onChange: (value?: string) => void;
  className?: string;
}) => (
  <Select
    size="xs"
    className={props.className ?? "w-52"}
    placeholder="Any country"
    aria-label="Country"
    searchable
    clearable
    data={countryOptions}
    value={props.value ?? null}
    renderOption={renderOption}
    leftSection={<CountryFlag code={props.value} />}
    onChange={(value) => props.onChange(value ?? undefined)}
    maxDropdownHeight={320}
    comboboxProps={{ width: 280, position: "bottom-start" }}
  />
);
