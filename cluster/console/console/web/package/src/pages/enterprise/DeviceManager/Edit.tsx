import * as CoreP from "@/apis/corev1/corev1";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import Cond from "@/components/Condition";
import { durationSeconds } from "@/components/ResourceLayout/overview";
import { NumberInput, TagsInput, Textarea, TextInput } from "@mantine/core";
import { Filter, KeyRound, Link2, RefreshCcw } from "lucide-react";
import * as React from "react";
import {
  Advanced,
  AdvancedGroup,
  Callout,
  ChoiceGroup,
  countSet,
  DurationField,
  EnumField,
  FormSection,
  Grid,
  KeyValueList,
  SecretField,
  secretName,
  toSecret,
  ToggleList,
  ToggleRow,
} from "../CollectorExporter/fields";
import ProviderPicker from "../CollectorExporter/ProviderPicker";
import {
  CATEGORY_LABEL,
  getProvider,
  ManagerKind,
  ManagerTile,
  PROVIDERS,
} from "./providers";

type ManagerType = EnterpriseP.DeviceManager_Spec["type"];

type Update = () => void;

const DEFAULT_POLL_INTERVAL = 300;
const MIN_POLL_INTERVAL = 30;
const DEFAULT_STALE_AFTER = 3600;

const RESERVED_AGENT_FILTERS = ["siteids", "accountids", "cursor", "limit"];

const createType = (kind: ManagerKind): ManagerType => {
  switch (kind) {
    case "crowdStrike":
      return {
        oneofKind: "crowdStrike",
        crowdStrike: EnterpriseP.DeviceManager_Spec_CrowdStrike.create(),
      };
    case "sentinelOne":
      return {
        oneofKind: "sentinelOne",
        sentinelOne: EnterpriseP.DeviceManager_Spec_SentinelOne.create(),
      };
    case "microsoftIntune":
      return {
        oneofKind: "microsoftIntune",
        microsoftIntune:
          EnterpriseP.DeviceManager_Spec_MicrosoftIntune.create(),
      };
    case "jamf":
      return {
        oneofKind: "jamf",
        jamf: EnterpriseP.DeviceManager_Spec_Jamf.create(),
      };
    case "onePassword":
      return {
        oneofKind: "onePassword",
        onePassword: EnterpriseP.DeviceManager_Spec_OnePassword.create(),
      };
    case "fleetDM":
      return {
        oneofKind: "fleetDM",
        fleetDM: EnterpriseP.DeviceManager_Spec_FleetDM.create(),
      };
    case "huntress":
      return {
        oneofKind: "huntress",
        huntress: EnterpriseP.DeviceManager_Spec_Huntress.create(),
      };
    case "iru":
      return {
        oneofKind: "iru",
        iru: EnterpriseP.DeviceManager_Spec_Iru.create(),
      };
  }
};

const cloneForEdit = (item: EnterpriseP.DeviceManager) => {
  const next = EnterpriseP.DeviceManager.clone(item);
  if (!next.spec) next.spec = EnterpriseP.DeviceManager_Spec.create();
  if (!next.spec.type.oneofKind) next.spec.type = createType("crowdStrike");
  return next;
};

const httpsError = (value: string) => {
  if (!value.trim()) return undefined;
  try {
    const url = new URL(value);
    if (url.protocol !== "https:") return "Use an https:// URL";
    if (!url.host || url.username || url.password) {
      return "Enter a host without credentials";
    }
    if (url.search || url.hash) {
      return "Remove the query and the fragment";
    }
    return undefined;
  } catch {
    return "Enter a full https:// URL";
  }
};

const textError = (value: string) => {
  if (value.length > 256) return "Use 256 characters at most";
  return /^[\x00-\x7F]*$/.test(value) ? undefined : "Use ASCII characters only";
};

const FilterField = (props: {
  label: string;
  description: string;
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
}) => (
  <Textarea
    label={props.label}
    description={props.description}
    placeholder={props.placeholder}
    autosize
    minRows={2}
    maxRows={6}
    maxLength={4096}
    styles={{
      input: { fontFamily: "var(--mantine-font-family-monospace)" },
    }}
    value={props.value}
    onChange={(event) => props.onChange(event.currentTarget.value)}
  />
);

