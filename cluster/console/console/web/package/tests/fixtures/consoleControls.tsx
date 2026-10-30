import "@mantine/core/styles.layer.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { Duration } from "@/apis/metav1/metav1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import DurationPicker from "@/components/DurationPicker";
import TimestampPicker from "@/components/TimestampPicker";
import {
  useHasDirtyForm,
  useListKeys,
  useRegisterDirtyForm,
} from "@/utils/forms";

const params = new URLSearchParams(window.location.search);

const DirtyForm = (props: { name: string }) => {
  const [value, setValue] = React.useState("");
  useRegisterDirtyForm(value !== "");
  return (
    <div>
      <input aria-label={props.name} value={value} onChange={(event) => setValue(event.target.value)} />
      <button onClick={() => setValue("")}>Save {props.name}</button>
    </div>
  );
};

const Forms = () => {
  const dirty = useHasDirtyForm();
  const [first, setFirst] = React.useState(true);
  const [second, setSecond] = React.useState(true);
  return (
    <>
      <output aria-label="Dirty forms">{dirty ? "dirty" : "clean"}</output>
      <button onClick={() => setFirst((value) => !value)}>Toggle first form</button>
      <button onClick={() => setSecond((value) => !value)}>Toggle second form</button>
      {first && <DirtyForm name="First form" />}
      {second && <DirtyForm name="Second form" />}
    </>
  );
};

const ListKeys = () => {
  const keys = useListKeys();
  const [rows, setRows] = React.useState(["alpha", "beta", "gamma"]);
  return (
    <>
      <button onClick={() => {
        keys.removeAt("rows", 0);
        setRows(rows.slice(1));
      }}>Remove first row</button>
      <button onClick={() => {
        keys.move("rows", rows.length - 1, 0);
        setRows([rows[rows.length - 1], ...rows.slice(0, -1)]);
      }}>Move last row first</button>
      <button onClick={() => setRows([...rows, `new-${rows.length}`])}>Add row</button>
      {rows.map((row, index) => (
        <div key={keys.keyAt("rows", index)} data-row={row} data-key={keys.keyAt("rows", index)}>
          <input aria-label={row} defaultValue={row} />
        </div>
      ))}
      <output data-other-key={keys.keyAt("other", 0)}>Other list</output>
    </>
  );
};

const Pickers = () => {
  const [duration, setDuration] = React.useState<Duration>();
  const [timestamp, setTimestamp] = React.useState<Timestamp>();
  return (
    <>
      <div data-duration>
        <DurationPicker value={duration} onChange={setDuration} placeholder="Duration" />
        <output aria-label="Duration value">{JSON.stringify(duration ?? null)}</output>
        <button onClick={() => setDuration(Duration.create({ type: { oneofKind: "hours", hours: 2 } }))}>Set duration externally</button>
      </div>
      <div data-timestamp>
        <TimestampPicker value={timestamp} onChange={setTimestamp} placeholder="Timestamp" isFuture={params.get("future") === "true"} />
        <output aria-label="Timestamp value">{timestamp ? Timestamp.toDate(timestamp).toISOString() : "none"}</output>
        <button onClick={() => setTimestamp(Timestamp.fromDate(new Date("2026-10-30T12:15:00Z")))}>Set timestamp externally</button>
      </div>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider>
      <main data-ready>
        {params.get("scenario") === "keys" ? <ListKeys /> : params.get("scenario") === "pickers" ? <Pickers /> : <Forms />}
      </main>
    </MantineProvider>
  </React.StrictMode>,
);
