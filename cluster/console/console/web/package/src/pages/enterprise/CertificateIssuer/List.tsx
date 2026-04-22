import { CertificateIssuer } from "@/apis/enterprisev1/enterprisev1";
import { ResourceListLabel } from "@/components/ResourceList";

import { getDomain } from "@/utils";
import { match } from "ts-pattern";

export const getType = (svc: CertificateIssuer): string => {
  return match(svc.spec?.type.oneofKind)
    .with("acme", () => "ACME")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: CertificateIssuer; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: CertificateIssuer }) => {
  const { item } = props;

  return (
    <div className="w-full mt-1 flex flex-row">
      <ResourceListLabel label="Type">{getType(item)}</ResourceListLabel>
    </div>
  );
};

export const ExtraComponent = (props: { item: CertificateIssuer }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};
