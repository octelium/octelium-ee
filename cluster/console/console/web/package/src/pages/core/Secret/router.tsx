import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "Secret",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),
};

export default resourceComponentInfo;
