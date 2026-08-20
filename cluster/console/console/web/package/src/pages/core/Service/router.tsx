import { ResourceComponentInfo } from "@/pages/utils/types";
import { Resource } from "@/utils/pb";
import LLMPlayground from "@/components/LLMPlayground";
import MCPPlayground from "@/components/MCPPlayground";
import * as CoreC from "@/apis/corev1/corev1";

import Edit from "./Edit";
import { ExtraComponent, LabelComponent, Summary } from "./List";
import Main, { ItemInfo, MainInfo } from "./Main";

const MainAction = (props: { item: Resource }) => {
  const service = props.item as CoreC.Service;
  return (
    <>
      <MCPPlayground service={service} />
      <LLMPlayground service={service} />
    </>
  );
};

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "Service",
  List: {
    // @ts-ignore
    labelComponent: LabelComponent,
    // @ts-ignore
    extraComponent: ExtraComponent,

    SummaryComponent: Summary,
  },
  Item: {
    // @ts-ignore
    Edit: Edit,
    // @ts-ignore
    Main: Main,
    // @ts-ignore
    itemInfo: ItemInfo,
    MainAction,
  },

  // @ts-ignore
  infoItemsGetter: MainInfo,

  cloneable: true,
};

export default resourceComponentInfo;
