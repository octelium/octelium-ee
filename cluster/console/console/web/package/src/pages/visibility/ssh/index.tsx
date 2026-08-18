import { toObjectRef, useLogListReq } from "@/components/AccessLogViewer/listReq";
import { SSHSessionViewer } from "@/components/SSHRecordingPlayer";

export default () => {
  const request = useLogListReq();

  return (
    <SSHSessionViewer
      userRef={toObjectRef(request?.userRef)}
      sessionRef={toObjectRef(request?.sessionRef)}
      serviceRef={toObjectRef(request?.serviceRef)}
      namespaceRef={toObjectRef(request?.namespaceRef)}
      deviceRef={toObjectRef(request?.deviceRef)}
      itemsPerPage={request?.common?.itemsPerPage}
      page={request?.common?.page}
    />
  );
};
