import { ObjectReference } from "@/apis/metav1/metav1";

import PageWrap from "@/components/PageWrap";
import { getResourceRef, Resource } from "@/utils/pb";
import { match } from "ts-pattern";
import AccessLogViewer from "../AccessLogViewer";
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
    <div>
      <div className="w-full">
        <div className="w-full">
          <div>
            <AccessLogViewer
              userRef={userRef}
              serviceRef={serviceRef}
              namespaceRef={namespaceRef}
              deviceRef={deviceRef}
              sessionRef={sessionRef}
              policyRef={policyRef}
              itemsPerPage={props.itemsPerPage}
            />
          </div>
        </div>
      </div>
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
