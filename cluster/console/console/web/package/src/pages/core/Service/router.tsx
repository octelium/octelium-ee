import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";
import { Resource } from "@/utils/pb";
import * as CoreC from "@/apis/corev1/corev1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "Service",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),

    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,
    MainAction: lazySpec(() => import("./MainAction")),
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  cloneable: true,
};

export default resourceComponentInfo;
