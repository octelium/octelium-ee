import * as React from "react";

import { getResourceRef, printServiceMode } from "@/utils/pb";
import { Button } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";

import {
  AccessLog,
  AccessLog_Entry_Common_Reason_Type,
  AccessLog_Entry_Common_Status,
  AccessLog_Entry_Info_DNS_Type,
  AccessLog_Entry_Info_HTTP,
  AccessLog_Entry_Info_HTTP_Request,
  AccessLog_Entry_Info_MySQL_Type,
  AccessLog_Entry_Info_Postgres_Type,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAccessLogRequest,
  ListAccessLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import Paginator from "@/components/Paginator";
import { isDev } from "@/utils";
import { getClientCore, getClientVisibilityAccessLog } from "@/utils/client";
import dayjs from "dayjs";
import { MdRefresh } from "react-icons/md";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import CardService from "../Card/CardService";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import InfoItem from "../InfoItem";
import Label from "../Label";
import AccessLogSummary from "../LogSummary/AccessLogSummary";
import { ResourceListLabel, ResourceListLabelWrap } from "../ResourceList";
import TimeAgo from "../TimeAgo";
import Editor from "./Editor";
import { SelectFromTimestamp } from "./utils";

export function convertBytes(
  bytes: number,
  options: { useBinaryUnits?: boolean; decimals?: number } = {},
): string {
  const { useBinaryUnits = false, decimals = 2 } = options;

  if (decimals < 0) {
    throw new Error(`Invalid decimals ${decimals}`);
  }

  const base = useBinaryUnits ? 1024 : 1000;
  const units = useBinaryUnits
    ? ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"]
    : ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

  const i = Math.floor(Math.log(bytes) / Math.log(base));

  return `${(bytes / Math.pow(base, i)).toFixed(decimals)} ${units[i]}`;
}

const InfoDetailHTTP = (props: { item: AccessLog_Entry_Info_HTTP }) => {
  const { item } = props;
  const x = item;

  return (
    <>
      {x.request && (
        <>
          {x.request.path && (
            <InfoDetail label="Request Path">{x.request.path}</InfoDetail>
          )}

          {x.request.method && (
            <InfoDetail label="Method">{x.request.method}</InfoDetail>
          )}

          {x.request.userAgent && (
            <InfoDetail label="User Agent">{x.request.userAgent}</InfoDetail>
          )}

          {x.request.referer && (
            <InfoDetail label="Referer">{x.request.referer}</InfoDetail>
          )}

          {x.request.bodyBytes > 0 && (
            <InfoDetail label="Body Size">
              {convertBytes(x.request.bodyBytes)}
            </InfoDetail>
          )}
        </>
      )}
      {x.response && (
        <>
          {x.response.code > 0 && (
            <InfoDetail label="Response Code">{x.response.code}</InfoDetail>
          )}

          {x.response.bodyBytes > 0 && (
            <InfoDetail label="Response Body Size">
              {convertBytes(x.response.bodyBytes)}
            </InfoDetail>
          )}
        </>
      )}
    </>
  );
};

export const InfoDetail = (props: {
  label: string;
  children?: React.ReactNode;
}) => {
  return (
    <div className="w-full flex items-center justify-center text-sm mb-1">
      <div className="font-bold text-black mr-2 min-w-[80px]">
        {props.label}
      </div>
      <div className="flex-1 w-full text-gray-700 font-bold">
        {props.children}
      </div>
    </div>
  );
};

const AccessLogInfo = (props: { accessLog: AccessLog }) => {
  const x = props.accessLog;
  if (!x.entry?.info) {
    return <></>;
  }

  return (
    <div className="w-full">
      {x.entry.info.type.oneofKind === `kubernetes` && (
        <>
          {x.entry.info.type.kubernetes.apiGroup && (
            <InfoDetail label="API Group">
              {x.entry.info.type.kubernetes.apiGroup}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.apiPrefix && (
            <InfoDetail label="API Prefix">
              {x.entry.info.type.kubernetes.apiPrefix}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.apiVersion && (
            <InfoDetail label="API Version">
              {x.entry.info.type.kubernetes.apiVersion}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.name && (
            <InfoDetail label="Name">
              {x.entry.info.type.kubernetes.name}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.namespace && (
            <InfoDetail label="Namespace">
              {x.entry.info.type.kubernetes.namespace}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.resource && (
            <InfoDetail label="Resource">
              {x.entry.info.type.kubernetes.resource}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.subresource && (
            <InfoDetail label="Sub-resource">
              {x.entry.info.type.kubernetes.subresource}
            </InfoDetail>
          )}
          {x.entry.info.type.kubernetes.verb && (
            <InfoDetail label="Verb">
              {x.entry.info.type.kubernetes.verb}
            </InfoDetail>
          )}

          {x.entry.info.type.kubernetes.http && (
            <InfoDetailHTTP item={x.entry.info.type.kubernetes.http} />
          )}
        </>
      )}

      {x.entry.info.type.oneofKind === `http` && (
        <InfoDetailHTTP item={x.entry!.info!.type.http} />
      )}

      {x.entry.info.type.oneofKind === `dns` && (
        <>
          {x.entry.info.type.dns && (
            <>
              {x.entry.info.type.dns.type && (
                <InfoDetail label="Type">
                  {AccessLog_Entry_Info_DNS_Type[x.entry.info.type.dns.type]}
                </InfoDetail>
              )}

              {x.entry.info.type.dns.name && (
                <InfoDetail label="Name">
                  {x.entry.info.type.dns.name}
                </InfoDetail>
              )}

              {x.entry.info.type.dns.answer && (
                <InfoDetail label="Answer">
                  {x.entry.info.type.dns.answer}
                </InfoDetail>
              )}

              {x.entry.info.type.dns.typeID && (
                <InfoDetail label="Type ID">
                  {x.entry.info.type.dns.typeID}
                </InfoDetail>
              )}
            </>
          )}
        </>
      )}

      {x.entry.info.type.oneofKind === `postgres` && (
        <>
          {x.entry.info.type.postgres && (
            <>
              {x.entry.info.type.postgres.type && (
                <InfoDetail label="Type">
                  {
                    AccessLog_Entry_Info_Postgres_Type[
                      x.entry.info.type.postgres.type
                    ]
                  }
                </InfoDetail>
              )}

              {x.entry.info.type.postgres.details.oneofKind === `query` && (
                <>
                  {x.entry.info.type.postgres.details.query &&
                    x.entry.info.type.postgres.details.query.query && (
                      <InfoDetail label="Query">
                        {x.entry.info.type.postgres.details.query.query}
                      </InfoDetail>
                    )}
                </>
              )}
            </>
          )}
        </>
      )}

      {x.entry.info.type.oneofKind === `mysql` && (
        <>
          {x.entry.info.type.mysql && (
            <>
              {x.entry.info.type.mysql.type && (
                <InfoDetail label="Type">
                  {
                    AccessLog_Entry_Info_MySQL_Type[
                      x.entry.info.type.mysql.type
                    ]
                  }
                </InfoDetail>
              )}

              {x.entry.info.type.mysql.details.oneofKind === `query` && (
                <>
                  {x.entry.info.type.mysql.details.query &&
                    x.entry.info.type.mysql.details.query.query && (
                      <InfoDetail label="Query">
                        {x.entry.info.type.mysql.details.query.query}
                      </InfoDetail>
                    )}
                </>
              )}
            </>
          )}
        </>
      )}

      {x.entry.info.type.oneofKind === `grpc` && (
        <>
          {x.entry.info.type.grpc && (
            <>
              {x.entry.info.type.grpc.serviceFullName && (
                <InfoDetail label="Service Full Name">
                  {x.entry.info.type.grpc.serviceFullName}
                </InfoDetail>
              )}

              {x.entry.info.type.grpc.service && (
                <InfoDetail label="Service">
                  {x.entry.info.type.grpc.service}
                </InfoDetail>
              )}
              {x.entry.info.type.grpc.package && (
                <InfoDetail label="Package">
                  {x.entry.info.type.grpc.package}
                </InfoDetail>
              )}
              {x.entry.info.type.grpc.method && (
                <InfoDetail label="Method">
                  {x.entry.info.type.grpc.method}
                </InfoDetail>
              )}

              {x.entry.info.type.grpc.http && (
                <InfoDetailHTTP item={x.entry.info.type.grpc.http} />
              )}
            </>
          )}
        </>
      )}
    </div>
  );
};

export const getPolicyReason = (arg?: AccessLog_Entry_Common_Reason_Type) => {
  return match(arg)
    .with(AccessLog_Entry_Common_Reason_Type.POLICY_MATCH, () => "Policy Match")
    .with(
      AccessLog_Entry_Common_Reason_Type.NO_POLICY_MATCH,
      () => "No Policy Match",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.USER_DEACTIVATED,
      () => "User Deactivated",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_NOT_ACTIVE,
      () => "Session Not Active",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_EXPIRED,
      () => "Session Expired",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.ACCESS_TOKEN_EXPIRED,
      () => "Access Token Expired",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.AUTHENTICATOR_AUTHENTICATION_REQUIRED,
      () => "Authenticator Authentication Required",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.AUTHENTICATOR_REGISTRATION_REQUIRED,
      () => "Authenticator Registration Required",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SCOPE_UNAUTHORIZED,
      () => "Unauthorized Scope",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.DEVICE_NOT_ACTIVE,
      () => "Device not Active",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_CLIENT_TYPE_INVALID,
      () => "Invalid Session type",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.DEVICE_LOCKED,
      () => "Locked Device",
    )
    .with(
      AccessLog_Entry_Common_Reason_Type.SESSION_LOCKED,
      () => "Locked Session",
    )
    .with(AccessLog_Entry_Common_Reason_Type.USER_LOCKED, () => "Locked User")
    .otherwise((typ) => (typ ? AccessLog_Entry_Common_Reason_Type[typ] : ""));
};

export const AccessLogC = (props: { accessLog: AccessLog }) => {
  const x = props.accessLog;

  let [showDetails, setShowDetails] = React.useState(false);
  if (!x.entry || !x.entry.common) {
    return <></>;
  }

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

        <span
          className={twMerge(
            `text-xs p-1 mx-1 text-white font-bold rounded-md shadow-xl`,
            x.entry?.common?.status === AccessLog_Entry_Common_Status.ALLOWED
              ? `bg-green-700`
              : `bg-red-600`,
          )}
        >
          {x.entry.common.status === AccessLog_Entry_Common_Status.ALLOWED
            ? `Allowed`
            : `Denied`}
        </span>

        {x.entry.common.reason &&
          x.entry.common.reason.type !==
            AccessLog_Entry_Common_Reason_Type.TYPE_UNKNOWN_REASON && (
            <Label>{getPolicyReason(x.entry.common.reason.type)}</Label>
          )}
        {x.entry.common.reason &&
          x.entry.common.reason?.details &&
          x.entry.common.reason.details.type.oneofKind === `policyMatch` &&
          x.entry.common.reason.details.type.policyMatch.type.oneofKind ===
            `policy` && (
            <ResourceListLabel
              label="Policy"
              itemRef={
                x.entry.common.reason.details.type.policyMatch.type.policy
                  .policyRef
              }
            ></ResourceListLabel>
          )}

        {x.entry?.common?.reason?.details?.type.oneofKind === `policyMatch` &&
          x.entry.common.reason.details.type.policyMatch.type.oneofKind ===
            `inlinePolicy` && (
            <ResourceListLabel
              label={`${x.entry.common.reason.details.type.policyMatch.type.inlinePolicy.resourceRef?.kind} InlinePolicy`}
              itemRef={
                x.entry.common.reason.details.type.policyMatch.type.inlinePolicy
                  .resourceRef
              }
            ></ResourceListLabel>
          )}
        {x.entry?.common?.mode && (
          <Label>{printServiceMode(x.entry.common.mode)}</Label>
        )}
      </div>

      <div className="w-full my-2">
        <InfoItem title="ID">
          <CopyText value={x.metadata!.id} />
        </InfoItem>
        {x.entry?.common?.sessionRef && (
          <InfoItem title="Session">
            <ResourceListLabelWrap>
              <CardSession itemRef={x.entry.common.sessionRef} />
            </ResourceListLabelWrap>
          </InfoItem>
        )}
        {x.entry?.common?.serviceRef && (
          <InfoItem title="Service">
            <ResourceListLabelWrap>
              <CardService itemRef={x.entry.common.serviceRef} />
            </ResourceListLabelWrap>
          </InfoItem>
        )}

        <AccessLogInfo accessLog={x} />
      </div>
      {/**
       <div className="w-full">
        <Collapse in={showDetails} transitionDuration={500}>
          <AccessLogInfo accessLog={x} />
        </Collapse>
      </div>
       **/}
    </div>
  );
};

export const getListAccessLogResponseTest = async () => {
  const r = await getClientCore().listSession({});
  const sess = r.response.items.at(0);
  const rSvcs = await getClientCore().listService({});
  const svc = rSvcs.response.items.at(0);
  return ListAccessLogResponse.create({
    items: [
      AccessLog.create({
        kind: "AccessLog",
        metadata: {
          createdAt: Timestamp.now(),
          id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
          actorRef: getResourceRef(sess!),
        },
        entry: {
          common: {
            sessionRef: getResourceRef(sess!),
            userRef: sess?.status?.userRef,
            deviceRef: sess?.status?.deviceRef,

            mode: Service_Spec_Mode.HTTP,
            serviceRef: getResourceRef(svc!),
            namespaceRef: svc?.status?.namespaceRef,

            status: AccessLog_Entry_Common_Status.ALLOWED,
            reason: {
              type: AccessLog_Entry_Common_Reason_Type.POLICY_MATCH,
            },
          },
          info: {
            type: {
              oneofKind: "http",
              http: {
                request: {
                  path: "/path/v1",
                } as AccessLog_Entry_Info_HTTP_Request,
              } as AccessLog_Entry_Info_HTTP,
            },
          },
        },
      }),
    ],
  });
};

const DoAccessLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  itemsPerPage?: number;
  from?: Timestamp;
}) => {
  let [page, setPage] = React.useState(0);

  const qry = useQuery({
    queryKey: ["visibility", "listAccessLog", { ...props }],

    queryFn: async () => {
      if (isDev()) {
        return getListAccessLogResponseTest();
      }

      const req = ListAccessLogRequest.create({
        userRef: props.userRef,
        sessionRef: props.sessionRef,
        serviceRef: props.serviceRef,
        namespaceRef: props.namespaceRef,
        regionRef: props.regionRef,
        policyRef: props.policyRef,
        deviceRef: props.deviceRef,
        common: {
          page,
          itemsPerPage: props.itemsPerPage ? props.itemsPerPage : 100,
        },
        from: props.from,
      });

      const { response } =
        await getClientVisibilityAccessLog().listAccessLog(req);
      return response;
    },
    refetchInterval: 60000,
  });

  /*
  React.useEffect(() => {
    qry.refetch();
  }, []);
  */

  return (
    <div>
      <div className="flex items-center mb-6">
        <Button
          size="compact-sm"
          variant="outline"
          className="ml-2 shadow-md"
          loading={qry.isLoading}
          onClick={() => {
            setPage(0);

            qry.refetch();
          }}
        >
          <MdRefresh />
        </Button>
      </div>
      <div className="ml-4 mt-4">
        {qry.data && qry.data.items.length > 0 && (
          <div>
            {qry.data.items.map((x) => (
              <AccessLogC key={x.metadata!.id} accessLog={x} />
            ))}
          </div>
        )}
      </div>

      {qry.data && qry.data.listResponseMeta && (
        <div className="mt-4 flex items-center justify-center">
          <Paginator meta={qry.data.listResponseMeta} />
        </div>
      )}
    </div>
  );
};

const AccessLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  policyRef?: ObjectReference;
  itemsPerPage?: number;
  page?: number;
}) => {
  let [from, setFrom] = React.useState<Timestamp>(
    Timestamp.fromDate(dayjs().subtract(6, "hour").toDate()),
  );

  return (
    <div className="w-full">
      <div className="flex items-center mb-4">
        <div className="font-bold text-gray-800 mr-2">Filter Since</div>
        <SelectFromTimestamp
          onUpdate={(v) => {
            setFrom(v);
          }}
        />
      </div>

      <div className="w-full my-8">
        <AccessLogSummary
          userRef={props.userRef}
          sessionRef={props.sessionRef}
          serviceRef={props.serviceRef}
          namespaceRef={props.namespaceRef}
          regionRef={props.regionRef}
          policyRef={props.policyRef}
          deviceRef={props.deviceRef}
          from={from}
        />
      </div>
      <DoAccessLogViewer
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        serviceRef={props.serviceRef}
        namespaceRef={props.namespaceRef}
        regionRef={props.regionRef}
        policyRef={props.policyRef}
        deviceRef={props.deviceRef}
        from={from}
      />
    </div>
  );
};

export default AccessLogViewer;
