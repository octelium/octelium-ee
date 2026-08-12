import { SecretStore } from "@/apis/enterprisev1/enterprisev1";
import { ResourceComponentInfo } from "@/pages/utils/types";
import { Resource } from "@/utils/pb";

import Edit from "./Edit";
import { LabelComponent, Summary } from "./List";
import Main, { MainInfo } from "./Main";

const resourceComponentInfo: ResourceComponentInfo = {
  API: "enterprise",
  Kind: "SecretStore",
  List: {
    labelComponent: ({ item }: { item: Resource }) => (
      <LabelComponent item={item as SecretStore} />
    ),
    SummaryComponent: Summary,
  },
  Item: {
    Edit: ({ item, onUpdate }) => (
      <Edit
        item={item as SecretStore}
        onUpdate={(next) => onUpdate(next)}
      />
    ),
    Main: ({ item }) => <Main item={item as SecretStore} />,
  },
  unCreatable: true,
  unDeletable: true,
  readOnlyEdit: true,

  infoItemsGetter: ({ item }) => MainInfo({ item: item as SecretStore }),
};

export default resourceComponentInfo;
