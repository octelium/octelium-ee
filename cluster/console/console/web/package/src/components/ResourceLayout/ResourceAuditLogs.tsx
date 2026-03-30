import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import AuditLogViewer from "../AuditLogViewer";
import { useContextResource } from "./utils";

export const ResourceAuditLogs = (props: { resource: Resource }) => {
  const { resource } = props;

  return (
    <div>
      <div className="w-full">
        <div className="w-full">
          <div>
            <AuditLogViewer resourceRef={getResourceRef(resource)} />
          </div>
        </div>
      </div>
    </div>
  );
};

const ResourceItemAuditLogsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceAuditLogs resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemAuditLogsPage;
