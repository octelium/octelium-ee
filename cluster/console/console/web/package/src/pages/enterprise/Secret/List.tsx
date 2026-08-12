import { Secret } from "@/apis/enterprisev1/enterprisev1";
import { GetSecretSummaryResponse } from "@/apis/visibilityv1/enterprise/venterprisev1";
import { SummaryItemCount, SummaryItemCountWrap, SummaryNoItems } from "@/components/Summary";

import { getDomain } from "@/utils";
import { getClientVisibilityEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

const ItemDetails = (props: { item: Secret; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: Secret }) => {
  const { item } = props;

  return <div className="w-full mt-1 flex flex-row"></div>;
};

export const ExtraComponent = (props: { item: Secret }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};

const DoSummary = ({ resp }: { resp: GetSecretSummaryResponse }) => (
  <SummaryItemCountWrap>
    <SummaryItemCount count={resp.totalNumber} to="/enterprise/secrets">Total</SummaryItemCount>
  </SummaryItemCountWrap>
);

export const Summary = ({ showNoItems }: { showNoItems?: boolean }) => {
  const query = useQuery({
    queryKey: ["visibility", "enterprise", "summary", "Secret"],
    queryFn: async () => (await getClientVisibilityEnterprise().getSecretSummary({})).response,
  });
  if (!query.data) return null;
  return query.data.totalNumber > 0 ? <DoSummary resp={query.data} /> : showNoItems ? <SummaryNoItems /> : null;
};
