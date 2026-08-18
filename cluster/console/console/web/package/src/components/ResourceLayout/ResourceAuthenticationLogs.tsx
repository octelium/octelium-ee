import { ObjectReference } from "@/apis/metav1/metav1";

import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import { Activity, ShieldUser } from "lucide-react";
import * as React from "react";
import { match } from "ts-pattern";
import { AuthenticationLogList } from "../AuthenticationLogViewer";
import AuthenticationLogHealthWidget from "../AuthenticationLogViewer/AuthenticationLogWidget";
import { useContextResource } from "./utils";

export const ResourceAuthenticationLogs = (props: {
  resource: Resource;
  itemsPerPage?: number;
}) => {
  const { resource } = props;
  const [periodMinutes, setPeriodMinutes] = React.useState(360);

  if (resource.apiVersion !== `core/v1`) {
    return <></>;
  }

  let userRef: ObjectReference | undefined;
  let sessionRef: ObjectReference | undefined;
  let identityProviderRef: ObjectReference | undefined;
  let deviceRef: ObjectReference | undefined;
  let credentialRef: ObjectReference | undefined;
  let authenticatorRef: ObjectReference | undefined;

  if (
    !match(resource.kind)
      .with("User", () => {
        userRef = getResourceRef(resource);
        return true;
      })
      .with("Session", () => {
        sessionRef = getResourceRef(resource);
        return true;
      })
      .with("IdentityProvider", () => {
        identityProviderRef = getResourceRef(resource);
        return true;
      })
      .with("Device", () => {
        deviceRef = getResourceRef(resource);
        return true;
      })
      .with("Credential", () => {
        credentialRef = getResourceRef(resource);
        return true;
      })
      .with("Authenticator", () => {
        authenticatorRef = getResourceRef(resource);
        return true;
      })
      .otherwise(() => false)
  ) {
    return <></>;
  }

  const displayName =
    resource.metadata?.displayName || resource.metadata?.name || "Resource";

  return (
    <div className="flex w-full flex-col gap-4">
      <header className="flex flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_2px_10px_rgba(15,23,42,0.04)] sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
            <ShieldUser size={18} strokeWidth={2.2} />
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h1 className="truncate text-base font-bold text-slate-900">
                Authentication logs
              </h1>
              <Activity size={14} className="shrink-0 text-slate-400" />
            </div>
            <p className="mt-0.5 truncate text-[0.72rem] font-semibold text-slate-500">
              Review sign-ins, authenticators, credentials, and sessions.
            </p>
            <span className="mt-2 inline-flex max-w-full truncate rounded-md bg-slate-100 px-2 py-1 text-[0.65rem] font-bold text-slate-600">
              {resource.kind} · {displayName}
            </span>
          </div>
        </div>
        <span className="inline-flex w-fit shrink-0 items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-[0.64rem] font-bold uppercase tracking-[0.06em] text-slate-500">
          Resource scope
        </span>
      </header>

      <AuthenticationLogHealthWidget
        userRef={userRef}
        sessionRef={sessionRef}
        deviceRef={deviceRef}
        identityProviderRef={identityProviderRef}
        credentialRef={credentialRef}
        authenticatorRef={authenticatorRef}
        periodMinutes={periodMinutes}
        onPeriodChange={setPeriodMinutes}
      />

      <section className="flex w-full flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_2px_10px_rgba(15,23,42,0.03)]">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-sm font-bold text-slate-800">Event stream</h2>
            <p className="mt-1 text-[0.72rem] font-medium text-slate-500">
              Individual authentication events matching the selected time range.
            </p>
          </div>
          <span className="text-[0.65rem] font-bold text-slate-400">
            Updates automatically
          </span>
        </div>
        <AuthenticationLogList
          userRef={userRef}
          sessionRef={sessionRef}
          deviceRef={deviceRef}
          identityProviderRef={identityProviderRef}
          credentialRef={credentialRef}
          authenticatorRef={authenticatorRef}
          periodMinutes={periodMinutes}
          itemsPerPage={props.itemsPerPage ?? 25}
        />
      </section>
    </div>
  );
};

const ResourceItemAuthenticationLogsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceAuthenticationLogs resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemAuthenticationLogsPage;
