import * as React from "react";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Select } from "@mantine/core";
import dayjs from "dayjs";
import { LuTimer } from "react-icons/lu";
import { twMerge } from "tailwind-merge";

export const timeRangePickList = [
  {
    label: "1 Minute",
    value: "1 minute",
  },
  {
    label: "5 Minutes",
    value: "5 minute",
  },
  {
    label: "10 Minutes",
    value: "10 minute",
  },
  {
    label: "15 Minutes",
    value: "15 minute",
  },
  {
    label: "30 Minutes",
    value: "30 minute",
  },
  {
    label: "45 Minutes",
    value: "45 minute",
  },
  {
    label: "1 Hour",
    value: "60 minute",
  },
  {
    label: "2 Hours",
    value: "2 hour",
  },
  {
    label: "3 Hours",
    value: "3 hour",
  },
  {
    label: "4 Hours",
    value: "4 hour",
  },
  {
    label: "6 Hours",
    value: "6 hour",
  },
  {
    label: "8 Hours",
    value: "8 hour",
  },
  {
    label: "10 Hours",
    value: "10 hour",
  },
  {
    label: "12 Hours",
    value: "12 hour",
  },
  {
    label: "16 Hours",
    value: "16 hour",
  },
  {
    label: "20 Hours",
    value: "20 hour",
  },
  {
    label: "24 Hours",
    value: "24 hour",
  },
  {
    label: "36 Hours",
    value: "36 hour",
  },
  {
    label: "2 Days",
    value: "2 day",
  },
  {
    label: "3 Days",
    value: "3 day",
  },
  {
    label: "4 Days",
    value: "4 day",
  },
  {
    label: "5 Days",
    value: "5 day",
  },
  {
    label: "6 Days",
    value: "6 day",
  },
  {
    label: "1 Week",
    value: "7 day",
  },
  {
    label: "2 Weeks",
    value: "14 day",
  },
  {
    label: "3 Weeks",
    value: "21 day",
  },
  {
    label: "1 Month",
    value: "30 day",
  },
  {
    label: "2 Months",
    value: "60 day",
  },
  {
    label: "3 Months",
    value: "90 day",
  },
  {
    label: "4 Months",
    value: "120 day",
  },
  {
    label: "5 Months",
    value: "150 day",
  },
  {
    label: "6 Months",
    value: "180 day",
  },
  {
    label: "1 Year",
    value: "365 day",
  },
];

export const SelectFromTimestamp = (props: {
  onUpdate: (item: Timestamp) => void;
  isFuture?: boolean;
  label?: string;
  description?: string;
}) => {
  const [from, setFrom] = React.useState<string | undefined>(undefined);

  return (
    <Select
      className="ml-4"
      value={from}
      size={`xs`}
      label={props.label}
      description={props.description}
      searchable
      rightSection={<LuTimer />}
      data={timeRangePickList}
      clearable
      onChange={(v) => {
        if (!v) {
          setFrom(undefined);
          return;
        }

        const args = v.split(" ");
        const num = parseInt(args.at(0) ?? "0");
        if (num < 1) {
          setFrom(undefined);
          return;
        }

        setFrom(v);

        const ts = props.isFuture
          ? dayjs().add(num, (args.at(1) as "day") ?? "day")
          : dayjs().subtract(num, (args.at(1) as "day") ?? "day");
        props.onUpdate(Timestamp.fromDate(ts.utc().toDate()));
      }}
    />
  );
};

export const NoLogFound = () => {
  return (
    <div
      className={twMerge(
        "flex text-center items-center justify-center",
        "font-bold text-4xl text-gray-600",
        "my-16",
      )}
    >
      No Items Found
    </div>
  );
};
