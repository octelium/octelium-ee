import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import { GetAuthenticationLogDataPointRequest } from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAuthenticationLog,
  refetchIntervalChart,
} from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import LineChart from "../Charts/LineChart";
import DurationPicker from "../DurationPicker";

const AuthenticationLogDataPoint = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  deviceRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  from?: Timestamp;
  to?: Timestamp;
  interval?: Duration;
}) => {
  let [interval, setInterval] = useState(props.interval);
  const qry = useQuery({
    queryKey: [
      "visibility",
      "getAuthenticationLogDataPoint",
      { ...props, interval },
    ],

    queryFn: async () => {
      const req = GetAuthenticationLogDataPointRequest.create({
        ...props,
        interval,
      });

      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogDataPoint(
          req
        );

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
        <div>
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

export default AuthenticationLogDataPoint;
