import { Region } from "@/apis/cordiumv1/cordiumv1";
import {
  GetRegionSummaryResponse,
  ListRegionOptions,
} from "@/apis/visibilityv1/cordium/vcordiumv1";
import Label from "@/components/Label";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getClientVisibilityCordium } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { CircleCheck } from "lucide-react";
import { coreRegionRefOf } from "../utils";

export const useEnabledRegionUIDs = () =>
  useQuery({
    queryKey: ["visibility", "cordium", "enabledRegions"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCordium().listRegion(
        ListRegionOptions.create({
          common: { itemsPerPage: 1000 },
          isEnabled: true,
        }),
      );
      return new Set(response.items.map((itm) => itm.metadata?.uid));
    },
  });

const useIsRegionEnabled = (item: Region) => {
  const qry = useEnabledRegionUIDs();
  return qry.data ? qry.data.has(item.metadata?.uid) : undefined;
};

export const RegionEnabledState = (props: { item: Region }) => {
  const isEnabled = useIsRegionEnabled(props.item);
  if (isEnabled === undefined) return null;

  return isEnabled ? (
    <Label tone="success">Enabled</Label>
  ) : (
    <Label outlined>Disabled</Label>
  );
};

const RegionEnabledLabel = (props: { item: Region }) => {
  const isEnabled = useIsRegionEnabled(props.item);
  if (isEnabled === undefined) return null;

  return (
    <ResourceListLabel label="Workspaces">
      <span className={isEnabled ? "text-emerald-600" : "text-slate-500"}>
        {isEnabled ? "Enabled" : "Disabled"}
      </span>
    </ResourceListLabel>
  );
};

export const LabelComponent = (props: { item: Region }) => {
  const { item } = props;
  const name = item.metadata!.name;

  return (
    <ResourceListLabelWrap>
      <RegionEnabledLabel item={item} />
      <ResourceListLabel label="Core Region" itemRef={coreRegionRefOf(name)} />
      <ResourceListLabel to={`/cordium/workspaces?regionRef.name=${name}`}>
        View Workspaces
      </ResourceListLabel>
    </ResourceListLabelWrap>
  );
};

const DoSummary = ({ resp }: { resp: GetRegionSummaryResponse }) => {
  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to="/cordium/regions">
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={resp.totalEnabled}
          to="/cordium/regions?isEnabled=true"
          icon={CircleCheck}
        >
          Enabled
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "cordium", "summary", "Region"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCordium().getRegionSummary(
        {},
      );
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return (
    <div>
      {qry.data.totalNumber > 0 && <DoSummary resp={qry.data} />}
      {qry.data.totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
