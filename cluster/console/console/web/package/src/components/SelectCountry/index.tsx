/*
import { Group, Select, SelectProps, Text } from "@mantine/core";
import countries from "i18n-iso-countries";
import enLocale from "i18n-iso-countries/langs/en.json";
import { useMemo, useState } from "react";

import "flag-icons/css/flag-icons.min.css";

countries.registerLocale(enLocale);

const SelectCountry = (props: {
  val?: string;
  onUpdate: (val?: string) => void;
}) => {
  const [selectedCountry, setSelectedCountry] = useState<string | null>(
    props.val ?? null,
  );

  const countriesData = useMemo(() => {
    const countryObj = countries.getNames("en", { select: "official" });

    return Object.entries(countryObj).map(([code, name]) => ({
      value: code,
      label: name,
    }));
  }, []);

  const renderSelectOption: SelectProps["renderOption"] = ({ option }) => (
    <Group gap="sm">
      <span
        className={`fi fi-${option.value.toLowerCase()}`}
        style={{ fontSize: "1.2rem", borderRadius: "2px" }}
      />
      <Text size="sm">{option.label}</Text>
    </Group>
  );

  return (
    <Select
      label="Country"
      placeholder="Select a country"
      data={countriesData}
      value={selectedCountry}
      onChange={(v) => {
        setSelectedCountry(v);
        props.onUpdate(v ?? undefined);
      }}
      searchable
      renderOption={renderSelectOption}
      leftSection={
        selectedCountry ? (
          <span
            className={`fi fi-${selectedCountry.toLowerCase()}`}
            style={{ fontSize: "1.2rem", borderRadius: "2px" }}
          />
        ) : null
      }
    />
  );
};
*/

const SelectCountry = (props: {
  val?: string;
  onUpdate: (val?: string) => void;
}) => {
  return <></>;
};

export default SelectCountry;
