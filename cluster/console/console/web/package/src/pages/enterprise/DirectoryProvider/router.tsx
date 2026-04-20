import { ResourceComponentInfo } from "@/pages/utils/types";

import { DirectoryProvider } from "@/apis/enterprisev1/enterprisev1";
import Edit from "./Edit";
import { ExtraComponent, LabelComponent } from "./List";
import Main, { MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "DirectoryProvider",
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

    createResource: () => {
      return DirectoryProvider.create({
        apiVersion: "enterprise/v1",
        kind: "DirectoryProvider",
        metadata: {},
        spec: {
          type: {
            oneofKind: "scim",
            scim: {},
          },
        },
        status: {},
      });
    },
  },

  // @ts-ignore
  infoItemsGetter: MainInfo,
};

export default resourceComponentInfo;
