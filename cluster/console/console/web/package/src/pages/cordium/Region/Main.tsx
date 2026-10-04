import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { coreRegionRefOf } from "../utils";
import { RegionEnabledState } from "./List";

export const MainInfo = (props: {
  item: CordiumP.Region;
}): ResourceMainInfo => {
  const { item } = props;
  const name = item.metadata!.name;

  return {
    items: [
      {
        label: "Workspaces",
        primary: true,
        value: <RegionEnabledState item={item} />,
        hint: "Whether the Region is enabled to host Cordium Workspaces.",
      },
      {
        label: "Core Region",
        value: <ResourceListLabel itemRef={coreRegionRefOf(name)} />,
      },
      {
        label: "Hosted Workspaces",
        value: (
          <ResourceListLabel to={`/cordium/workspaces?regionRef.name=${name}`}>
            View Workspaces
          </ResourceListLabel>
        ),
      },
    ],
  };
};
