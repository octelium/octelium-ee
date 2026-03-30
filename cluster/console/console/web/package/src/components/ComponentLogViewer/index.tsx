import * as React from "react";

import { getResourceRef } from "@/utils/pb";
import { Collapse } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";

import {
  ListComponentLogRequest,
  ListComponentLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityComponentLog } from "@/utils/client";

import { ComponentLog, ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import Paginator from "@/components/Paginator";
import dayjs from "dayjs";
import Editor from "../AccessLogViewer/Editor";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CopyText from "../CopyText";
import InfoItem from "../InfoItem";
import Label from "../Label";
import TimeAgo from "../TimeAgo";

const AuditLogInfo = (props: { accessLog: ComponentLog }) => {
  const x = props.accessLog;
  if (!x.entry) {
    return <></>;
  }

  return <div className="w-full"></div>;
};

export const ComponentLogC = (props: { accessLog: ComponentLog }) => {
  const x = props.accessLog;

  let [showDetails, setShowDetails] = React.useState(false);

  return (
    <div
      onMouseEnter={() => {
        setShowDetails(true);
      }}
      onMouseLeave={() => {
        setShowDetails(false);
      }}
      className="w-full mt-2 mb-2 border-b-[1px] border-b-slate-300"
    >
      <div className="w-full flex items-center">
        <div>
          <Editor item={props.accessLog} />
        </div>
        <span className="font-bold text-slate-500 text-xs mx-2">
          <TimeAgo rfc3339={x.metadata!.createdAt} />
        </span>

        {x.entry?.component && (
          <>
            <Label>{`${x.entry!.component.namespace}/${
              x.entry!.component.type
            }`}</Label>
            <span className="ml-2">
              <Label>{`${x.entry!.component.uid}`}</Label>
            </span>
          </>
        )}
        {x.entry?.level && x.entry.level > 0 && (
          <Label>{ComponentLog_Entry_Level[x.entry!.level]}</Label>
        )}
      </div>
      <div className="w-full my-2">
        <InfoItem title="ID">
          <CopyText value={x.metadata!.id} />
        </InfoItem>

        {x.entry?.message && x.entry.message.length > 0 && (
          <InfoItem title="Message">
            <span>{x.entry.message}</span>
          </InfoItem>
        )}
      </div>

      <div className="w-full">
        <Collapse in={showDetails} transitionDuration={500}>
          <AuditLogInfo accessLog={x} />
        </Collapse>
      </div>
    </div>
  );
};

const ComponentLogViewer = (props: { itemsPerPage?: number }) => {
  let [page, setPage] = React.useState(0);
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate())
  );

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listComponentLog",

      page,
      from ? Timestamp.toDate(from).toISOString() : undefined,
    ],

    queryFn: async () => {
      if (isDev()) {
        const r = await getClientCore().listSession({});
        const sess = r.response.items.at(0);
        const rSvcs = await getClientCore().listService({});
        const svc = rSvcs.response.items.at(0);
        return ListComponentLogResponse.create({
          items: [
            ComponentLog.create({
              kind: "ComponentLog",
              metadata: {
                createdAt: Timestamp.now(),
                id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
                actorRef: getResourceRef(sess!),
              },
              entry: {
                component: {
                  namespace: "octelium",
                  type: "nocturne",
                },
                level: ComponentLog_Entry_Level.INFO,
                message: "Component is starting...",
              },
            }),
          ],
        });
      }

      const { response } =
        await getClientVisibilityComponentLog().listComponentLog(
          ListComponentLogRequest.create({
            common: {
              page,
              itemsPerPage: props.itemsPerPage ? props.itemsPerPage : 100,
            },
            from,
          })
        );
      return response;
    },
    refetchInterval: 60000,
  });

  return (
    <div>
      <div className="flex items-center mb-4">
        <div className="font-bold text-gray-800 mr-2">Filter Since</div>
        <SelectFromTimestamp
          onUpdate={(v) => {
            setFrom(v);
          }}
        />
      </div>
      <div className="ml-4 mt-4">
        {qry.isSuccess && qry.data && qry.data.items.length > 0 && (
          <div>
            {qry.data.items.map((x) => (
              <ComponentLogC key={x.metadata!.id} accessLog={x} />
            ))}
          </div>
        )}
      </div>

      <Paginator meta={qry.data?.listResponseMeta} />
    </div>
  );
};

export default ComponentLogViewer;
