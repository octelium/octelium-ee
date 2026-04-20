import { ResourceComponentInfo } from "@/pages/utils/types";

import {
  CollectorExporter,
  CollectorExporter_Spec_OTLP,
} from "@/apis/enterprisev1/enterprisev1";
import Edit from "./Edit";
import { ExtraComponent, LabelComponent } from "./List";
import Main, { MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "CollectorExporter",
  List: {
    // @ts-ignore
    labelComponent: LabelComponent,
    // @ts-ignore
    extraComponent: ExtraComponent,
  },
  Item: {
    // @ts-ignore
    Edit: Edit,
    // @ts-ignore
    Main: Main,

    createResource: () => {
      return CollectorExporter.create({
        apiVersion: "enterprise/v1",
        kind: "CollectorExporter",
        metadata: {},
        spec: {
          type: {
            oneofKind: "otlp",
            otlp: {
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
            } as CollectorExporter_Spec_OTLP,
          },
        },
        status: {},
      });
    },
  },

  // @ts-ignore
  infoItemsGetter: MainInfo,
};

export default resourceComponentInfo;
