import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { SegmentedControl, Select, TextInput } from "@mantine/core";
import { Cloud, KeyRound } from "lucide-react";
import * as React from "react";

type ProviderType = EnterpriseP.DNSProvider_Spec["type"];
type ProviderKind = Exclude<ProviderType["oneofKind"], undefined>;

const providerOptions: { label: string; value: ProviderKind }[] = [
  { label: "Cloudflare", value: "cloudflare" },
  { label: "AWS", value: "aws" },
  { label: "DigitalOcean", value: "digitalocean" },
  { label: "Google", value: "google" },
  { label: "Azure", value: "azure" },
  { label: "Linode", value: "linode" },
  { label: "OVH", value: "ovh" },
];

const fromSecret = (name = "") => ({
  type: { oneofKind: "fromSecret" as const, fromSecret: name },
});

const createType = (kind: ProviderKind): ProviderType => {
  switch (kind) {
    case "cloudflare":
      return {
        oneofKind: "cloudflare",
        cloudflare: EnterpriseP.DNSProvider_Spec_Cloudflare.create({
          apiToken: fromSecret(),
        }),
      };
    case "aws":
      return {
        oneofKind: "aws",
        aws: EnterpriseP.DNSProvider_Spec_AWS.create({
          secretAccessKey: fromSecret(),
        }),
      };
    case "digitalocean":
      return {
        oneofKind: "digitalocean",
        digitalocean: EnterpriseP.DNSProvider_Spec_DigitalOcean.create({
          apiToken: fromSecret(),
        }),
      };
    case "google":
      return {
        oneofKind: "google",
        google: EnterpriseP.DNSProvider_Spec_Google.create({
          serviceAccount: fromSecret(),
        }),
      };
    case "azure":
      return {
        oneofKind: "azure",
        azure: EnterpriseP.DNSProvider_Spec_Azure.create({
          clientSecret: fromSecret(),
        }),
      };
    case "linode":
      return {
        oneofKind: "linode",
        linode: EnterpriseP.DNSProvider_Spec_Linode.create({
          apiToken: fromSecret(),
        }),
      };
    case "ovh":
      return {
        oneofKind: "ovh",
        ovh: EnterpriseP.DNSProvider_Spec_OVH.create({
          applicationSecret: fromSecret(),
        }),
      };
  }
};

const cloneForEdit = (item: EnterpriseP.DNSProvider) => {
  const next = EnterpriseP.DNSProvider.clone(item);
  if (!next.spec) next.spec = EnterpriseP.DNSProvider_Spec.create();
  if (!next.spec.type.oneofKind) next.spec.type = createType("cloudflare");
  return next;
};

const secretName = (value?: { type: { oneofKind?: string; fromSecret?: string } }) =>
  value?.type.oneofKind === "fromSecret" ? value.type.fromSecret : undefined;

const isValidEmail = (value: string) =>
  !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);

