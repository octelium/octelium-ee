import { ObjectReference } from "@/apis/metav1/metav1";

import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import { match } from "ts-pattern";
import AuthenticationLogHealthWidget from "../AuthenticationLogViewer/AuthenticationLogWidget";
import { useContextResource } from "./utils";

export const ResourceAuthenticationLogs = (props: { resource: Resource }) => {
  const { resource } = props;
  if (resource.apiVersion !== `core/v1`) {
    return <></>;
  }

  let userRef: ObjectReference | undefined;
  let sessionRef: ObjectReference | undefined;
  let identityProviderRef: ObjectReference | undefined;
  let deviceRef: ObjectReference | undefined;

  if (
    !match(resource.kind)
      .with("User", () => {
        userRef = getResourceRef(resource);
        return true;
      })
      .with("Session", () => {
        sessionRef = getResourceRef(resource);
        return true;
      })
      .with("IdentityProvider", () => {
        identityProviderRef = getResourceRef(resource);
        return true;
      })
      .with("Device", () => {
        deviceRef = getResourceRef(resource);
        return true;
      })
      .otherwise(() => false)
  ) {
    return <></>;
  }

  return (
    <div>
      <div className="w-full">
        <div className="w-full">
          <div>
            <AuthenticationLogHealthWidget
              userRef={userRef}
              sessionRef={sessionRef}
              deviceRef={deviceRef}
              identityProviderRef={identityProviderRef}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

const ResourceItemAuthenticationLogsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceAuthenticationLogs resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemAuthenticationLogsPage;
