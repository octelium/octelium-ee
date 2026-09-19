import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

import {
  Integration,
  Integration_Spec_Slack,
} from "@/apis/accessv1/accessv1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "access",
  Kind: "Integration",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),

    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,
    MainAction: lazySpec(() => import("./ResolveIdentity")),

    createResource: () =>
      Integration.create({
        apiVersion: "access/v1",
        kind: "Integration",
        metadata: {},
        spec: {
          type: {
            oneofKind: "slack",
            slack: Integration_Spec_Slack.create({
              botToken: { type: { oneofKind: "fromSecret", fromSecret: "" } },
              signingSecret: {
                type: { oneofKind: "fromSecret", fromSecret: "" },
              },
            }),
          },
        },
        status: {},
      }),
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  cloneable: true,
};

export default resourceComponentInfo;
