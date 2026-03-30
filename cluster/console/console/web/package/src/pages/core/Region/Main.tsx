import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import { getResourceRef } from "@/utils/pb";

export const AccessLog = (props: { item: CoreC.Region }) => {
  return <AccessLogViewer regionRef={getResourceRef(props.item)} />;
};

export default (props: { item: CoreC.Region }) => {
  const { item } = props;
  return <div className="w-full"></div>;
};
