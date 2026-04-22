import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  GetSSHSessionRequest,
  SSHSession,
  SSHSession_State,
} from "@/apis/visibilityv1/visibilityv1";
import { SSHSessionC } from "@/components/SSHRecordingPlayer";
import { XTermSSHReplay } from "@/components/SSHRecordingPlayer/Player";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";

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
    refetchInterval: 60000,
  });

  if (!qry.data) {
    return <></>;
  }

  return (
    <div className="w-full">
      <SSHSessionC item={qry.data} />

      <div className="my-8"></div>

      <XTermSSHReplay sshSession={qry.data} />
    </div>
  );
};
