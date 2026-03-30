import { ListSSHSessionRequest } from "@/apis/visibilityv1/visibilityv1";
import { useLogListReq } from "@/components/AccessLogViewer/listReq";
import { NoLogFound } from "@/components/AccessLogViewer/utils";
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
    <div className="w-full">
      {qry.data && qry.data.items.length === 0 && <NoLogFound />}
      {qry.data && qry.data.items.length > 0 && (
        <div>
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
