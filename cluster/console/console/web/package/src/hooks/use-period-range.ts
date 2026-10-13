import {
  ALL_PERIODS,
  DEFAULT_PERIOD_MINUTES,
  isKnownPeriod,
} from "@/utils/visibility";
import { useDebouncedCallback } from "@mantine/hooks";
import * as React from "react";
import { useLocation, useSearchParams } from "react-router-dom";

export const usePeriodRange = (
  options: {
    defaultMinutes?: number;
    legacyParam?: string;
    maxMinutes?: number;
  } = {},
) => {
  const {
    defaultMinutes = DEFAULT_PERIOD_MINUTES,
    legacyParam,
    maxMinutes,
  } = options;
  const [searchParams, setSearchParams] = useSearchParams();
  const location = useLocation();
  const rangeParam = Number(searchParams.get("range"));
  const legacyMinutes =
    !searchParams.has("range") && legacyParam
      ? ALL_PERIODS.find(
          (option) =>
            option.label === searchParams.get(legacyParam) &&
            (maxMinutes === undefined || option.minutes <= maxMinutes),
        )?.minutes
      : undefined;
  const periodMinutes =
    isKnownPeriod(rangeParam) &&
    (maxMinutes === undefined || rangeParam <= maxMinutes)
      ? rangeParam
      : (legacyMinutes ?? defaultMinutes);

  const setPeriodMinutes = useDebouncedCallback((value: number) => {
    const next = new URLSearchParams(searchParams);
    if (legacyParam) next.delete(legacyParam);
    if (value === defaultMinutes) next.delete("range");
    else next.set("range", String(value));
    setSearchParams(next, { replace: true, preventScrollReset: true });
  }, 250);

  React.useEffect(() => {
    setPeriodMinutes.cancel();
  }, [location.key, setPeriodMinutes]);

  return { periodMinutes, setPeriodMinutes };
};
