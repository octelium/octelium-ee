import { User_Spec_Type } from "@/apis/corev1/corev1";

import { ResourceListLabel } from "@/components/ResourceList";
import { DNSProvider } from "@/apis/enterprisev1/enterprisev1";

import { getDomain, toNumOrZero } from "@/utils";
import Label from "@/components/Label";
import { match } from "ts-pattern";

const getType = (svc: DNSProvider): string => {
  return match(svc.spec?.type.oneofKind)
    .with("cloudflare", () => "Cloudflare")
    .with("aws", () => "AWS Route 53")
    .with("digitalocean", () => "DigitalOcean")
    .with("google", () => "Google")
    .with("azure", () => "Azure")
    .with(undefined, () => "Unset")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: DNSProvider; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: DNSProvider }) => {
  const { item } = props;

  return (
    <div className="w-full mt-1 flex flex-row">
      <ResourceListLabel label="Type">{getType(item)}</ResourceListLabel>
    </div>
  );
};

export const ExtraComponent = (props: { item: DNSProvider }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};
