import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import { GetAccessLogDataPointRequest } from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAccessLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import LineChart from "../Charts/LineChart";
import DurationPicker from "../DurationPicker";

const AccessLogDataPoint = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  from?: Timestamp;
  to?: Timestamp;
  interval?: Duration;
}) => {
  let [interval, setInterval] = useState(props.interval);
  const qry = useQuery({
    queryKey: ["visibility", "getAccessLogDataPoint", { ...props, interval }],

    queryFn: async () => {
      const req = GetAccessLogDataPointRequest.create({
        ...props,
        interval,
      });

      const { response } =
        await getClientVisibilityAccessLog().getAccessLogDataPoint(req);

      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  if (!qry.data || !qry.isSuccess) {
    return <></>;
  }

  return (
    <div>
      {qry.data && qry.data.datapoints && (
        <div className="w-full">
          <div className="w-full flex items-end justify-end">
            <div className="flex-1"></div>
            <div>
              <DurationPicker
                value={interval}
                onChange={(v) => {
                  setInterval(v);
                }}
              />
            </div>
          </div>
          <LineChart
            points={qry.data.datapoints.map((x) => ({
              ts: x.timestamp!,
              value: x.count,
            }))}
          />
        </div>
      )}
    </div>
  );
};

export default AccessLogDataPoint;
