import React from "react";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

import { Tooltip } from "@mantine/core";

const TimeAgo = (props: { rfc3339?: Timestamp }) => {
  const t = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;
  const [time, setTime] = React.useState(t ? dayjs(t).fromNow() : "—");

  React.useEffect(() => {
    if (!t) {
      setTime("—");
      return;
    }

    setTime(dayjs(t).fromNow());

    const interval = setInterval(() => setTime(dayjs(t).fromNow()), 10000);
    return () => {
      clearInterval(interval);
    };
  }, [t?.getTime()]);

  if (!t) return <span>—</span>;

  return (
    <Tooltip
      label={
        <p className="font-bold shadow-md text-xs rounded-sm">
          {dayjs(t).local().format("hh:mm:ss A, ddd MMM D, YYYY")}
        </p>
      }
      transitionProps={{
        transition: "fade",
        duration: 340,
      }}
    >
      <span>{time}</span>
    </Tooltip>
  );
};

export const TimeRemaining = (props: { rfc3339?: Timestamp }) => {
  const target = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;
  const [label, setLabel] = React.useState("—");

  React.useEffect(() => {
    const update = () => {
      if (!target) {
        setLabel("—");
        return;
      }
      const diff = target.getTime() - Date.now();
      if (diff <= 0) {
        setLabel("Ended");
        return;
      }
      const minutes = Math.floor(diff / 60000);
      const days = Math.floor(minutes / 1440);
      const hours = Math.floor((minutes % 1440) / 60);
      const remainingMinutes = minutes % 60;
      if (days > 0) setLabel(`${days}d ${hours}h remaining`);
      else if (hours > 0) setLabel(`${hours}h ${remainingMinutes}m remaining`);
      else setLabel(`${Math.max(1, remainingMinutes)}m remaining`);
    };

    update();
    const interval = window.setInterval(update, 10000);
    return () => window.clearInterval(interval);
  }, [target?.getTime()]);

  return <span className={label === "Ended" ? "text-slate-400" : "text-emerald-600"}>{label}</span>;
};

export default TimeAgo;
