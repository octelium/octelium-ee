import { Secret } from "@/apis/corev1/corev1";

import { getDomain } from "@/utils";

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
