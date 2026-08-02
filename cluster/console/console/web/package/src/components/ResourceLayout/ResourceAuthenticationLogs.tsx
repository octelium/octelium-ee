import { ObjectReference } from "@/apis/metav1/metav1";

import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import { match } from "ts-pattern";
import { AuthenticationLogList } from "../AuthenticationLogViewer";
import AuthenticationLogHealthWidget from "../AuthenticationLogViewer/AuthenticationLogWidget";
import { useContextResource } from "./utils";

export const ResourceAuthenticationLogs = (props: {
  resource: Resource;
  itemsPerPage?: number;
}) => {
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
    <div className="flex w-full flex-col gap-8">
      <AuthenticationLogHealthWidget
        userRef={userRef}
        sessionRef={sessionRef}
        deviceRef={deviceRef}
        identityProviderRef={identityProviderRef}
      />

      <section className="flex w-full flex-col gap-4 border-t border-slate-200 pt-6">
        <div>
          <h2 className="text-sm font-bold text-slate-800">
            Authentication logs
          </h2>
          <p className="mt-1 text-[0.72rem] font-medium text-slate-500">
            Recent authentication activity for this resource
          </p>
        </div>
        <AuthenticationLogList
          userRef={userRef}
          sessionRef={sessionRef}
          deviceRef={deviceRef}
          identityProviderRef={identityProviderRef}
          itemsPerPage={props.itemsPerPage ?? 25}
        />
      </section>
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
