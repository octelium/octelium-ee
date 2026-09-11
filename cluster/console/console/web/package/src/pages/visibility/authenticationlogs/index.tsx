import {
  toObjectRef,
  useLogListReq,
} from "@/components/AccessLogViewer/listReq";
import AuthenticationLogViewer from "@/components/AuthenticationLogViewer";
import LogPageShell, {
  useLogPageParams,
} from "@/components/LogViewer/LogPageShell";
import { ShieldUser } from "lucide-react";

export default () => {
  const req = useLogListReq();
  const pagination = useLogPageParams();
  return (
    <LogPageShell
      title="Authentication logs"
      description="Login, re-authentication, credential, identity-provider, and authenticator events."
      icon={ShieldUser}
    >
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
