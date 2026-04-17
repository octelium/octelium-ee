import { AccessLog, ComponentLog } from "@/apis/corev1/corev1";
import { AuditLog, AuthenticationLog } from "@/apis/enterprisev1/enterprisev1";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityComponentLog,
} from "@/utils/client";
import { getVisibilityAPIKindFromPath } from "@/utils/pb";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useSearchParams } from "react-router-dom";
import { match } from "ts-pattern";
import * as VisibilityP from "../../apis/visibilityv1/visibilityv1";
import { AccessLogC } from "../AccessLogViewer";
import { NoLogFound } from "../AccessLogViewer/utils";
import { AuditLogC } from "../AuditLogViewer";
import { AuthenticationLogC } from "../AuthenticationLogViewer";
import { ComponentLogC } from "../ComponentLogViewer";
import AccessLogSummary from "../LogSummary/AccessLogSummary";
import AuthenticationLogSummary from "../LogSummary/AuthenticationLogSummary";
import ComponentLogSummary from "../LogSummary/ComponentLogSummary";
import Paginator from "../Paginator";
import { parseQueryString } from "../ResourceLayout/queryParse";

type objectRef = {
  uid?: string;
  name?: string;
};

export default () => {
  let [searchParams, _] = useSearchParams();

  const loc = useLocation();
  const kind = getVisibilityAPIKindFromPath(loc.pathname);

  let parsedQry = parseQueryString<{
    type?: string;
    mode?: string;
    level?: string;
    common?: {
      page?: number;
      itemsPerPage?: number;
    };
    namespaceRef?: objectRef;
    userRef?: objectRef;
    sessionRef?: objectRef;
    serviceRef?: objectRef;
    identityProviderRef?: objectRef;
    authenticatorRef?: objectRef;
    credentialRef?: objectRef;
    resourceRef?: objectRef;
    regionRef?: objectRef;
    deviceRef?: objectRef;
    policyRef?: objectRef;
  }>(searchParams.toString());

  if (parsedQry.common && parsedQry.common.page && parsedQry.common.page > 0) {
    parsedQry.common.page = parsedQry.common.page - 1;
  }

  // @ts-ignore
  const req = VisibilityP[`List${kind}Request`][`fromJsonString`](
    JSON.stringify(parsedQry),
  );

  const qry = useQuery({
    queryKey: [loc.pathname, JSON.stringify(parsedQry)],
    queryFn: async () => {
      return await match(kind)
        .with(`AccessLog`, () =>
          getClientVisibilityAccessLog().listAccessLog(req),
        )
        .with(`AuditLog`, () => getClientVisibilityAuditLog().listAuditLog(req))
        .with(`AuthenticationLog`, () =>
          getClientVisibilityAuthenticationLog().listAuthenticationLog(req),
        )
        .with(`ComponentLog`, () =>
          getClientVisibilityComponentLog().listComponentLog(req),
        )
        .otherwise(() => undefined);
    },
    refetchInterval: 60000,
  });

  if (!qry.data || qry.isLoading) {
    return <></>;
  }

  return (
    <div className="w-full">
      {qry.data && qry.data.response.items.length === 0 && <NoLogFound />}

      {qry.data && qry.data.response.items.length > 0 && (
        <div>
          {match(kind)
            .with(`AccessLog`, () => {
              const r = req as VisibilityP.ListAccessLogRequest;
              return (
                <AccessLogSummary
                  serviceRef={r.serviceRef}
                  namespaceRef={r.namespaceRef}
                  sessionRef={r.sessionRef}
                  deviceRef={r.deviceRef}
                  userRef={r.deviceRef}
                  regionRef={r.regionRef}
                  policyRef={r.policyRef}
                  from={r.from}
                  to={r.to}
                />
              );
            })

            .with(`AuthenticationLog`, () => {
              const r = req as VisibilityP.ListAuthenticationLogRequest;
              return (
                <AuthenticationLogSummary
                  sessionRef={r.sessionRef}
                  deviceRef={r.deviceRef}
                  userRef={r.deviceRef}
                  identityProviderRef={r.identityProviderRef}
                  credentialRef={r.credentialRef}
                  authenticatorRef={r.authenticatorRef}
                  from={r.from}
                  to={r.to}
                />
              );
            })
            .with(`ComponentLog`, () => {
              const r = req as VisibilityP.ListComponentLogRequest;
              return (
                <ComponentLogSummary level={r.level} from={r.from} to={r.to} />
              );
            })

            .otherwise(() => (
              <></>
            ))}
        </div>
      )}

      {qry.data && qry.data.response.listResponseMeta && (
        <div className="mt-4 flex items-center justify-center">
          <Paginator meta={qry.data.response.listResponseMeta} />
        </div>
      )}
      <div className="ml-4 mt-4">
        {qry.data && qry.data.response.items.length > 0 && (
          <div>
            {qry.data.response.items.map((x) =>
              match(kind)
                .with(`AccessLog`, () => (
                  <AccessLogC key={x.metadata!.id} accessLog={x as AccessLog} />
                ))
                .with(`AuditLog`, () => (
                  <AuditLogC key={x.metadata!.id} auditLog={x as AuditLog} />
                ))
                .with(`AuthenticationLog`, () => (
                  <AuthenticationLogC
                    key={x.metadata!.id}
                    authLog={x as AuthenticationLog}
                  />
                ))
                .with(`ComponentLog`, () => (
                  <ComponentLogC key={x.metadata!.id} log={x as ComponentLog} />
                ))
                .otherwise(() => <></>),
            )}
          </div>
        )}
      </div>

      {qry.data && qry.data.response.listResponseMeta && (
        <div className="mt-4 flex items-center justify-center">
          <Paginator meta={qry.data.response.listResponseMeta} />
        </div>
      )}
    </div>
  );
};
