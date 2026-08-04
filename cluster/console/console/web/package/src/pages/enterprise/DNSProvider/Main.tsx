import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceMainInfo } from "@/pages/utils/types";
import { getDNSProviderType } from "./List";

const secretRef = (name?: string) =>
  name ? (
    <ResourceListLabel
      itemRef={ObjectReference.create({
        apiVersion: "enterprise/v1",
        kind: "Secret",
        name,
      })}
    />
  ) : (
    <span className="text-slate-400">Not configured</span>
  );

const copied = (value?: string) =>
  value ? <CopyText value={value} /> : <span className="text-slate-400">Not configured</span>;

export default (_props: { item: EnterpriseP.DNSProvider }) => <></>;

export const MainInfo = (props: {
  item: EnterpriseP.DNSProvider;
}): ResourceMainInfo => {
  const type = props.item.spec?.type;
  const items: NonNullable<ResourceMainInfo["items"]> = [
    {
      label: "Provider type",
      value: <Label>{getDNSProviderType(props.item)}</Label>,
    },
  ];

  if (type?.oneofKind === "cloudflare") {
    items.push(
      { label: "Account email", value: copied(type.cloudflare.email) },
      {
        label: "API Token Secret",
        value: secretRef(
          type.cloudflare.apiToken?.type.oneofKind === "fromSecret"
            ? type.cloudflare.apiToken.type.fromSecret
            : undefined,
        ),
      },
    );
  }

  if (type?.oneofKind === "aws") {
    items.push(
      { label: "Access Key ID", value: copied(type.aws.accessKeyID) },
      { label: "Region", value: type.aws.region || "Not configured" },
      {
        label: "Secret Access Key",
        value: secretRef(
          type.aws.secretAccessKey?.type.oneofKind === "fromSecret"
            ? type.aws.secretAccessKey.type.fromSecret
            : undefined,
        ),
      },
      ...(type.aws.assumeRoleARN
        ? [{ label: "Assume Role ARN", value: copied(type.aws.assumeRoleARN), span: "full" as const }]
        : []),
    );
  }

  if (type?.oneofKind === "digitalocean") {
    items.push({
      label: "API Token Secret",
      value: secretRef(
        type.digitalocean.apiToken?.type.oneofKind === "fromSecret"
          ? type.digitalocean.apiToken.type.fromSecret
          : undefined,
      ),
    });
  }

  if (type?.oneofKind === "google") {
    items.push(
      { label: "Project", value: copied(type.google.project) },
      {
        label: "Service Account Secret",
        value: secretRef(
          type.google.serviceAccount?.type.oneofKind === "fromSecret"
            ? type.google.serviceAccount.type.fromSecret
            : undefined,
        ),
      },
    );
  }

  if (type?.oneofKind === "azure") {
    items.push(
      { label: "Client ID", value: copied(type.azure.clientID) },
      { label: "Tenant ID", value: copied(type.azure.tenantID) },
      { label: "Subscription ID", value: copied(type.azure.subscriptionID) },
      { label: "Resource group", value: type.azure.resourceGroupName || "Not configured" },
      { label: "Cloud", value: type.azure.cloud || "public" },
      {
        label: "Client Secret",
        value: secretRef(
          type.azure.clientSecret?.type.oneofKind === "fromSecret"
            ? type.azure.clientSecret.type.fromSecret
            : undefined,
        ),
      },
    );
  }

  if (type?.oneofKind === "linode") {
    items.push({
      label: "API Token Secret",
      value: secretRef(
        type.linode.apiToken?.type.oneofKind === "fromSecret"
          ? type.linode.apiToken.type.fromSecret
          : undefined,
      ),
    });
  }

  if (type?.oneofKind === "ovh") {
    items.push(
      { label: "Endpoint", value: type.ovh.endpoint || "Not configured" },
      { label: "Application key", value: copied(type.ovh.applicationKey) },
      { label: "Consumer key", value: copied(type.ovh.consumerKey) },
      {
        label: "Application Secret",
        value: secretRef(
          type.ovh.applicationSecret?.type.oneofKind === "fromSecret"
            ? type.ovh.applicationSecret.type.fromSecret
            : undefined,
        ),
      },
    );
  }

  return { items };
};
