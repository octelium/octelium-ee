import { ResourceComponentInfo } from "@/pages/utils/types";

import Edit from "./Edit";
import { ExtraComponent, LabelComponent, Summary } from "./List";
import Main, { ItemInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "Authenticator",
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
  },
  unCreatable: true,
  unEditable: true,
};

export default resourceComponentInfo;
