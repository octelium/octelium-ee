import { Namespace } from "@/apis/corev1/corev1";
import { GetNamespaceSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { SummaryItemCount, SummaryItemCountWrap } from "@/components/Summary";
import { getClientVisibilityCore } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import { NamespaceServicesLabel } from "./Main";

export const LabelComponent = (props: { item: Namespace }) => {
  return <NamespaceServicesLabel item={props.item} />;
};

const DoSummary = (props: { resp: GetNamespaceSummaryResponse }) => {
  const { resp } = props;

  return (
    <div className="w-full">
      <SummaryItemCountWrap>
        <SummaryItemCount count={resp.totalNumber} to={`/core/namespaces`}>
          Total
        </SummaryItemCount>
      </SummaryItemCountWrap>
    </div>
  );
};

export const Summary = () => {
  const qry = useQuery({
    queryKey: ["visibility", "core", "summary", "Namespace"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getNamespaceSummary(
        {},
      );

      return response;
    },
  });
  if (!qry.isSuccess || !qry.data) {
    return <></>;
  }

  return <DoSummary resp={qry.data} />;
};
