import AccessLogViewer from "@/components/AccessLogViewer";
import type { AccessLogStatusFilter } from "@/components/AccessLogViewer/utils";
import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";
import { ShieldCheck } from "lucide-react";

export default () => {
  const req = useLogListReq();
  const pagination = useLogPageParams();

  return (
    <LogPageShell
      title="Access logs"
      description="Authorization decisions and request activity across Services, identities, and Policies."
      icon={ShieldCheck}
    >
      <AccessLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        serviceRef={toObjectRef(req?.serviceRef)}
        namespaceRef={toObjectRef(req?.namespaceRef)}
        regionRef={toObjectRef(req?.regionRef)}
        policyRef={toObjectRef(req?.policyRef)}
        status={req?.status as AccessLogStatusFilter | undefined}
        query={req?.common?.query}
        itemsPerPage={25}
        page={pagination.page}
        onPageChange={pagination.setPage}
      />
    </LogPageShell>
  );
};
