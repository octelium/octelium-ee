import { ResourceComponentInfo } from "@/pages/utils/types";

import Edit from "./Edit";
import { ExtraComponent, LabelComponent, Summary } from "./List";
import Main from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "Secret",
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
  },
};

export default resourceComponentInfo;
