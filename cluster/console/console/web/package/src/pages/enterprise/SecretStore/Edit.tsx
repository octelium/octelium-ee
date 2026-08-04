import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Alert, SegmentedControl, TextInput } from "@mantine/core";
import { Boxes, Cloud, Info, LockKeyhole } from "lucide-react";
import * as React from "react";

type StoreType = EnterpriseP.SecretStore_Spec["type"];
type StoreKind = Exclude<StoreType["oneofKind"], undefined>;

const storeOptions: { label: string; value: StoreKind }[] = [
  { label: "Kubernetes", value: "kubernetes" },
  { label: "HashiCorp Vault", value: "hashicorpVault" },
  { label: "AWS KMS", value: "awsKeyManagementService" },
  { label: "Azure Key Vault", value: "azureKeyVault" },
  { label: "Google Cloud KMS", value: "googleCloudKeyManagementService" },
];

const createType = (kind: StoreKind): StoreType => {
  switch (kind) {
    case "kubernetes":
      return {
        oneofKind: "kubernetes",
        kubernetes: EnterpriseP.SecretStore_Spec_Kubernetes.create(),
      };
    case "hashicorpVault":
      return {
        oneofKind: "hashicorpVault",
        hashicorpVault: EnterpriseP.SecretStore_Spec_HashicorpVault.create(),
      };
    case "awsKeyManagementService":
      return {
        oneofKind: "awsKeyManagementService",
        awsKeyManagementService:
          EnterpriseP.SecretStore_Spec_AWSKeyManagementService.create(),
      };
    case "azureKeyVault":
      return {
        oneofKind: "azureKeyVault",
        azureKeyVault: EnterpriseP.SecretStore_Spec_AzureKeyVault.create(),
      };
    case "googleCloudKeyManagementService":
      return {
        oneofKind: "googleCloudKeyManagementService",
        googleCloudKeyManagementService:
          EnterpriseP.SecretStore_Spec_GoogleCloudKeyManagementService.create(),
      };
  }
};

const cloneForEdit = (item: EnterpriseP.SecretStore) => {
  const next = EnterpriseP.SecretStore.clone(item);
  if (!next.spec) next.spec = EnterpriseP.SecretStore_Spec.create();
  if (!next.spec.type.oneofKind) next.spec.type = createType("kubernetes");
  return next;
};

const isValidURL = (value: string) => {
  if (!value) return true;
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
};

