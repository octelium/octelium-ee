import {
  MultiSelect,
  Select,
  type MultiSelectProps,
  type SelectProps,
} from "@mantine/core";
import countries from "i18n-iso-countries";
import enLocale from "i18n-iso-countries/langs/en.json";

import "flag-icons/css/flag-icons.min.css";

countries.registerLocale(enLocale);

const countryOptions = Object.entries(
  countries.getNames("en", { select: "official" }),
)
  .map(([value, label]) => ({ value, label }))
  .sort((a, b) => a.label.localeCompare(b.label));

export const CountryFlag = (props: { code?: string }) => {
  const code = props.code?.trim().toLowerCase();
  if (!code || code.length !== 2) return null;

  return (
    <span
      aria-hidden="true"
      className={`fi fi-${code} shrink-0 rounded-[2px]`}
      style={{ fontSize: "1.05rem" }}
    />
  );
};

const renderOption: SelectProps["renderOption"] = ({ option }) => (
  <span className="flex items-center gap-2">
    <CountryFlag code={option.value} />
    <span className="min-w-0 truncate">{option.label}</span>
    <span className="ml-auto text-[0.65rem] font-bold uppercase text-slate-400">
      {option.value}
    </span>
  </span>
);

const renderMultiOption: MultiSelectProps["renderOption"] = ({ option }) => (
  <span className="flex items-center gap-2">
    <CountryFlag code={option.value} />
    <span className="min-w-0 truncate">{option.label}</span>
    <span className="ml-auto text-[0.65rem] font-bold uppercase text-slate-400">
      {option.value}
    </span>
  </span>
);

const SelectCountry = (props: {
  val?: string;
  values?: string[];
  multiple?: boolean;
  onUpdate: (value?: string | string[]) => void;
}) => {
  if (props.multiple) {
    return (
      <MultiSelect
        label="Allowed countries"
        description="Add one or more ISO 3166-1 alpha-2 country codes."
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
