import { ListPolicyOptions, Policy } from "@/apis/accessv1/accessv1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";

import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import { getDomain } from "@/utils";
import { getClientAccess } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

const ItemDetails = (props: { item: Policy; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Policy }) => {
  const { item } = props;

  return (
    <ResourceListLabelWrap>
      {item.spec!.isDisabled && (
        <ResourceListLabel>
          <span className="text-red-400">Disabled</span>
        </ResourceListLabel>
      )}
      {item.spec!.rules && item.spec!.rules.length > 0 && (
        <ResourceListLabel>{item.spec!.rules.length} Rules</ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

export const ExtraComponent = (props: { item: Policy }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = (props: {
  totalNumber: number;
  totalDisabled: number;
  totalRule: number;
}) => {
  const { totalNumber, totalDisabled, totalRule } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={totalNumber} to={`/access/policies`}>
          Total
        </SummaryItemCount>
        <SummaryItemCount
          count={totalDisabled}
          to={`/access/policies?isDisabled=true`}
        >
          Disabled
        </SummaryItemCount>
        <SummaryItemCount count={totalRule}>Rules</SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = (props: { showNoItems?: boolean }) => {
  const qry = useQuery({
    queryKey: ["visibility", "access", "summary", "Policy"],
    queryFn: async () => {
      const { response } = await getClientAccess().listPolicy(
        ListPolicyOptions.create({}),
      );
      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  const items = qry.data.items;
  const totalNumber = items.length;
  const totalDisabled = items.filter((x) => x.spec?.isDisabled).length;
  const totalRule = items.reduce(
    (acc, x) => acc + (x.spec?.rules.length ?? 0),
    0,
  );

  return (
    <div>
      {totalNumber > 0 && (
        <div>
          <DoSummary
            totalNumber={totalNumber}
            totalDisabled={totalDisabled}
            totalRule={totalRule}
          />
        </div>
      )}
      {totalNumber === 0 && props.showNoItems && <SummaryNoItems />}
    </div>
  );
};
