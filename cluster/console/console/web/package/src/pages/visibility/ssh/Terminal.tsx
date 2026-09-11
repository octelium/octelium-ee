import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  GetSSHSessionRequest,
  SSHSession,
  SSHSession_State,
} from "@/apis/visibilityv1/visibilityv1";
import { SSHSessionC } from "@/components/SSHRecordingPlayer";
import { ListLoading } from "@/components/Loading";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, RotateCcw, SquareTerminal } from "lucide-react";
import * as React from "react";
import { useParams } from "react-router-dom";

const XTermSSHReplay = React.lazy(() =>
  import("@/components/SSHRecordingPlayer/Player").then((module) => ({
    default: module.XTermSSHReplay,
  })),
);

export default () => {
  const { name } = useParams();

  const qry = useQuery({
    queryKey: ["visibility", "getSSHSession", name],

    queryFn: async () => {
      if (isDev()) {
        return SSHSession.create({
          id: "12345",
          startedAt: Timestamp.now(),
          endedAt: Timestamp.now(),
          state: SSHSession_State.COMPLETED,
        });
      }

      const { response } = await getClientVisibilityAccessLog().getSSHSession(
        GetSSHSessionRequest.create({
          id: name,
        }),
      );
      return response;
    },
    enabled: !!name,
    refetchInterval: (query) =>
      query.state.data?.state === SSHSession_State.ONGOING ? 60_000 : false,
  });

  if (qry.isLoading) {
    return <ListLoading label="SSH recording" />;
  }

  if (qry.isError || !qry.data) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-4">
        <div className="w-full max-w-lg rounded-xl border border-red-200 bg-white p-5 text-center shadow-card">
          <AlertTriangle className="mx-auto text-red-500" size={24} />
          <h1 className="mt-3 text-base font-semibold text-slate-900">
            SSH recording could not be loaded
          </h1>
          <p className="mt-1 break-words text-xs leading-5 text-slate-600">
            {qry.error instanceof Error
              ? qry.error.message
              : `No recording was found for ${name}.`}
          </p>
          <button
            type="button"
            onClick={() => qry.refetch()}
            className="mx-auto mt-4 inline-flex items-center gap-1.5 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 shadow-card hover:bg-slate-50"
          >
            <RotateCcw size={12} />
            Try again
          </button>
        </div>
      </div>
    );
  }

  return (
    <main className="flex w-full flex-col gap-4 py-4">
      <section className="rounded-xl border border-slate-200 bg-white p-4 shadow-card">
        <SSHSessionC item={qry.data} />
      </section>

      <section className="overflow-hidden rounded-xl border border-slate-800 bg-slate-950 shadow-raised">
        <header className="flex items-center gap-2 border-b border-slate-800 px-4 py-3 text-slate-200">
          <SquareTerminal size={15} />
          <h2 className="text-xs font-semibold uppercase tracking-[0.06em]">
            Terminal replay
          </h2>
        </header>
        <React.Suspense
          fallback={
            <div
              className="h-[460px] animate-pulse bg-slate-900"
              aria-label="Loading terminal player"
            />
          }
        >
          <XTermSSHReplay sshSession={qry.data} />
        </React.Suspense>
      </section>
    </main>
  );
};
