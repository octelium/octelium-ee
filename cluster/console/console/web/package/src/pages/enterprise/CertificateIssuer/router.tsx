import { ResourceComponentInfo } from "@/pages/utils/types";

import Edit from "./Edit";
import { ExtraComponent, LabelComponent } from "./List";
import Main, { MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "CertificateIssuer",
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
  unCreatable: true,
  unDeletable: true,

  // @ts-ignore
  infoItemsGetter: MainInfo,
};

export default resourceComponentInfo;
