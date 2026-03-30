import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import { getResourceRef } from "@/utils/pb";

export const AccessLog = (props: { item: CoreC.Namespace }) => {
  return <AccessLogViewer namespaceRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreC.Namespace }) => {
  const { item } = props;
  return <div className="w-full"></div>;
};
