import { ListSSHSessionRequest } from "@/apis/visibilityv1/visibilityv1";
import { useLogListReq } from "@/components/AccessLogViewer/listReq";
import Paginator from "@/components/Paginator";
import {
  ResourceListItem,
  ResourceListWrapper,
} from "@/components/ResourceList";
import { SSHSessionC } from "@/components/SSHRecordingPlayer";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

export default () => {
  const r = useLogListReq();

  const req = r
    ? ListSSHSessionRequest.fromJsonString(JSON.stringify(r))
    : ListSSHSessionRequest.create({});

  const qry = useQuery({
    queryKey: [
      "visibility",
      "listSSHSession",
      ListSSHSessionRequest.toJsonString(req),
    ],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().listSSHSession(req);
      return response;
    },
    refetchInterval: 60000,
  });

  return (
    <div className="w-full flex flex-col gap-6">
      {qry.data?.items.length === 0 && (
        <div className="flex items-center justify-center py-16">
          <span className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
            No SSH sessions found
          </span>
        </div>
      )}

      {qry.data && qry.data.items.length > 0 && (
        <>
          <ResourceListWrapper>
            {qry.data.items.map((x) => (
              <ResourceListItem key={x.id} path={`/visibility/ssh/${x.id}`}>
                <SSHSessionC item={x} />
              </ResourceListItem>
            ))}
          </ResourceListWrapper>
          <Paginator meta={qry.data.listResponseMeta} />
        </>
      )}
    </div>
  );
};
