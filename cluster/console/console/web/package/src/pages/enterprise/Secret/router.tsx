import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazySpec } from "@/pages/utils/lazy";
import { Secret, Secret_Spec } from "@/apis/enterprisev1/enterprisev1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
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
        apiVersion: "enterprise/v1",
        kind: "Secret",
        metadata: {},
        spec: Secret_Spec.create(),
        status: {},
      }),
  },
};

export default resourceComponentInfo;
