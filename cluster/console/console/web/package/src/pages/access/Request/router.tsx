import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "access",
  Kind: "Request",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),

    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,
    MainAction: lazySpec(() => import("./RevokeRequest")),
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  unCreatable: true,
  unEditable: true,
};

export default resourceComponentInfo;
