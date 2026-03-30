import { ResourceComponentInfo } from "@/pages/utils/types";

import { ExtraComponent, LabelComponent } from "./List";
import Edit from "./Edit";
import Main from "./Main";

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
};

export default resourceComponentInfo;
