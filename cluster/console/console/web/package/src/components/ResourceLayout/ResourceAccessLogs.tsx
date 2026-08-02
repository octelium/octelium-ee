import { ObjectReference } from "@/apis/metav1/metav1";

import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import { match } from "ts-pattern";
import { AccessLogList } from "../AccessLogViewer";
import AccessLogHealthWidget from "../AccessLogViewer/AccessLogWidget";
import { useContextResource } from "./utils";

export const ResourceAccessLogs = (props: {
  resource: Resource;
  itemsPerPage?: number;
}) => {
  const { resource } = props;
  if (resource.apiVersion !== `core/v1`) {
    return <></>;
  }

  let userRef: ObjectReference | undefined;
  let sessionRef: ObjectReference | undefined;
  let deviceRef: ObjectReference | undefined;
  let namespaceRef: ObjectReference | undefined;
  let serviceRef: ObjectReference | undefined;
  let policyRef: ObjectReference | undefined;

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
      .with("Device", () => {
        deviceRef = getResourceRef(resource);
        return true;
      })
      .with("Service", () => {
        serviceRef = getResourceRef(resource);
        return true;
      })
      .with("Namespace", () => {
        namespaceRef = getResourceRef(resource);
        return true;
      })
      .with("Policy", () => {
        policyRef = getResourceRef(resource);
        return true;
      })
      .otherwise(() => false)
  ) {
    return <></>;
  }

  return (
    <div className="flex w-full flex-col gap-8">
      <AccessLogHealthWidget
        userRef={userRef}
        serviceRef={serviceRef}
        namespaceRef={namespaceRef}
        deviceRef={deviceRef}
        sessionRef={sessionRef}
        policyRef={policyRef}
      />

      <section className="flex w-full flex-col gap-4 border-t border-slate-200 pt-6">
        <div>
          <h2 className="text-sm font-bold text-slate-800">Access logs</h2>
          <p className="mt-1 text-[0.72rem] font-medium text-slate-500">
            Recent access activity for this resource
          </p>
        </div>
        <AccessLogList
          userRef={userRef}
          serviceRef={serviceRef}
          namespaceRef={namespaceRef}
          deviceRef={deviceRef}
          sessionRef={sessionRef}
          policyRef={policyRef}
          itemsPerPage={props.itemsPerPage ?? 25}
        />
      </section>
    </div>
  );
};

const ResourceItemAccessLogsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceAccessLogs resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemAccessLogsPage;