const Edit = (props: {
  item: EnterpriseP.DNSProvider;
  onUpdate: (item: EnterpriseP.DNSProvider) => void;
}) => {
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const configurations = React.useRef<Partial<Record<ProviderKind, ProviderType>>>({
    [req.spec!.type.oneofKind!]: structuredClone(req.spec!.type),
  });
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;

  React.useEffect(() => {
    const next = cloneForEdit(props.item);
    setReq(next);
    configurations.current = {
      [next.spec!.type.oneofKind!]: structuredClone(next.spec!.type),
    };
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.DNSProvider.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.DNSProvider.clone(next));
  };

  const changeType = (value: string) => {
    if (!req.spec || !providerOptions.some((option) => option.value === value)) return;
    const nextKind = value as ProviderKind;
    const currentKind = req.spec.type.oneofKind;
    if (currentKind) {
      configurations.current[currentKind] = structuredClone(req.spec.type);
    }
    req.spec.type = structuredClone(
      configurations.current[nextKind] ?? createType(nextKind),
    );
    updateReq();
  };

  if (!req.spec) return null;
  const type = req.spec.type;

  const secretPicker = (
    label: string,
    description: string,
    value: string | undefined,
    onChange: (name: string) => void,
  ) => (
    <SelectResource
      api="enterprise"
      kind="Secret"
      required
      label={label}
      description={description}
      defaultValue={value}
      onChange={(secret) => onChange(secret?.metadata?.name ?? "")}
    />
  );

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto rounded-xl border border-slate-200 bg-slate-50/50 p-3">
        <div className="mb-2 flex items-center gap-2 text-[0.72rem] font-bold text-slate-700">
          <Cloud size={14} /> DNS provider
        </div>
        <SegmentedControl
          value={type.oneofKind ?? "cloudflare"}
          onChange={changeType}
          data={providerOptions}
          className="min-w-[660px]"
          fullWidth
        />
      </div>

      <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
        <div className="flex items-center gap-2 text-[0.75rem] font-bold text-slate-800">
          <KeyRound size={15} /> Provider configuration
        </div>

        {type.oneofKind === "cloudflare" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput
              required
              type="email"
              label="Account email"
              description="Email associated with the Cloudflare account."
              placeholder="admin@example.com"
              value={type.cloudflare.email}
              error={isValidEmail(type.cloudflare.email) ? undefined : "Enter a valid email address"}
              onChange={(event) => {
                type.cloudflare.email = event.currentTarget.value;
                updateReq();
              }}
            />
            {secretPicker(
              "API Token Secret",
              "Secret containing a Cloudflare API token with DNS permissions.",
              secretName(type.cloudflare.apiToken),
              (name) => {
                type.cloudflare.apiToken = fromSecret(name);
                updateReq();
              },
            )}
          </div>
        )}

        {type.oneofKind === "aws" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Access Key ID" placeholder="AKIA…" value={type.aws.accessKeyID} onChange={(event) => { type.aws.accessKeyID = event.currentTarget.value; updateReq(); }} />
            {secretPicker("Secret Access Key", "Secret containing the AWS secret access key.", secretName(type.aws.secretAccessKey), (name) => { type.aws.secretAccessKey = fromSecret(name); updateReq(); })}
            <TextInput required label="Region" description="AWS region containing the Route 53 configuration." placeholder="us-east-1" value={type.aws.region} onChange={(event) => { type.aws.region = event.currentTarget.value; updateReq(); }} />
            <TextInput label="Assume Role ARN" description="Optional IAM role assumed for DNS management." placeholder="arn:aws:iam::123456789012:role/dns-manager" value={type.aws.assumeRoleARN} error={type.aws.assumeRoleARN && !/^arn:aws[a-z-]*:iam::\d{12}:role\/.+/.test(type.aws.assumeRoleARN) ? "Enter a valid IAM role ARN" : undefined} onChange={(event) => { type.aws.assumeRoleARN = event.currentTarget.value; updateReq(); }} />
          </div>
        )}

        {type.oneofKind === "digitalocean" &&
          secretPicker("API Token Secret", "Secret containing the DigitalOcean API token.", secretName(type.digitalocean.apiToken), (name) => { type.digitalocean.apiToken = fromSecret(name); updateReq(); })}

        {type.oneofKind === "google" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Project" description="Google Cloud project that owns the DNS zones." placeholder="production-infrastructure" value={type.google.project} onChange={(event) => { type.google.project = event.currentTarget.value; updateReq(); }} />
            {secretPicker("Service Account Secret", "Secret containing the Google service-account JSON credentials.", secretName(type.google.serviceAccount), (name) => { type.google.serviceAccount = fromSecret(name); updateReq(); })}
          </div>
        )}

        {type.oneofKind === "azure" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Client ID" value={type.azure.clientID} onChange={(event) => { type.azure.clientID = event.currentTarget.value; updateReq(); }} />
            {secretPicker("Client Secret", "Secret containing the Azure application client secret.", secretName(type.azure.clientSecret), (name) => { type.azure.clientSecret = fromSecret(name); updateReq(); })}
            <TextInput required label="Tenant ID" value={type.azure.tenantID} onChange={(event) => { type.azure.tenantID = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Subscription ID" value={type.azure.subscriptionID} onChange={(event) => { type.azure.subscriptionID = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Resource group" value={type.azure.resourceGroupName} onChange={(event) => { type.azure.resourceGroupName = event.currentTarget.value; updateReq(); }} />
            <Select label="Azure cloud" description="Azure cloud environment. Empty values use Public Azure." value={type.azure.cloud || "public"} data={[{ label: "Public", value: "public" }, { label: "China", value: "china" }, { label: "US Government", value: "usgovernment" }, { label: "German", value: "german" }]} onChange={(value) => { type.azure.cloud = value === "public" ? "" : value ?? ""; updateReq(); }} />
          </div>
        )}

        {type.oneofKind === "linode" &&
          secretPicker("API Token Secret", "Secret containing the Linode API token.", secretName(type.linode.apiToken), (name) => { type.linode.apiToken = fromSecret(name); updateReq(); })}

        {type.oneofKind === "ovh" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Endpoint" description="OVH API endpoint, such as ovh-eu." placeholder="ovh-eu" value={type.ovh.endpoint} onChange={(event) => { type.ovh.endpoint = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Application key" value={type.ovh.applicationKey} onChange={(event) => { type.ovh.applicationKey = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Consumer key" value={type.ovh.consumerKey} onChange={(event) => { type.ovh.consumerKey = event.currentTarget.value; updateReq(); }} />
            {secretPicker("Application Secret", "Secret containing the OVH application secret.", secretName(type.ovh.applicationSecret), (name) => { type.ovh.applicationSecret = fromSecret(name); updateReq(); })}
          </div>
        )}
      </section>
    </div>
  );
};

export default Edit;
