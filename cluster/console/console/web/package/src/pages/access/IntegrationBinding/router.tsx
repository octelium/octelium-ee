import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo } from "@/pages/utils/lazy";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "access",
  Kind: "IntegrationBinding",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),

    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    hasMain: true,
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  unCreatable: true,
  unEditable: true,
  unDeletable: true,
};

export default resourceComponentInfo;