const ListField = (props: {
  label: string;
  description: string;
  placeholder: string;
  value: string[];
  onChange: (value: string[]) => void;
}) => (
  <TagsInput
    label={props.label}
    description={props.description}
    placeholder={props.value.length ? "" : props.placeholder}
    splitChars={[","]}
    acceptValueOnBlur
    maxTags={128}
    value={props.value}
    onChange={props.onChange}
  />
);

const ConnectionSection = (props: {
  description: string;
  children: React.ReactNode;
}) => (
  <FormSection
    icon={KeyRound}
    title="Connection"
    description={props.description}
  >
    {props.children}
  </FormSection>
);

const CrowdStrikeForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_CrowdStrike;
  update: Update;
}) => {
  const { value, update } = props;
  const Region = EnterpriseP.DeviceManager_Spec_CrowdStrike_Region;
  return (
    <>
      <ConnectionSection description="The Falcon API client needs read access to the hosts, and to the Zero Trust Assessment when it is collected.">
        <Grid>
          <TextInput
            required
            label="Client ID"
            placeholder="0123456789abcdef0123456789abcdef"
            value={value.clientID}
            error={textError(value.clientID)}
            onChange={(event) => {
              value.clientID = event.currentTarget.value.trim();
              update();
            }}
          />
          <SecretField
            required
            label="Client secret"
            description="The Secret that holds the secret of the API client."
            value={secretName(value.clientSecret)}
            onChange={(name) => {
              value.clientSecret = toSecret(name);
              update();
            }}
          />
          <EnumField
            label="Falcon cloud"
            description="Discovered automatically when it is not set."
            value={value.region}
            options={[
              { label: "us-1", value: Region.US_1 },
              { label: "us-2", value: Region.US_2 },
              { label: "eu-1", value: Region.EU_1 },
              { label: "us-gov-1", value: Region.US_GOV_1 },
              { label: "us-gov-2", value: Region.US_GOV_2 },
            ]}
            onChange={(next) => {
              value.region = next;
              update();
            }}
          />
        </Grid>
        <ToggleList>
          <ToggleRow
            label="Collect the Zero Trust Assessment"
            description="Collects the scores of the hosts and reads the signed data.zta token on Windows and macOS, which raises the Binding's Assurance to Signed."
            checked={!value.disableZeroTrustAssessment}
            onChange={(checked) => {
              value.disableZeroTrustAssessment = !checked;
              update();
            }}
          />
        </ToggleList>
      </ConnectionSection>

      <Advanced
        summary="Restrict the collected hosts or read a member tenant."
        count={countSet(value.memberCID, value.hostFilter)}
      >
        <AdvancedGroup title="Inventory scope">
          <TextInput
            label="Member CID"
            description="Customer ID of the member tenant in a multi-tenant Falcon environment."
            value={value.memberCID}
            error={textError(value.memberCID)}
            onChange={(event) => {
              value.memberCID = event.currentTarget.value.trim();
              update();
            }}
          />
          <FilterField
            label="Host filter"
            description="An FQL filter that restricts the collected hosts."
            placeholder="platform_name:'Windows'+tags:'FalconGroupingTags/corporate'"
            value={value.hostFilter}
            onChange={(next) => {
              value.hostFilter = next;
              update();
            }}
          />
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const SentinelOneForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_SentinelOne;
  update: Update;
}) => {
  const { value, update } = props;
  const [filters] = React.useState(() =>
    Object.entries(value.agentFilters).map(([key, item]) => ({
      key,
      value: item,
    })),
  );

  return (
    <>
      <ConnectionSection description="The API token needs read access to the agents of the selected sites and accounts.">
        <TextInput
          required
          type="url"
          label="Management URL"
          placeholder="https://example.sentinelone.net"
          value={value.managementURL}
          error={httpsError(value.managementURL)}
          onChange={(event) => {
            value.managementURL = event.currentTarget.value.trim();
            update();
          }}
        />
        <SecretField
          required
          label="API token"
          description="The Secret that holds the SentinelOne API token."
          value={secretName(value.apiToken)}
          onChange={(name) => {
            value.apiToken = toSecret(name);
            update();
          }}
        />
      </ConnectionSection>

      <Advanced
        summary="Restrict the collected agents to sites, accounts or filters."
        count={countSet(value.siteIDs, value.accountIDs, filters)}
      >
        <AdvancedGroup title="Inventory scope">
          <Grid>
            <ListField
              label="Site IDs"
              description="Press Enter or comma after each ID."
              placeholder="225494730938493804"
              value={value.siteIDs}
              onChange={(next) => {
                value.siteIDs = next;
                update();
              }}
            />
            <ListField
              label="Account IDs"
              description="Press Enter or comma after each ID."
              placeholder="225494730938493915"
              value={value.accountIDs}
              onChange={(next) => {
                value.accountIDs = next;
                update();
              }}
            />
          </Grid>
          <KeyValueList
            label="Agent filters"
            description="Additional listing filters of the agents."
            items={filters}
            make={() => ({ key: "", value: "" })}
            addLabel="Add filter"
            keyPlaceholder="isDecommissioned"
            valuePlaceholder="false"
            keyError={(key, index) => {
              const trimmed = key.trim();
              if (!trimmed) return "Required";
              if (RESERVED_AGENT_FILTERS.includes(trimmed.toLowerCase())) {
                return "Reserved";
              }
              return filters.findIndex(
                (item) => item.key.trim() === trimmed,
              ) !== index
                ? "Duplicate"
                : undefined;
            }}
            onUpdate={() => {
              value.agentFilters = Object.fromEntries(
                filters
                  .filter((item) => item.key.trim())
                  .map((item) => [item.key.trim(), item.value]),
              );
              update();
            }}
          />
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const IntuneForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_MicrosoftIntune;
  update: Update;
}) => {
  const { value, update } = props;
  const Cloud = EnterpriseP.DeviceManager_Spec_MicrosoftIntune_Cloud;
  return (
    <>
      <ConnectionSection description="The Entra ID application needs read access to the managed devices in Microsoft Graph.">
        <Grid>
          <TextInput
            required
            label="Tenant ID"
            placeholder="00000000-0000-0000-0000-000000000000"
            value={value.tenantID}
            error={textError(value.tenantID)}
            onChange={(event) => {
              value.tenantID = event.currentTarget.value.trim();
              update();
            }}
          />
          <TextInput
            required
            label="Client ID"
            placeholder="00000000-0000-0000-0000-000000000000"
            value={value.clientID}
            error={textError(value.clientID)}
            onChange={(event) => {
              value.clientID = event.currentTarget.value.trim();
              update();
            }}
          />
          <SecretField
            required
            label="Client secret"
            description="The Secret that holds the secret of the application."
            value={secretName(value.clientSecret)}
            onChange={(name) => {
              value.clientSecret = toSecret(name);
              update();
            }}
          />
          <EnumField
            label="Cloud"
            description="The public cloud is used when it is not set."
            value={value.cloud}
            options={[
              { label: "Public", value: Cloud.PUBLIC },
              { label: "US Government", value: Cloud.US_GOV },
              { label: "China (21Vianet)", value: Cloud.CHINA },
            ]}
            onChange={(next) => {
              value.cloud = next;
              update();
            }}
          />
        </Grid>
      </ConnectionSection>

      <Advanced
        summary="Restrict the collected devices or relax the compliance mapping."
        count={countSet(
          value.filter,
          value.isGracePeriodCompliant,
          value.isConfigManagerCompliant,
        )}
      >
        <AdvancedGroup title="Inventory scope">
          <FilterField
            label="Filter"
            description="A Microsoft Graph $filter expression that restricts the collected managed devices."
            placeholder="operatingSystem eq 'Windows'"
            value={value.filter}
            onChange={(next) => {
              value.filter = next;
              update();
            }}
          />
        </AdvancedGroup>
        <AdvancedGroup title="Compliance">
          <ToggleList>
            <ToggleRow
              label="Treat the grace period as compliant"
              description="Reports the devices in their compliance grace period as compliant instead of in a grace period."
              checked={value.isGracePeriodCompliant}
              onChange={(checked) => {
                value.isGracePeriodCompliant = checked;
                update();
              }}
            />
            <ToggleRow
              label="Treat Configuration Manager as compliant"
              description="Reports the devices whose compliance is evaluated by Configuration Manager as compliant."
              checked={value.isConfigManagerCompliant}
              onChange={(checked) => {
                value.isConfigManagerCompliant = checked;
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const JamfForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_Jamf;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <ConnectionSection description="The Jamf Pro API client needs read access to the computers inventory.">
        <TextInput
          required
          type="url"
          label="Server URL"
          placeholder="https://example.jamfcloud.com"
          value={value.baseURL}
          error={httpsError(value.baseURL)}
          onChange={(event) => {
            value.baseURL = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid>
          <TextInput
            required
            label="Client ID"
            placeholder="00000000-0000-0000-0000-000000000000"
            value={value.clientID}
            error={textError(value.clientID)}
            onChange={(event) => {
              value.clientID = event.currentTarget.value.trim();
              update();
            }}
          />
          <SecretField
            required
            label="Client secret"
            description="The Secret that holds the secret of the API client."
            value={secretName(value.clientSecret)}
            onChange={(name) => {
              value.clientSecret = toSecret(name);
              update();
            }}
          />
        </Grid>
      </ConnectionSection>

      <FormSection
        icon={Filter}
        title="Compliance"
        description="Jamf Pro has no compliance state of its own, so it is derived from membership of smart computer groups."
      >
        <ListField
          label="Compliant groups"
          description="Names of the smart computer groups whose members are compliant. Compliance is not reported when it is empty."
          placeholder="Compliant Macs"
          value={value.compliantGroups}
          onChange={(next) => {
            value.compliantGroups = next.map((name) => name.trim());
            update();
          }}
        />
      </FormSection>

      <Advanced
        summary="Restrict the collected computers."
        count={countSet(value.filter)}
      >
        <AdvancedGroup title="Inventory scope">
          <FilterField
            label="Filter"
            description="A Jamf Pro RSQL filter expression that uses the field names of the computer inventory API of the server."
            placeholder="general.managed==true"
            value={value.filter}
            onChange={(next) => {
              value.filter = next;
              update();
            }}
          />
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const OnePasswordForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_OnePassword;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <ConnectionSection description="The API token of the 1Password Device Trust tenant.">
        <SecretField
          required
          label="API token"
          description="The Secret that holds the 1Password Device Trust API token."
          value={secretName(value.apiToken)}
          onChange={(name) => {
            value.apiToken = toSecret(name);
            update();
          }}
        />
      </ConnectionSection>

      <Advanced
        summary="Use another API endpoint."
        count={countSet(value.baseURL)}
      >
        <AdvancedGroup title="Endpoint">
          <TextInput
            type="url"
            label="API URL"
            description="The default public API is used when it is not set."
            placeholder="https://api.kolide.com"
            value={value.baseURL}
            error={httpsError(value.baseURL)}
            onChange={(event) => {
              value.baseURL = event.currentTarget.value.trim();
              update();
            }}
          />
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

type FleetScope = "all" | "none" | "team";

const FleetDMForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_FleetDM;
  update: Update;
}) => {
  const { value, update } = props;
  const scope: FleetScope =
    value.teamID === undefined ? "all" : value.teamID === 0 ? "none" : "team";

  return (
    <>
      <ConnectionSection description="The API token needs read access to the hosts.">
        <TextInput
          required
          type="url"
          label="Server URL"
          placeholder="https://fleet.example.com"
          value={value.baseURL}
          error={httpsError(value.baseURL)}
          onChange={(event) => {
            value.baseURL = event.currentTarget.value.trim();
            update();
          }}
        />
        <SecretField
          required
          label="API token"
          description="The Secret that holds the Fleet API token."
          value={secretName(value.apiToken)}
          onChange={(name) => {
            value.apiToken = toSecret(name);
            update();
          }}
        />
      </ConnectionSection>

      <FormSection
        icon={Filter}
        title="Inventory scope"
        description="The hosts that are collected from the server."
      >
        <ChoiceGroup<FleetScope>
          label="Hosts"
          value={scope}
          options={[
            { value: "all", label: "Every team" },
            { value: "none", label: "No team", hint: "Hosts without a team." },
            { value: "team", label: "A single team" },
          ]}
          onChange={(next) => {
            value.teamID = next === "all" ? undefined : next === "none" ? 0 : 1;
            update();
          }}
        />
        {scope === "team" && (
          <NumberInput
            required
            label="Team ID"
            min={1}
            allowDecimal={false}
            allowNegative={false}
            value={value.teamID}
            onChange={(next) => {
              value.teamID = Number(next) || 1;
              update();
            }}
          />
        )}
      </FormSection>
    </>
  );
};

const HuntressForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_Huntress;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <ConnectionSection description="The API credentials of the Huntress account.">
        <Grid>
          <TextInput
            required
            label="API key"
            description="The public part of the credentials."
            placeholder="hk_0123456789abcdef"
            value={value.apiKey}
            error={textError(value.apiKey)}
            onChange={(event) => {
              value.apiKey = event.currentTarget.value.trim();
              update();
            }}
          />
          <SecretField
            required
            label="API secret"
            description="The Secret that holds the secret part of the credentials."
            value={secretName(value.apiSecret)}
            onChange={(name) => {
              value.apiSecret = toSecret(name);
              update();
            }}
          />
        </Grid>
      </ConnectionSection>

      <Advanced
        summary="Restrict the collected organizations or use another endpoint."
        count={countSet(value.organizationIDs, value.baseURL)}
      >
        <AdvancedGroup title="Inventory scope">
          <ListField
            label="Organization IDs"
            description="The agents of every organization of the account are collected when it is empty."
            placeholder="1042"
            value={value.organizationIDs.map(String)}
            onChange={(next) => {
              value.organizationIDs = next
                .map(Number)
                .filter((id) => Number.isSafeInteger(id) && id > 0);
              update();
            }}
          />
        </AdvancedGroup>
        <AdvancedGroup title="Endpoint">
          <TextInput
            type="url"
            label="API URL"
            description="The default public API is used when it is not set."
            placeholder="https://api.huntress.io"
            value={value.baseURL}
            error={httpsError(value.baseURL)}
            onChange={(event) => {
              value.baseURL = event.currentTarget.value.trim();
              update();
            }}
          />
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const IruForm = (props: {
  value: EnterpriseP.DeviceManager_Spec_Iru;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <ConnectionSection description="The API URL and the API token of the Iru tenant.">
      <TextInput
        required
        type="url"
        label="API URL"
        placeholder="https://example.api.iru.com"
        value={value.baseURL}
        error={httpsError(value.baseURL)}
        onChange={(event) => {
          value.baseURL = event.currentTarget.value.trim();
          update();
        }}
      />
      <SecretField
        required
        label="API token"
        description="The Secret that holds the Iru API token."
        value={secretName(value.apiToken)}
        onChange={(name) => {
          value.apiToken = toSecret(name);
          update();
        }}
      />
    </ConnectionSection>
  );
};

const TypeForm = (props: { type: ManagerType; update: Update }) => {
  const { type, update } = props;
  switch (type.oneofKind) {
    case "crowdStrike":
      return <CrowdStrikeForm value={type.crowdStrike} update={update} />;
    case "sentinelOne":
      return <SentinelOneForm value={type.sentinelOne} update={update} />;
    case "microsoftIntune":
      return <IntuneForm value={type.microsoftIntune} update={update} />;
    case "jamf":
      return <JamfForm value={type.jamf} update={update} />;
    case "onePassword":
      return <OnePasswordForm value={type.onePassword} update={update} />;
    case "fleetDM":
      return <FleetDMForm value={type.fleetDM} update={update} />;
    case "huntress":
      return <HuntressForm value={type.huntress} update={update} />;
    case "iru":
      return <IruForm value={type.iru} update={update} />;
    default:
      return null;
  }
};

const PollingSection = (props: {
  spec: EnterpriseP.DeviceManager_Spec;
  update: Update;
}) => {
  const { spec, update } = props;
  const polling = spec.polling;
  const edit = (
    mutate: (next: EnterpriseP.DeviceManager_Spec_Polling) => void,
  ) => {
    const next =
      spec.polling ?? EnterpriseP.DeviceManager_Spec_Polling.create();
    mutate(next);
    spec.polling =
      next.isDisabled ||
      next.interval ||
      next.timeout ||
      next.staleAfter ||
      next.maxObservationAge
        ? next
        : undefined;
    update();
  };

  const interval = Math.max(
    durationSeconds(polling?.interval) || DEFAULT_POLL_INTERVAL,
    MIN_POLL_INTERVAL,
  );
  const staleAfter =
    durationSeconds(polling?.staleAfter) || DEFAULT_STALE_AFTER;

  return (
    <FormSection
      icon={RefreshCcw}
      title="Inventory collection"
      description="How often the provider's inventory is read and for how long it stays usable."
    >
      <ToggleList>
        <ToggleRow
          label="Collect the inventory periodically"
          description="A disabled manager keeps its configuration, but no longer reads the provider."
          checked={!polling?.isDisabled}
          onChange={(checked) => edit((next) => (next.isDisabled = !checked))}
        />
      </ToggleList>
      <Grid>
        <DurationField
          label="Interval"
          description="5 minutes by default, and 30 seconds at least."
          value={polling?.interval}
          onChange={(value) => edit((next) => (next.interval = value))}
        />
        <DurationField
          label="Timeout"
          description="Of a single collection. 2 minutes by default."
          value={polling?.timeout}
          onChange={(value) => edit((next) => (next.timeout = value))}
        />
        <DurationField
          label="Stale after"
          description="An older inventory no longer links Devices or refreshes their posture. 1 hour by default."
          value={polling?.staleAfter}
          onChange={(value) => edit((next) => (next.staleAfter = value))}
        />
        <DurationField
          label="Maximum observation age"
          description="The posture of a device that the provider last saw earlier is not valid. 7 days by default."
          value={polling?.maxObservationAge}
          onChange={(value) => edit((next) => (next.maxObservationAge = value))}
        />
      </Grid>
      {staleAfter <= interval && (
        <Callout title="Stale after must be longer than the interval">
          The inventory would become stale before the next collection, so the
          Cluster rejects this configuration.
        </Callout>
      )}
    </FormSection>
  );
};

const LinkingSection = (props: {
  spec: EnterpriseP.DeviceManager_Spec;
  kind?: ManagerKind;
  update: Update;
}) => {
  const { spec, update } = props;
  const provider = getProvider(props.kind);
  const edit = (
    mutate: (next: EnterpriseP.DeviceManager_Spec_Linking) => void,
  ) => {
    const next =
      spec.linking ?? EnterpriseP.DeviceManager_Spec_Linking.create();
    mutate(next);
    spec.linking =
      next.requireOwnerMatch || next.verificationInterval ? next : undefined;
    update();
  };

  return (
    <FormSection
      icon={Link2}
      title="Device linking"
      description="How the Cluster's Devices are bound to the entries of the inventory."
    >
      {provider?.reportsOwners ? (
        <ToggleList>
          <ToggleRow
            label="Require an owner match"
            description="Only binds the entries owned by the Device's User. Other entries fall through to the next DeviceManager."
            checked={!!spec.linking?.requireOwnerMatch}
            onChange={(checked) =>
              edit((next) => (next.requireOwnerMatch = checked))
            }
          />
        </ToggleList>
      ) : (
        <Callout tone="info" title={`${provider?.label} reports no owners`}>
          Devices are bound by their identifiers alone, so the provider cannot
          confirm that an entry belongs to the Device's User.
        </Callout>
      )}
      {provider?.publishesProbes ? (
        <Grid>
          <DurationField
            label="Verification interval"
            description="How often the bound Devices are probed again. A Device that is not verified in time loses its Probe evidence."
            value={spec.linking?.verificationInterval}
            onChange={(value) =>
              edit((next) => (next.verificationInterval = value))
            }
          />
        </Grid>
      ) : (
        <Callout tone="info" title={`${provider?.label} publishes no Probes`}>
          Devices cannot be probed for a provider identifier, so the Bindings
          are not verified periodically.
        </Callout>
      )}
    </FormSection>
  );
};

type ConditionMode = "every" | "matching";

const ConditionSection = (props: {
  spec: EnterpriseP.DeviceManager_Spec;
  update: Update;
}) => {
  const { spec, update } = props;
  return (
    <FormSection
      icon={Filter}
      title="Applies to"
      description="The Devices that this manager can bind. The expression can use ctx.device and ctx.user."
    >
      <ChoiceGroup<ConditionMode>
        label="Devices"
        value={spec.condition ? "matching" : "every"}
        options={[
          { value: "every", label: "Every Device" },
          { value: "matching", label: "Devices that match a condition" },
        ]}
        onChange={(next) => {
          spec.condition =
            next === "every"
              ? undefined
              : CoreP.Condition.create({
                  type: { oneofKind: "match", match: "" },
                });
          update();
        }}
      />
      {spec.condition && (
        <Cond
          item={spec.condition}
          onChange={(value) => {
            spec.condition = value;
            update();
          }}
        />
      )}
    </FormSection>
  );
};

const Edit = (props: {
  item: EnterpriseP.DeviceManager;
  onUpdate: (item: EnterpriseP.DeviceManager) => void;
}) => {
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const configurations = React.useRef<Partial<Record<string, ManagerType>>>({});
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;
  const initialKind = React.useRef(req.spec?.type.oneofKind);

  React.useEffect(() => {
    const next = cloneForEdit(props.item);
    setReq(next);
    configurations.current = {};
    initialKind.current = next.spec?.type.oneofKind;
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.DeviceManager.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.DeviceManager.clone(next));
  };

  if (!req.spec) return null;
  const spec = req.spec;
  const kind = spec.type.oneofKind;

  const changeType = (next: ManagerKind) => {
    if (next === kind) return;
    if (kind) configurations.current[kind] = structuredClone(spec.type);
    spec.type = structuredClone(
      configurations.current[next] ?? createType(next),
    );

    const provider = getProvider(next);
    if (spec.linking && !provider?.reportsOwners) {
      spec.linking.requireOwnerMatch = false;
    }
    if (spec.linking && !provider?.publishesProbes) {
      spec.linking.verificationInterval = undefined;
    }
    if (
      !spec.linking?.requireOwnerMatch &&
      !spec.linking?.verificationInterval
    ) {
      spec.linking = undefined;
    }
    updateReq();
  };

  return (
    <div className="@container flex min-w-0 flex-col gap-4">
      <ProviderPicker
        title="Provider"
        description="The EDR, MDM or device trust platform whose inventory supplies the Devices' posture."
        noun="provider"
        placeholder="Pick the platform that manages your devices."
        providers={PROVIDERS}
        categoryLabels={CATEGORY_LABEL}
        value={kind}
        onChange={changeType}
        tile={(value, size) => <ManagerTile kind={value} size={size} />}
        empty={(query) => <>No provider matches “{query}”.</>}
      />

      {!!props.item.metadata?.uid && kind !== initialKind.current && (
        <Callout tone="info" title="The provider changes when you save">
          The Bindings to the entries that the new provider does not report
          become Lost, and their Devices are selected again.
        </Callout>
      )}

      <TypeForm key={kind} type={spec.type} update={updateReq} />

      <PollingSection spec={spec} update={updateReq} />
      <LinkingSection spec={spec} kind={kind} update={updateReq} />
      <ConditionSection spec={spec} update={updateReq} />
    </div>
  );
};

export default Edit;
