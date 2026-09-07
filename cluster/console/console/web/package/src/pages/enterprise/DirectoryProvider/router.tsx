import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

import { DirectoryProvider } from "@/apis/enterprisev1/enterprisev1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "DirectoryProvider",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),
    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,

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

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  cloneable: true,
};

export default resourceComponentInfo;
