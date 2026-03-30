import { ResourceComponentInfo } from "@/pages/utils/types";

import Edit from "./Edit";
import { ExtraComponent, LabelComponent, ListFilter, Summary } from "./List";
import Main, { ResourceItemInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "User",
  List: {
    // @ts-ignore
    labelComponent: LabelComponent,
    // @ts-ignore
    extraComponent: ExtraComponent,

    listFilter: ListFilter,

    SummaryComponent: Summary,
  },
  Item: {
    // @ts-ignore
    Edit: Edit,
    // @ts-ignore
    Main: Main,
    // @ts-ignore
    itemInfo: ResourceItemInfo,
  },
};

export default resourceComponentInfo;
