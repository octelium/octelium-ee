import { ResourceComponentInfo } from "@/pages/utils/types";

import {
  CollectorExporter,
  CollectorExporter_Spec,
  CollectorExporter_Spec_OTLP,
} from "@/apis/enterprisev1/enterprisev1";
import { Resource } from "@/utils/pb";
import Edit from "./Edit";
import { LabelComponent, Summary } from "./List";
import Main, { MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "CollectorExporter",
  List: {
    labelComponent: ({ item }: { item: Resource }) => <LabelComponent item={item as CollectorExporter} />,
    SummaryComponent: Summary,
  },
  Item: {
    Edit: ({ item, onUpdate }) => <Edit item={item as CollectorExporter} onUpdate={(next) => onUpdate(next)} />,
    Main: ({ item }) => <Main item={item as CollectorExporter} />,

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

  infoItemsGetter: ({ item }) => MainInfo({ item: item as CollectorExporter }),

  cloneable: true,
};

export default resourceComponentInfo;
