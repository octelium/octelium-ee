import * as React from "react";

import { getResourceRef } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";

import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAuditLogRequest,
  ListAuditLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityAuditLog } from "@/utils/client";

import { AuditLog } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import Paginator from "@/components/Paginator";
import dayjs from "dayjs";
import Editor from "../AccessLogViewer/Editor";
import { InfoDetail } from "../AccessLogViewer/Old";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import InfoItem from "../InfoItem";
import { ResourceListLabel, ResourceListLabelWrap } from "../ResourceList";
import TimeAgo from "../TimeAgo";

const AuditLogInfo = (props: { accessLog: AuditLog }) => {
  const x = props.accessLog;
  if (!x.entry) {
    return <></>;
  }

  return (
    <div className="w-full">
      {x.entry.operation && (
        <InfoDetail label="Operation">{x.entry.operation}</InfoDetail>
      )}
    </div>
  );
};

export const AuditLogC = (props: { accessLog: AuditLog }) => {
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
      </div>
      <div className="w-full my-2">
        <InfoItem title="ID">
          <CopyText value={x.metadata!.id} />
        </InfoItem>
        {x.entry?.resourceRef && (
          <InfoItem title="Resource">
            <ResourceListLabel
              label={`Resource (${x.entry.resourceRef.kind})`}
              itemRef={getResourceRef(x.entry.resourceRef)}
            ></ResourceListLabel>
          </InfoItem>
        )}
        {x.entry?.sessionRef && (
          <InfoItem title="Session">
            <ResourceListLabelWrap>
              <CardSession itemRef={x.entry.sessionRef} />
            </ResourceListLabelWrap>
          </InfoItem>
        )}

        {x.entry?.operation && x.entry.operation.length > 0 && (
          <InfoItem title="Operation">
            <span>{x.entry.operation}</span>
          </InfoItem>
        )}
      </div>

      {/**
       
       <div className="w-full">
        <Collapse in={showDetails} transitionDuration={500}>
          <AuditLogInfo accessLog={x} />
        </Collapse>
      </div>
       **/}
    </div>
  );
};

const AuditLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  resourceRef?: ObjectReference;

  deviceRef?: ObjectReference;

  itemsPerPage?: number;
  page?: number;
}) => {
  const [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listAuditLog",
      props.userRef?.uid,
      props.sessionRef?.uid,
      props.serviceRef?.uid,
      props.resourceRef?.uid,
      props.deviceRef?.uid,
      props.page,
      from ? Timestamp.toDate(from).toISOString() : undefined,
    ],

    queryFn: async () => {
      if (isDev()) {
        const r = await getClientCore().listSession({});
        const sess = r.response.items.at(0);
        const rSvcs = await getClientCore().listService({});
        const svc = rSvcs.response.items.at(0);
        return ListAuditLogResponse.create({
          items: [
            AuditLog.create({
              kind: "AuditLog",
              metadata: {
                createdAt: Timestamp.now(),
                id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
                actorRef: getResourceRef(sess!),
              },
              entry: {
                sessionRef: getResourceRef(sess!),
                userRef: sess?.status?.userRef,
                deviceRef: sess?.status?.deviceRef,

                resourceRef: getResourceRef(sess!),
                operation: `octelium.api.core.v1.ListUser`,
              },
            }),
          ],
        });
      }

      const { response } = await getClientVisibilityAuditLog().listAuditLog(
        ListAuditLogRequest.create({
          common: {
            page: props.page ?? 0,
            itemsPerPage: props.itemsPerPage ? props.itemsPerPage : 100,
          },
          userRef: props.userRef,
          sessionRef: props.sessionRef,
          deviceRef: props.deviceRef,
          resourceRef: props.resourceRef,

          from,
        }),
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
              <AuditLogC key={x.metadata!.id} accessLog={x} />
            ))}
          </div>
        )}
      </div>

      <Paginator meta={qry.data?.listResponseMeta} />
    </div>
  );
};

export default AuditLogViewer;
