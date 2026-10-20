import {
  AccessLog_Entry_Common_Reason_Type,
  Service_Spec_Mode,
} from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import type { AccessLogStatusFilter } from "@/components/AccessLogViewer/utils";
import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";

export default () => {
  const req = useLogListReq();
  const pagination = useLogPageParams();
  const reason = req?.reason
    ? AccessLog_Entry_Common_Reason_Type[
        req.reason.toUpperCase() as keyof typeof AccessLog_Entry_Common_Reason_Type
      ]
    : undefined;
  const mode = req?.mode
    ? Service_Spec_Mode[
        req.mode.toUpperCase() as keyof typeof Service_Spec_Mode
      ]
    : undefined;

  return (
    <LogPageShell title="Access logs">
      <AccessLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        serviceRef={toObjectRef(req?.serviceRef)}
        namespaceRef={toObjectRef(req?.namespaceRef)}
        regionRef={toObjectRef(req?.regionRef)}
        policyRef={toObjectRef(req?.policyRef)}
        status={req?.status as AccessLogStatusFilter | undefined}
        reason={reason}
        mode={mode}
        query={req?.common?.query}
        itemsPerPage={25}
        page={pagination.page}
        onPageChange={pagination.setPage}
      />
    </LogPageShell>
  );
};
