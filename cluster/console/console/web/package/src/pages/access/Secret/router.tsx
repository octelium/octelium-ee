import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

import { Secret, Secret_Spec } from "@/apis/accessv1/accessv1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "access",
  Kind: "Secret",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),

    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,

    createResource: () =>
      Secret.create({
        apiVersion: "access/v1",
        kind: "Secret",
        metadata: {},
        spec: Secret_Spec.create(),
        status: {},
      }),
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),
};

export default resourceComponentInfo;
