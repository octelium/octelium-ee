import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import AuditLogViewer from "@/components/AuditLogViewer";

export default () => {
  const req = useLogListReq();

  return (
    <div className="w-full">
      <AuditLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        serviceRef={toObjectRef(req?.serviceRef)}
        resourceRef={toObjectRef(req?.resourceRef)}
      />
    </div>
  );
};
