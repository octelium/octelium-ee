import { User_Spec_Type } from "@/apis/corev1/corev1";

import { ResourceListLabel } from "@/components/ResourceList";
import { CertificateIssuer } from "@/apis/enterprisev1/enterprisev1";

import { getDomain, toNumOrZero } from "@/utils";
import Label from "@/components/Label";
import { match } from "ts-pattern";

const getType = (svc: CertificateIssuer): string => {
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
