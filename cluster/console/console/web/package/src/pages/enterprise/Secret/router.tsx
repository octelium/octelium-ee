import { ResourceComponentInfo } from "@/pages/utils/types";
import { Secret, Secret_Spec } from "@/apis/enterprisev1/enterprisev1";

import Edit from "./Edit";
import { ExtraComponent, LabelComponent, Summary } from "./List";
import Main from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "Secret",
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
    createResource: () => Secret.create({
      apiVersion: "enterprise/v1",
      kind: "Secret",
      metadata: {},
      spec: Secret_Spec.create(),
      status: {},
    }),
  },
};

export default resourceComponentInfo;
