import {
  MultiSelect,
  Select,
  type MultiSelectProps,
  type SelectProps,
} from "@mantine/core";
import countries from "i18n-iso-countries";
import enLocale from "i18n-iso-countries/langs/en.json";

countries.registerLocale(enLocale);

const flags = import.meta.glob<string>(
  "/node_modules/flag-icons/flags/4x3/*.svg",
  { eager: true, query: "?url&no-inline", import: "default" },
);

const countryOptions = Object.entries(
  countries.getNames("en", { select: "official" }),
)
  .map(([value, label]) => ({ value, label }))
  .sort((a, b) => a.label.localeCompare(b.label));

export const CountryFlag = (props: { code?: string }) => {
  const code = props.code?.trim().toLowerCase();
  if (!code || code.length !== 2) return null;
  const src = flags[`/node_modules/flag-icons/flags/4x3/${code}.svg`];
  if (!src) return null;

  return (
    <img
      src={src}
      alt=""
      aria-hidden="true"
      loading="lazy"
      className="shrink-0 rounded-[2px]"
      style={{ width: "1.4rem", height: "1.05rem" }}
    />
  );
};

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
