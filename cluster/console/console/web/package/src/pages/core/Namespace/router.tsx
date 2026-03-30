import { ResourceComponentInfo } from "@/pages/utils/types";

import { ExtraComponent, LabelComponent } from "./List";
import Edit from "./Edit";
import Main from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "Namespace",
  List: {
    // @ts-ignore
    labelComponent: LabelComponent,
    // @ts-ignore
    extraComponent: ExtraComponent,
  },
  Item: {
    // @ts-ignore
    Edit: Edit,
    // @ts-ignore
    Main: Main,
  },
};

export default resourceComponentInfo;
