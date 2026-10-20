import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import AuthenticationLogViewer from "@/components/AuthenticationLogViewer";
import {
  parseAuthenticationLogStatus,
  parseFailureReason,
} from "@/components/AuthenticationLogViewer/failure";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";

export default () => {
  const req = useLogListReq();
  const pagination = useLogPageParams();
  return (
    <LogPageShell title="Authentication logs">
      <AuthenticationLogViewer
        userRef={toObjectRef(req?.userRef)}
        sessionRef={toObjectRef(req?.sessionRef)}
        deviceRef={toObjectRef(req?.deviceRef)}
        identityProviderRef={toObjectRef(req?.identityProviderRef)}
        credentialRef={toObjectRef(req?.credentialRef)}
        authenticatorRef={toObjectRef(req?.authenticatorRef)}
        status={parseAuthenticationLogStatus(req?.status)}
        reason={parseFailureReason(req?.reason)}
        itemsPerPage={25}
        page={pagination.page}
        onPageChange={pagination.setPage}
        query={req?.common?.query}
      />
    </LogPageShell>
  );
};
