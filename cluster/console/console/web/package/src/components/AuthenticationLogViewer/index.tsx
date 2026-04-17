import * as React from "react";

import { MdRefresh } from "react-icons/md";

import { getResourceRef } from "@/utils/pb";
import { Button } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";

import { ObjectReference } from "@/apis/metav1/metav1";
import {
  ListAuthenticationLogRequest,
  ListAuthenticationLogResponse,
} from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import {
  getClientCore,
  getClientVisibilityAuthenticationLog,
} from "@/utils/client";

import {
  Authenticator_Status_Type,
  Session_Status_Authentication_Info_AAL,
  Session_Status_Authentication_Info_Authenticator_Mode,
  Session_Status_Authentication_Info_Type,
} from "@/apis/corev1/corev1";
import { AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import Paginator from "@/components/Paginator";
import dayjs from "dayjs";
import { match } from "ts-pattern";
import Editor from "../AccessLogViewer/Editor";
import { InfoDetail } from "../AccessLogViewer/Old";
import { SelectFromTimestamp } from "../AccessLogViewer/utils";
import CardSession from "../Card/CardSession";
import CopyText from "../CopyText";
import InfoItem from "../InfoItem";
import Label from "../Label";
import AuthenticationLogSummary from "../LogSummary/AuthenticationLogSummary";
import { ResourceListLabelWrap } from "../ResourceList";
import TimeAgo from "../TimeAgo";

const AuthenticationLogInfo = (props: { accessLog: AuthenticationLog }) => {
  const x = props.accessLog;
  if (!x.entry) {
    return <></>;
  }

  return (
    <div className="w-full">
      {x.entry.authentication?.info && (
        <div className="w-full">
          {x.entry.authentication.info.downstream?.userAgent && (
            <InfoDetail label="User Agent">
              {x.entry.authentication.info.downstream.userAgent}
            </InfoDetail>
          )}

          {x.entry.authentication.info.downstream?.clientVersion && (
            <InfoDetail label="Client Version">
              {x.entry.authentication.info.downstream.clientVersion}
            </InfoDetail>
          )}

          {x.entry.authentication.info.downstream?.ipAddress && (
            <InfoDetail label="IP Address">
              {x.entry.authentication.info.downstream.ipAddress}
            </InfoDetail>
          )}

          {x.entry.authentication.info.aal !==
            Session_Status_Authentication_Info_AAL.AAL_UNSET && (
            <InfoDetail label="Authenticator Assurance Level">
              {
                Session_Status_Authentication_Info_AAL[
                  x.entry.authentication.info.aal
                ]
              }
            </InfoDetail>
          )}

          {x.entry.authentication.info.details.oneofKind ===
            `identityProvider` && (
            <>
              {x.entry.authentication.info.details.identityProvider.email && (
                <InfoDetail label="Email">
                  {x.entry.authentication.info.details.identityProvider.email}
                </InfoDetail>
              )}
            </>
          )}

          {x.entry.authentication.info.details.oneofKind ===
            `authenticator` && (
            <>
              {x.entry.authentication.info.details.authenticator.type && (
                <InfoDetail label="Authenticator Type">
                  <div className="flex">
                    <Label>
                      {
                        Authenticator_Status_Type[
                          x.entry.authentication.info.details.authenticator.type
                        ]
                      }
                    </Label>
                  </div>
                </InfoDetail>
              )}

              {x.entry.authentication.info.details.authenticator.mode && (
                <InfoDetail label="Authenticator Mode">
                  <div className="flex">
                    <Label>
                      {
                        Session_Status_Authentication_Info_Authenticator_Mode[
                          x.entry.authentication.info.details.authenticator.mode
                        ]
                      }
                    </Label>
                  </div>
                </InfoDetail>
              )}

              {x.entry.authentication.info.details.authenticator.info?.type
                .oneofKind === `fido` && (
                <>
                  {x.entry.authentication.info.details.authenticator.info.type
                    .fido.isPasskey && (
                    <InfoDetail label="Passkey">Yes</InfoDetail>
                  )}

                  {x.entry.authentication.info.details.authenticator.info.type
                    .fido.isHardware && (
                    <InfoDetail label="Hardware-based Authenticator">
                      <Label>Yes</Label>
                    </InfoDetail>
                  )}

                  {x.entry.authentication.info.details.authenticator.info.type
                    .fido.isSoftware && (
                    <InfoDetail label="Software-based Authenticator">
                      Yes
                    </InfoDetail>
                  )}

                  {x.entry.authentication.info.details.authenticator.info.type
                    .fido.isAttestationVerified && (
                    <InfoDetail label="Attestation Verified">Yes</InfoDetail>
                  )}
                </>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
};

export const AuthenticationLogC = (props: { accessLog: AuthenticationLog }) => {
  const x = props.accessLog;

  let [showDetails, setShowDetails] = React.useState(false);
  if (!x.entry) {
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
        {x.entry.authentication &&
          x.entry.authentication.info &&
          x.entry.authentication.info.type && (
            <Label>
              {match(x.entry.authentication.info.type)
                .with(
                  Session_Status_Authentication_Info_Type.AUTHENTICATOR,
                  () => "Authenticator",
                )
                .with(
                  Session_Status_Authentication_Info_Type.CREDENTIAL,
                  () => "Credential",
                )
                .with(
                  Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
                  () => "Identity Provider",
                )
                .with(
                  Session_Status_Authentication_Info_Type.REFRESH_TOKEN,
                  () => "Refresh Token",
                )
                .with(
                  Session_Status_Authentication_Info_Type.INTERNAL,
                  () => "Internal",
                )
                .with(
                  Session_Status_Authentication_Info_Type.EXTERNAL,
                  () => "External",
                )
                .otherwise((o) => Session_Status_Authentication_Info_Type[o])}
            </Label>
          )}
        {x.entry &&
          x.entry.authentication &&
          x.entry.authentication.info &&
          x.entry.authentication.info.aal &&
          x.entry.authentication.info.aal > 0 && (
            <Label>
              {
                Session_Status_Authentication_Info_AAL[
                  x.entry.authentication.info.aal
                ]
              }
            </Label>
          )}
      </div>

      <div className="w-full my-2">
        <InfoItem title="ID">
          <CopyText value={x.metadata!.id} />
        </InfoItem>
        {x.entry.sessionRef && (
          <InfoItem title="Session">
            <ResourceListLabelWrap>
              <CardSession itemRef={x.entry.sessionRef} />
            </ResourceListLabelWrap>
          </InfoItem>
        )}

        <AuthenticationLogInfo accessLog={x} />
      </div>
    </div>
  );
};

const DoAuthenticationLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  itemsPerPage?: number;
  from?: Timestamp;
}) => {
  let [page, setPage] = React.useState(0);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listAuthenticationLog",
      props.userRef?.uid,
      props.sessionRef?.uid,
      props.deviceRef?.uid,
      props.identityProviderRef?.uid,
      page,
      props.from?.seconds,
      props.from?.nanos,
    ],

    queryFn: async () => {
      if (isDev()) {
        const r = await getClientCore().listSession({});
        const sess = r.response.items.at(0);
        const rSvcs = await getClientCore().listService({});
        const svc = rSvcs.response.items.at(0);
        return ListAuthenticationLogResponse.create({
          items: [
            AuthenticationLog.create({
              kind: "AuthenticationLog",
              metadata: {
                createdAt: Timestamp.now(),
                id: "mulb-o92x-p092j5ltc3q1nyajoiidx0tq-1r9h-x3p0",
                actorRef: getResourceRef(sess!),
              },
              entry: {
                sessionRef: getResourceRef(sess!),
                userRef: sess?.status?.userRef,
                deviceRef: sess?.status?.deviceRef,
                authentication: {
                  info: {
                    type: Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
                  },
                },
              },
            }),
          ],
        });
      }

      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLog(
          ListAuthenticationLogRequest.create({
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            common: {
              page,
              itemsPerPage: props.itemsPerPage ? props.itemsPerPage : 100,
            },
            from: props.from,
          }),
        );
      return response;
    },
    refetchInterval: 60000,
  });

  return (
    <div>
      <div className="flex items-center mb-6">
        <div className="text-2xl font-bold flex items-center">
          {qry.isSuccess &&
            qry.data &&
            qry.data.listResponseMeta &&
            qry.data.listResponseMeta.totalCount > 0 && (
              <span className="ml-2">
                <Label>
                  {qry.data.listResponseMeta.totalCount} Logs (Page: {page + 1})
                </Label>
              </span>
            )}
        </div>

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
        {qry.isSuccess && qry.data && qry.data.items.length > 0 && (
          <div>
            {qry.data.items.map((x) => (
              <AuthenticationLogC key={x.metadata!.id} accessLog={x} />
            ))}
          </div>
        )}
      </div>

      <Paginator meta={qry.data?.listResponseMeta} />
    </div>
  );
};

const AuthenticationLogViewer = (props: {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  itemsPerPage?: number;
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

      <AuthenticationLogSummary
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        deviceRef={props.deviceRef}
        identityProviderRef={props.identityProviderRef}
        from={from}
      />

      <DoAuthenticationLogViewer
        userRef={props.userRef}
        sessionRef={props.sessionRef}
        deviceRef={props.deviceRef}
        identityProviderRef={props.identityProviderRef}
        from={from}
      />
    </div>
  );
};

export default AuthenticationLogViewer;
