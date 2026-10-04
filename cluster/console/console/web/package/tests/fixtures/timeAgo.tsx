import "@mantine/core/styles.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import TimeAgo from "@/components/TimeAgo";
import themeMantine from "@/utils/theme/mantine";

const timestamp = Timestamp.fromDate(new Date("2026-10-04T10:00:00Z"));

const Fixture = () => {
  const [version, setVersion] = React.useState(0);

  return (
    <MantineProvider theme={themeMantine}>
      <div style={{ padding: 40 }}>
        <button onClick={() => setVersion((value) => value + 1)}>Refresh</button>
        <div style={{ height: 300, overflow: "auto", marginTop: 16 }}>
          {Array.from({ length: 100 }, (_, index) => (
            <div key={index} data-row={index} style={{ height: 32 }}>
              <span>Row {index} v{version} </span>
              <TimeAgo rfc3339={timestamp} />
            </div>
          ))}
        </div>
      </div>
    </MantineProvider>
  );
};

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Fixture />
  </React.StrictMode>,
);
