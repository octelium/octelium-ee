import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { coreRegionRefOf } from "../utils";
import { RegionEnabledState } from "./List";
import { CountryFlag } from "@/components/SelectCountry";
import { MutedValue } from "@/components/ResourceLayout/overview";
import { MapPin, MapPinned, Power, SquareStack } from "lucide-react";

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
        icon: Power,
        value: <RegionEnabledState item={item} />,
        hint: "Whether the Region hosts Cordium Workspaces",
      },
      {
        label: "Location",
        primary: true,
        icon: MapPin,
        value:
          item.status?.country || item.status?.city ? (
            <span className="inline-flex items-center gap-1.5">
              {item.status.country && <CountryFlag code={item.status.country} />}
              {[item.status.city, item.status.country].filter(Boolean).join(", ")}
            </span>
          ) : (
            <MutedValue>Unknown</MutedValue>
          ),
      },
      {
        label: "Core Region",
        primary: true,
        icon: MapPinned,
        value: <ResourceListLabel itemRef={coreRegionRefOf(name)} />,
      },
      {
        label: "Hosted Workspaces",
        primary: true,
        icon: SquareStack,
        value: (
          <ResourceListLabel to={`/cordium/workspaces?regionRef.name=${name}`}>
            View Workspaces
          </ResourceListLabel>
        ),
      },
    ],
  };
};
