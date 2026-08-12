import { ResourceComponentInfo } from "@/pages/utils/types";
import { Resource } from "@/utils/pb";
import { DNSProvider } from "@/apis/enterprisev1/enterprisev1";

import Edit from "./Edit";
import { LabelComponent, Summary } from "./List";
import Main, { MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "DNSProvider",
  List: {
    labelComponent: ({ item }: { item: Resource }) => (
      <LabelComponent item={item as DNSProvider} />
    ),
    SummaryComponent: Summary,
  },
  Item: {
    Edit: ({ item, onUpdate }) => (
      <Edit
        item={item as DNSProvider}
        onUpdate={(next) => onUpdate(next)}
      />
    ),
    Main: ({ item }) => <Main item={item as DNSProvider} />,
  },

  unCreatable: true,
  unDeletable: true,

  infoItemsGetter: ({ item }) => MainInfo({ item: item as DNSProvider }),
};

export default resourceComponentInfo;
