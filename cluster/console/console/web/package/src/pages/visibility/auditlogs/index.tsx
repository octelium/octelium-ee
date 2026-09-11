import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import AuditLogViewer from "@/components/AuditLogViewer";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";
import { ClipboardList } from "lucide-react";

export default () => {
  const req = useLogListReq();
  const pagination = useLogPageParams();

  return (
    <LogPageShell
      title="Audit logs"
      description="Administrative changes to cluster resources, including the actor and affected object."
      icon={ClipboardList}
    >
      <AuditLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        resourceRef={toObjectRef(req?.resourceRef ?? req?.serviceRef)}
        itemsPerPage={25}
        page={pagination.page}
        onPageChange={pagination.setPage}
        query={req?.common?.query}
      />
    </LogPageShell>
  );
};
