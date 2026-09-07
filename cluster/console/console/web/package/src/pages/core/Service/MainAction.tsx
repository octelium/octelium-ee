import * as CoreC from "@/apis/corev1/corev1";
import LLMPlayground from "@/components/LLMPlayground";
import MCPPlayground from "@/components/MCPPlayground";
import { Resource } from "@/utils/pb";

const MainAction = (props: { item: Resource }) => {
  const service = props.item as CoreC.Service;
  return (
    <>
      <MCPPlayground service={service} />
      <LLMPlayground service={service} />
    </>
  );
};

export default MainAction;
