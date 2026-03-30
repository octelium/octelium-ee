import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListSSHSessionRequest,
  ListSSHSessionResponse,
  SSHSession,
  SSHSession_State,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import * as React from "react";
import { match } from "ts-pattern";
import { NoLogFound, SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardService from "../Card/CardService";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import InfoItem from "../InfoItem";
import Label from "../Label";
import Paginator from "../Paginator";
import { ResourceListItem, ResourceListWrapper } from "../ResourceList";
import TimeAgo from "../TimeAgo";

import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

export const SSHSessionC = (props: { item: SSHSession }) => {
  const { item } = props;

  return (
    <div className="w-full py-2 border-b-slate-300">
      <div>
        <InfoItem title="ID">
          <CopyText value={item.id} />
        </InfoItem>
        <InfoItem title="Started">
          <TimeAgo rfc3339={item.startedAt} />
        </InfoItem>
        {item.endedAt && (
          <InfoItem title="Ended">
            <TimeAgo rfc3339={item.endedAt} />
            <span className="ml-3 text-gray-500">
              {`(~${dayjs(Timestamp.toDate(item.endedAt!)).from(
                Timestamp.toDate(item.startedAt!),
                true
              )})`}
            </span>
          </InfoItem>
        )}
        <InfoItem title="State">
          <Label>
            {match(item.state)
              .with(SSHSession_State.COMPLETED, () => "Completed")
              .with(SSHSession_State.ONGOING, () => "Ongoing")
              .otherwise(() => "")}
          </Label>
        </InfoItem>
      </div>
      {item.sessionRef && (
        <InfoItem title="Session">
          <div className="w-full flex items-center">
            <CardSession itemRef={item.sessionRef} />
          </div>
        </InfoItem>
      )}
      {item.serviceRef && (
        <InfoItem title="Service">
          <div className="w-full flex items-center">
            <CardService itemRef={item.serviceRef} />
          </div>
        </InfoItem>
      )}
    </div>
  );
};

export default (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
}) => {
  const [from, setFrom] = React.useState<Timestamp | undefined>(undefined);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listSSHSession",
      props.userRef?.uid,
      props.sessionRef?.uid,
      props.serviceRef?.uid,
      props.namespaceRef?.uid,
      props.regionRef?.uid,
      props.deviceRef?.uid,
      props.page,
      props.itemsPerPage,
      from?.nanos,
      from?.seconds,
    ],

    queryFn: async () => {
      if (isDev()) {
        return ListSSHSessionResponse.create({
          items: [
            SSHSession.create({
              id: "12345",
              startedAt: Timestamp.now(),
              endedAt: Timestamp.now(),
              sessionRef: props.sessionRef,
              serviceRef: props.serviceRef,
              state: SSHSession_State.COMPLETED,
            }),
          ],
        });
      }

      const { response } = await getClientVisibilityAccessLog().listSSHSession(
        ListSSHSessionRequest.create({
          userRef: props.userRef,
          sessionRef: props.sessionRef,
          serviceRef: props.serviceRef,
          namespaceRef: props.namespaceRef,
          deviceRef: props.deviceRef,
          common: {
            page: props.page ?? 0,
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
    <div className="w-full">
      {qry.data && qry.data.items.length === 0 && <NoLogFound />}
      {qry.data && qry.data.items.length > 0 && (
        <div>
          <div className="flex items-center mb-4">
            <div className="font-bold text-gray-800 mr-2">Filter Since</div>
            <SelectFromTimestamp
              onUpdate={(v) => {
                setFrom(v);
              }}
            />
          </div>

          <div className="w-full my-4">
            <Paginator meta={qry.data?.listResponseMeta} />
          </div>

          <ResourceListWrapper>
            {qry.data.items.map((x) => (
              <ResourceListItem key={x.id} path={`/visibility/ssh/${x.id}`}>
                <SSHSessionC key={x.id} item={x} />
              </ResourceListItem>
            ))}
          </ResourceListWrapper>

          <div className="w-full my-4">
            <Paginator meta={qry.data?.listResponseMeta} />
          </div>
        </div>
      )}
    </div>
  );
};
