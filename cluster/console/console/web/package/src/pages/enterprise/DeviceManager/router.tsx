import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

import {
  DeviceManager,
  DeviceManager_Spec,
  DeviceManager_Spec_CrowdStrike,
} from "@/apis/enterprisev1/enterprisev1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "DeviceManager",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),
    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,

    createResource: () => {
      return DeviceManager.create({
        apiVersion: "enterprise/v1",
        kind: "DeviceManager",
        metadata: {},
        spec: DeviceManager_Spec.create({
          type: {
            oneofKind: "crowdStrike",
            crowdStrike: DeviceManager_Spec_CrowdStrike.create(),
          },
        }),
        status: {},
      });
    },
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  cloneable: true,
};

export default resourceComponentInfo;
