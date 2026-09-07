import { ResourceComponentInfo } from "@/pages/utils/types";
import { lazyNamed, lazyResourceInfo, lazySpec } from "@/pages/utils/lazy";

import {
  CollectorExporter,
  CollectorExporter_Spec,
  CollectorExporter_Spec_OTLP,
} from "@/apis/enterprisev1/enterprisev1";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "CollectorExporter",
  List: {
    labelComponent: lazyNamed(() => import("./List"), "LabelComponent"),
    SummaryComponent: lazyNamed(() => import("./List"), "Summary"),
  },
  Item: {
    Edit: lazySpec(() => import("./Edit")),
    hasMain: true,

    createResource: () => {
      return CollectorExporter.create({
        apiVersion: "enterprise/v1",
        kind: "CollectorExporter",
        metadata: {},
        spec: CollectorExporter_Spec.create({
          type: {
            oneofKind: "otlp",
            otlp: CollectorExporter_Spec_OTLP.create({
              auth: {
                type: {
                  oneofKind: "bearer",
                  bearer: {
                    type: {
                      oneofKind: "fromSecret",
                      fromSecret: "",
                    },
                  },
                },
              },
            }),
          },
        }),
        status: {},
      });
    },
  },

  infoItemsGetter: lazyResourceInfo(() => import("./Main")),

  cloneable: true,
};

export default resourceComponentInfo;
