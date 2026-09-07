import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

import {
  IdentityProvider,
  IdentityProvider_Spec_Github,
} from "@/apis/corev1/corev1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "core",
  Kind: "IdentityProvider",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),

    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,

    createResource: () => {
      return IdentityProvider.create({
        apiVersion: "core/v1",
        kind: "IdentityProvider",
        metadata: {},
        spec: {
          type: {
            oneofKind: "github",
            github: {
              clientSecret: {
                type: {
                  oneofKind: "fromSecret",
                  fromSecret: "",
                },
              },
            } as IdentityProvider_Spec_Github,
          },
        },
        status: {},
      });
    },
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  cloneable: true,
};

export default resourceComponentInfo;