const Edit = (props: {
  item: EnterpriseP.SecretStore;
  onUpdate: (item: EnterpriseP.SecretStore) => void;
}) => {
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const configurations = React.useRef<Partial<Record<StoreKind, StoreType>>>({
    [req.spec!.type.oneofKind!]: structuredClone(req.spec!.type),
  });
  const itemKey = props.item.metadata?.uid || props.item.metadata?.name;

  React.useEffect(() => {
    const next = cloneForEdit(props.item);
    setReq(next);
    configurations.current = {
      [next.spec!.type.oneofKind!]: structuredClone(next.spec!.type),
    };
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.SecretStore.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.SecretStore.clone(next));
  };

  const changeType = (value: string) => {
    if (!req.spec || !storeOptions.some((option) => option.value === value)) return;
    const nextKind = value as StoreKind;
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

  return (
    <div className="space-y-4">
      <Alert
        color="blue"
        icon={<LockKeyhole size={16} />}
        title="Secret store editing is not currently supported"
      >
        The cluster currently uses Kubernetes-native key storage. The
        configuration is shown below for reference, but cannot be changed from
        the console yet.
      </Alert>

      <fieldset disabled className="space-y-4 opacity-55">
        <div className="overflow-x-auto rounded-xl border border-slate-200 bg-slate-50/50 p-3">
          <div className="mb-2 flex items-center gap-2 text-[0.72rem] font-bold text-slate-700">
            <Cloud size={14} /> Secret store backend
          </div>
          <SegmentedControl
            value={type.oneofKind ?? "kubernetes"}
            onChange={changeType}
            data={storeOptions}
            className="min-w-[650px]"
            fullWidth
          />
        </div>

        <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 text-[0.75rem] font-bold text-slate-800">
            <Boxes size={15} /> Backend configuration
          </div>

        {type.oneofKind === "kubernetes" && (
          <Alert color="blue" icon={<Info size={15} />} title="Kubernetes-native storage">
            This backend uses the cluster’s Kubernetes key storage and requires
            no additional configuration.
          </Alert>
        )}

        {type.oneofKind === "hashicorpVault" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput
              required
              type="url"
              label="Vault address"
              description="HTTP or HTTPS address of the Vault server."
              placeholder="https://vault.example.com"
              value={type.hashicorpVault.address}
              error={isValidURL(type.hashicorpVault.address) ? undefined : "Enter a valid HTTP or HTTPS URL"}
              className="md:col-span-2"
              onChange={(event) => {
                type.hashicorpVault.address = event.currentTarget.value;
                updateReq();
              }}
            />
            <TextInput required label="Role" description="Vault role used by the cluster." placeholder="octelium" value={type.hashicorpVault.role} onChange={(event) => { type.hashicorpVault.role = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Key" description="Vault encryption key name." placeholder="cluster-secrets" value={type.hashicorpVault.key} onChange={(event) => { type.hashicorpVault.key = event.currentTarget.value; updateReq(); }} />
          </div>
        )}

        {type.oneofKind === "awsKeyManagementService" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Key ID" description="KMS key ID, ARN, or alias." placeholder="alias/octelium-secrets" value={type.awsKeyManagementService.keyID} onChange={(event) => { type.awsKeyManagementService.keyID = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Region" placeholder="us-east-1" value={type.awsKeyManagementService.region} onChange={(event) => { type.awsKeyManagementService.region = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Role ARN" description="IAM role assumed to access the KMS key." placeholder="arn:aws:iam::123456789012:role/octelium-kms" value={type.awsKeyManagementService.roleARN} error={type.awsKeyManagementService.roleARN && !/^arn:aws[a-z-]*:iam::\d{12}:role\/.+/.test(type.awsKeyManagementService.roleARN) ? "Enter a valid IAM role ARN" : undefined} className="md:col-span-2" onChange={(event) => { type.awsKeyManagementService.roleARN = event.currentTarget.value; updateReq(); }} />
          </div>
        )}

        {type.oneofKind === "azureKeyVault" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Client ID" placeholder="00000000-0000-0000-0000-000000000000" value={type.azureKeyVault.clientID} onChange={(event) => { type.azureKeyVault.clientID = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Tenant ID" placeholder="00000000-0000-0000-0000-000000000000" value={type.azureKeyVault.tenantID} onChange={(event) => { type.azureKeyVault.tenantID = event.currentTarget.value; updateReq(); }} />
            <TextInput required type="url" label="Vault URL" placeholder="https://example.vault.azure.net" value={type.azureKeyVault.vaultURL} error={isValidURL(type.azureKeyVault.vaultURL) ? undefined : "Enter a valid HTTP or HTTPS URL"} onChange={(event) => { type.azureKeyVault.vaultURL = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Key" description="Azure Key Vault key name." placeholder="octelium-secrets" value={type.azureKeyVault.key} onChange={(event) => { type.azureKeyVault.key = event.currentTarget.value; updateReq(); }} />
          </div>
        )}

        {type.oneofKind === "googleCloudKeyManagementService" && (
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput required label="Project" placeholder="production-infrastructure" value={type.googleCloudKeyManagementService.project} onChange={(event) => { type.googleCloudKeyManagementService.project = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Location" placeholder="global" value={type.googleCloudKeyManagementService.location} onChange={(event) => { type.googleCloudKeyManagementService.location = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Key ring" placeholder="octelium" value={type.googleCloudKeyManagementService.keyRing} onChange={(event) => { type.googleCloudKeyManagementService.keyRing = event.currentTarget.value; updateReq(); }} />
            <TextInput required label="Key" placeholder="cluster-secrets" value={type.googleCloudKeyManagementService.key} onChange={(event) => { type.googleCloudKeyManagementService.key = event.currentTarget.value; updateReq(); }} />
          </div>
        )}
        </section>
      </fieldset>
    </div>
  );
};

export default Edit;
