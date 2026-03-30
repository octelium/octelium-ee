import AccessLogViewer from "@/components/AccessLogViewer";
import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";

export default () => {
  const req = useLogListReq();

  return (
    <div className="w-full">
      <AccessLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        serviceRef={toObjectRef(req?.serviceRef)}
        namespaceRef={toObjectRef(req?.namespaceRef)}
        regionRef={toObjectRef(req?.regionRef)}
        policyRef={toObjectRef(req?.policyRef)}
      />
    </div>
  );
};
