import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import AuthenticationLogViewer from "@/components/AuthenticationLogViewer";
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
        itemsPerPage={25}
        page={pagination.page}
        onPageChange={pagination.setPage}
        query={req?.common?.query}
      />
    </LogPageShell>
  );
};
