import { ResourceComponentInfo } from "@/pages/utils/types";

import {
  IdentityProvider,
  IdentityProvider_Spec_Github,
} from "@/apis/corev1/corev1";
import Edit from "./Edit";
import { ExtraComponent, LabelComponent, Summary } from "./List";
import Main, { ItemInfo, MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "IdentityProvider",
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

    createResource: () => {
      return IdentityProvider.create({
        apiVersion: "core/v1",
        kind: "IdentityProvider",
        metadata: {},
        spec: {
          type: {
            oneofKind: "github",
            github: {
              clientSecret: {
                type: {
                  oneofKind: "fromSecret",
                  fromSecret: "",
                },
              },
            } as IdentityProvider_Spec_Github,
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
