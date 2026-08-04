import { DNSProvider } from "@/apis/enterprisev1/enterprisev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import { match } from "ts-pattern";

export const getDNSProviderType = (item: DNSProvider): string =>
  match(item.spec?.type.oneofKind)
    .with("cloudflare", () => "Cloudflare")
    .with("aws", () => "AWS Route 53")
    .with("digitalocean", () => "DigitalOcean")
    .with("google", () => "Google Cloud DNS")
    .with("azure", () => "Azure DNS")
    .with("linode", () => "Linode")
    .with("ovh", () => "OVH")
    .otherwise(() => "Not configured");

const getDNSProviderContext = (item: DNSProvider) =>
  match(item.spec?.type)
    .with({ oneofKind: "cloudflare" }, ({ cloudflare }) =>
      cloudflare.email
        ? { label: "Account", value: cloudflare.email }
        : undefined,
    )
    .with({ oneofKind: "aws" }, ({ aws }) =>
      aws.region ? { label: "Region", value: aws.region } : undefined,
    )
    .with({ oneofKind: "google" }, ({ google }) =>
      google.project ? { label: "Project", value: google.project } : undefined,
    )
    .with({ oneofKind: "azure" }, ({ azure }) => {
      const value = [azure.cloud || "public", azure.resourceGroupName]
        .filter(Boolean)
        .join(" · ");
      return value ? { label: "Environment", value } : undefined;
    })
    .with({ oneofKind: "ovh" }, ({ ovh }) =>
      ovh.endpoint ? { label: "Endpoint", value: ovh.endpoint } : undefined,
    )
    .otherwise(() => undefined);

export const LabelComponent = (props: { item: DNSProvider }) => {
  const context = getDNSProviderContext(props.item);

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Type">
        {getDNSProviderType(props.item)}
      </ResourceListLabel>
      {context && (
        <ResourceListLabel label={context.label}>
          {context.value}
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};
