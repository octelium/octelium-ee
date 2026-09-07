import { SecretStore } from "@/apis/enterprisev1/enterprisev1";
import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "SecretStore",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),
    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,
  },
  unCreatable: true,
  unDeletable: true,
  readOnlyEdit: true,

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),
};

export default resourceComponentInfo;
