import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import AuthenticationLogViewer from "@/components/AuthenticationLogViewer";

export default () => {
  const req = useLogListReq();
  return (
    <div className="w-full">
      <AuthenticationLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        identityProviderRef={toObjectRef(req?.identityProviderRef)}
      />
    </div>
  );
};
