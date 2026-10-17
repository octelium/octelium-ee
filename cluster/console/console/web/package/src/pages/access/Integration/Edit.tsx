import * as AccessP from "@/apis/accessv1/accessv1";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { twMerge } from "@/lib/utils";
import { Collapse, Switch, TextInput } from "@mantine/core";
import {
  AlertTriangle,
  Check,
  ChevronRight,
  KeyRound,
  LucideIcon,
  MessageSquareText,
  Settings2,
  UsersRound,
  Vote,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { Callout, ProviderKind, ProviderMark } from "../common";

type IntegrationType = AccessP.Integration_Spec["type"];

const PROVIDERS: Array<{
  value: ProviderKind;
  label: string;
  description: string;
  features: string[];
}> = [
  {
    value: "slack",
    label: "Slack",
    description: "Approve or reject from a channel or DMs.",
    features: ["Channel & DMs", "Buttons", "Slash command"],
  },
  {
    value: "jira",
    label: "Jira",
    description: "One issue per request, decided by its status.",
    features: ["Issues", "Status transitions"],
  },
  {
    value: "webhook",
    label: "Webhook",
    description: "Signed JSON to any system you run.",
    features: ["Any system", "Signed both ways"],
  },
];

const createType = (kind: IntegrationType["oneofKind"]): IntegrationType => {
  if (kind === "jira") {
    return {
      oneofKind: "jira",
      jira: AccessP.Integration_Spec_Jira.create({
        apiToken: { type: { oneofKind: "fromSecret", fromSecret: "" } },
      }),
    };
  }
  if (kind === "webhook") {
    return {
      oneofKind: "webhook",
      webhook: AccessP.Integration_Spec_Webhook.create({
        signingSecret: { type: { oneofKind: "fromSecret", fromSecret: "" } },
      }),
    };
  }
  return {
    oneofKind: "slack",
    slack: AccessP.Integration_Spec_Slack.create({
      botToken: { type: { oneofKind: "fromSecret", fromSecret: "" } },
      signingSecret: { type: { oneofKind: "fromSecret", fromSecret: "" } },
    }),
  };
};

const cloneForEdit = (item: AccessP.Integration) => {
  const next = AccessP.Integration.clone(item);
  if (!next.spec) {
    next.spec = AccessP.Integration_Spec.create();
  }
  if (!next.spec.type.oneofKind) {
    next.spec.type = createType("slack");
  }
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

const isValidEmail = (value: string) =>
  !value || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);

const FormSection = (props: {
  icon: LucideIcon;
  title: string;
  description: string;
  children: React.ReactNode;
}) => {
  const Icon = props.icon;
  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <header className="flex items-start gap-3 border-b border-slate-100 px-4 py-3">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
          <Icon size={14} strokeWidth={2.2} />
        </span>
        <div className="min-w-0">
          <h3 className="text-body font-semibold text-slate-900">
            {props.title}
          </h3>
          <p className="text-xs font-normal leading-5 text-slate-500">
            {props.description}
          </p>
        </div>
      </header>
      <div className="flex flex-col gap-4 px-4 py-4">{props.children}</div>
    </section>
  );
};

const Advanced = (props: { children: React.ReactNode }) => {
  const [open, setOpen] = React.useState(false);
  return (
    <div className="rounded-lg border border-dashed border-slate-200">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        className="flex w-full cursor-pointer items-center gap-1.5 px-3 py-2 text-xs font-semibold text-slate-600 outline-none transition-colors duration-150 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        <ChevronRight
          size={13}
          strokeWidth={2.5}
          className={twMerge(
            "transition-transform duration-150",
            open && "rotate-90",
          )}
        />
        Advanced settings
      </button>
      <Collapse expanded={open}>
        <div className="grid gap-4 px-3 pb-3 md:grid-cols-2">{props.children}</div>
      </Collapse>
    </div>
  );
};

const SecretSelect = (props: {
  label: string;
  description: string;
  required?: boolean;
  value?: string;
  onChange: (name: string) => void;
}) => (
  <SelectResource
    api="access"
    kind="Secret"
    required={props.required}
    clearable={!props.required}
    label={props.label}
    description={props.description}
    defaultValue={props.value}
    onChange={(secret) => props.onChange(secret?.metadata?.name ?? "")}
  />
);

const fromSecretValue = (
  arg?: { type: { oneofKind: "fromSecret"; fromSecret: string } | { oneofKind: undefined } },
) => (arg?.type.oneofKind === "fromSecret" ? arg.type.fromSecret : undefined);

const SecretsHint = () => (
  <p className="text-xs font-normal leading-5 text-slate-500">
    Credentials are never stored in the Integration itself. Pick existing
    access Secrets, or{" "}
    <Link
      to="/access/secrets/create"
      className="font-semibold text-blue-700 underline decoration-blue-300 underline-offset-2 hover:text-blue-800"
    >
      create a Secret
    </Link>{" "}
    first. The Setup guide tab explains where each value comes from.
  </p>
);

const Edit = (props: {
  item: AccessP.Integration;
  onUpdate: (item: AccessP.Integration) => void;
}) => {
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const configurations = React.useRef<Partial<Record<string, IntegrationType>>>({
    [req.spec!.type.oneofKind!]: structuredClone(req.spec!.type),
  });
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;
  const isExisting = !!props.item.metadata?.uid;

  React.useEffect(() => {
    const next = cloneForEdit(props.item);
    setReq(next);
    configurations.current = {
      [next.spec?.type?.oneofKind ?? "slack"]: structuredClone(
        next.spec?.type ?? createType("slack"),
      ),
    };
  }, [itemKey]);

  const updateReq = () => {
    const next = AccessP.Integration.clone(req);
    setReq(next);
    props.onUpdate(AccessP.Integration.clone(next));
  };

  if (!req.spec) return null;
  const type = req.spec.type;

  const changeType = (value: ProviderKind) => {
    if (isExisting) return;
    if (!req.spec) return;
    const currentKind = req.spec.type.oneofKind;
    if (currentKind === value) return;
    if (currentKind) {
      configurations.current[currentKind] = structuredClone(req.spec.type);
    }
    req.spec.type = structuredClone(
      configurations.current[value] ?? createType(value),
    );
    updateReq();
  };

  const emailMatching = !req.spec.identityResolution?.disableEmailDiscovery;

  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-2">
        <div>
          <h3 className="text-body font-semibold text-slate-900">Provider</h3>
          <p className="text-xs font-normal text-slate-500">
            {isExisting
              ? "The provider of an existing Integration cannot be changed. Create a new Integration to use another one."
              : "Where access requests are presented and decided."}
          </p>
        </div>
        <div
          role="radiogroup"
          aria-label="Provider"
          className="grid grid-cols-1 gap-2 sm:grid-cols-3"
        >
          {PROVIDERS.map((provider) => {
            const active = type.oneofKind === provider.value;
            return (
              <button
                key={provider.value}
                type="button"
                role="radio"
                aria-checked={active}
                disabled={isExisting && !active}
                onClick={() => changeType(provider.value)}
                className={twMerge(
                  "relative flex min-w-0 cursor-pointer flex-col gap-2 rounded-xl border bg-white px-3.5 py-3 text-left outline-none transition-[border-color,box-shadow] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-not-allowed disabled:opacity-50",
                  active
                    ? "border-slate-800 shadow-raised ring-1 ring-slate-800"
                    : "border-slate-200 hover:border-slate-300",
                )}
              >
                <span className="flex items-center gap-2">
                  <ProviderMark provider={provider.value} size={18} />
                  <span className="text-body font-semibold text-slate-900">
                    {provider.label}
                  </span>
                  {active && (
                    <span className="ml-auto flex h-4 w-4 items-center justify-center rounded-full bg-slate-800 text-white">
                      <Check size={10} strokeWidth={3} />
                    </span>
                  )}
                </span>
                <span className="text-xs font-normal leading-5 text-slate-500">
                  {provider.description}
                </span>
                <span className="flex flex-wrap gap-1">
                  {provider.features.map((feature) => (
                    <span
                      key={feature}
                      className="rounded-full border border-slate-200 bg-slate-50 px-1.5 py-px text-micro font-medium text-slate-600"
                    >
                      {feature}
                    </span>
                  ))}
                </span>
              </button>
            );
          })}
        </div>
      </section>

      {type.oneofKind === "slack" && (
        <>
          <FormSection
            icon={KeyRound}
            title="Slack app credentials"
            description="From your Slack app: OAuth & Permissions and Basic Information."
          >
            <SecretsHint />
            <div className="grid gap-4 md:grid-cols-2">
              <SecretSelect
                required
                label="Bot token"
                description="Secret holding the Bot User OAuth Token (xoxb-…)."
                value={fromSecretValue(type.slack.botToken)}
                onChange={(name) => {
                  type.slack.botToken = {
                    type: { oneofKind: "fromSecret", fromSecret: name },
                  };
                  updateReq();
                }}
              />
              <SecretSelect
                required
                label="Signing secret"
                description="Secret holding the app's Signing Secret. Verifies every click."
                value={fromSecretValue(type.slack.signingSecret)}
                onChange={(name) => {
                  type.slack.signingSecret = {
                    type: { oneofKind: "fromSecret", fromSecret: name },
                  };
                  updateReq();
                }}
              />
            </div>
          </FormSection>

          <FormSection
            icon={MessageSquareText}
            title="Where requests are posted"
            description="Shared deliveries go to one channel. Reviewer and requester DMs need no channel."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <TextInput
                label="Channel ID"
                description="The review channel, e.g. C05ACCESS42. Invite the bot to it."
                placeholder="C01ABC234DE"
                value={type.slack.channelID}
                error={
                  type.slack.channelID &&
                  !/^[CG][A-Z0-9]{6,}$/.test(type.slack.channelID)
                    ? "Use the channel ID (C…), not its name"
                    : undefined
                }
                onChange={(event) => {
                  type.slack.channelID = event.target.value.trim();
                  updateReq();
                }}
              />
              <TextInput
                label="Mention user group"
                description="Optional. A Slack user group ID (S…) mentioned in every shared message."
                placeholder="S01ABC234DE"
                value={type.slack.mentionUserGroupID}
                onChange={(event) => {
                  type.slack.mentionUserGroupID = event.target.value.trim();
                  updateReq();
                }}
              />
            </div>
            <Advanced>
              <TextInput
                label="Workspace ID"
                description="Discovered from the bot token when empty. Set it to pin the workspace (T…)."
                placeholder="T01ABC234DE"
                value={type.slack.teamID}
                onChange={(event) => {
                  type.slack.teamID = event.target.value.trim();
                  updateReq();
                }}
              />
              <TextInput
                type="url"
                label="API base URL"
                description="Only for a Slack API proxy. Not the Octelium callback URL."
                placeholder="https://slack.com/api"
                value={type.slack.baseURL}
                error={
                  isValidURL(type.slack.baseURL)
                    ? undefined
                    : "Enter a valid HTTP or HTTPS URL"
                }
                onChange={(event) => {
                  type.slack.baseURL = event.target.value;
                  updateReq();
                }}
              />
            </Advanced>
          </FormSection>
        </>
      )}

      {type.oneofKind === "jira" && (
        <>
          <FormSection
            icon={KeyRound}
            title="Jira Cloud account"
            description="Octelium uses this account's API token to create and update issues."
          >
            <SecretsHint />
            <div className="grid gap-4 md:grid-cols-2">
              <TextInput
                required
                type="url"
                label="Site URL"
                description="Your Jira Cloud site."
                placeholder="https://example.atlassian.net"
                value={type.jira.url}
                error={
                  isValidURL(type.jira.url)
                    ? undefined
                    : "Enter a valid HTTP or HTTPS URL"
                }
                onChange={(event) => {
                  type.jira.url = event.target.value.trim();
                  updateReq();
                }}
              />
              <TextInput
                required
                type="email"
                label="Account email"
                description="The Atlassian account that owns the API token."
                placeholder="octelium-bot@example.com"
                value={type.jira.email}
                error={
                  isValidEmail(type.jira.email)
                    ? undefined
                    : "Enter a valid email address"
                }
                onChange={(event) => {
                  type.jira.email = event.target.value.trim();
                  updateReq();
                }}
              />
              <SecretSelect
                required
                label="API token"
                description="Secret holding the account's API token."
                value={fromSecretValue(type.jira.apiToken)}
                onChange={(name) => {
                  type.jira.apiToken = {
                    type: { oneofKind: "fromSecret", fromSecret: name },
                  };
                  updateReq();
                }}
              />
            </div>
          </FormSection>

          <FormSection
            icon={MessageSquareText}
            title="Where issues are created"
            description="Every presented request becomes one issue of this project."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <TextInput
                label="Project key"
                description="For example ACCESS. Required to create issues."
                placeholder="ACCESS"
                value={type.jira.projectKey}
                onChange={(event) => {
                  type.jira.projectKey = event.target.value.trim().toUpperCase();
                  updateReq();
                }}
              />
              <TextInput
                label="Issue type"
                description="Must exist in the project. Task when empty."
                placeholder="Task"
                value={type.jira.issueTypeName}
                onChange={(event) => {
                  type.jira.issueTypeName = event.target.value;
                  updateReq();
                }}
              />
            </div>
          </FormSection>

          <FormSection
            icon={Vote}
            title="Decisions from Jira"
            description="Reviewers decide by moving the issue to one of these statuses. Leave empty to decide in the access portal only."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <TextInput
                label="Approving status"
                description="Moving an issue to it approves the review step."
                placeholder="Approved"
                value={type.jira.approveStatus}
                onChange={(event) => {
                  type.jira.approveStatus = event.target.value;
                  updateReq();
                }}
              />
              <TextInput
                label="Rejecting status"
                description="Moving an issue to it rejects the request."
                placeholder="Declined"
                value={type.jira.rejectStatus}
                error={
                  type.jira.approveStatus &&
                  type.jira.approveStatus.trim().toLowerCase() ===
                    type.jira.rejectStatus.trim().toLowerCase()
                    ? "The approving and the rejecting statuses must differ"
                    : undefined
                }
                onChange={(event) => {
                  type.jira.rejectStatus = event.target.value;
                  updateReq();
                }}
              />
              <SecretSelect
                label="Webhook secret"
                description="Secret shared with the Jira webhook. Required to accept any decision from Jira."
                value={fromSecretValue(type.jira.webhookSecret)}
                onChange={(name) => {
                  type.jira.webhookSecret = name
                    ? { type: { oneofKind: "fromSecret", fromSecret: name } }
                    : undefined;
                  updateReq();
                }}
              />
            </div>
            {(type.jira.approveStatus || type.jira.rejectStatus) &&
              !type.jira.webhookSecret && (
                <Callout tone="warning" icon={AlertTriangle} title="No webhook secret">
                  Decision statuses are set, but without a webhook secret the
                  Cluster rejects every Jira delivery, so moving an issue
                  decides nothing.
                </Callout>
              )}
          </FormSection>
        </>
      )}

      {type.oneofKind === "webhook" && (
        <>
          <FormSection
            icon={MessageSquareText}
            title="Delivery endpoint"
            description="Octelium POSTs a signed JSON document for every presentation change."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <TextInput
                required
                type="url"
                label="Delivery URL"
                description="Your HTTPS endpoint."
                placeholder="https://example.com/octelium/access"
                value={type.webhook.url}
                error={
                  isValidURL(type.webhook.url)
                    ? undefined
                    : "Enter a valid HTTP or HTTPS URL"
                }
                onChange={(event) => {
                  type.webhook.url = event.target.value.trim();
                  updateReq();
                }}
              />
              <TextInput
                label="Name"
                description="Included in every payload so your system can route it."
                placeholder="access-requests"
                value={type.webhook.name}
                onChange={(event) => {
                  type.webhook.name = event.target.value;
                  updateReq();
                }}
              />
            </div>
          </FormSection>

          <FormSection
            icon={KeyRound}
            title="Signatures"
            description="HMAC-SHA256 over {timestamp}:{body}, in the X-Octelium-Signature header."
          >
            <SecretsHint />
            <div className="grid gap-4 md:grid-cols-2">
              <SecretSelect
                required
                label="Signing secret"
                description="Signs the deliveries sent to your endpoint."
                value={fromSecretValue(type.webhook.signingSecret)}
                onChange={(name) => {
                  type.webhook.signingSecret = {
                    type: { oneofKind: "fromSecret", fromSecret: name },
                  };
                  updateReq();
                }}
              />
              <SecretSelect
                label="Inbound secret"
                description="Optional. Lets your system submit decisions and requests back. Calls are rejected without it."
                value={fromSecretValue(type.webhook.inboundSecret)}
                onChange={(name) => {
                  type.webhook.inboundSecret = name
                    ? { type: { oneofKind: "fromSecret", fromSecret: name } }
                    : undefined;
                  updateReq();
                }}
              />
            </div>
          </FormSection>
        </>
      )}

      <FormSection
        icon={UsersRound}
        title="People"
        description="Decisions and DMs need to know which Cluster User a provider account belongs to."
      >
        <Switch
          label="Match provider accounts to Users by email"
          description="The account's email must equal the email of exactly one User. Matches are remembered as linked accounts."
          checked={emailMatching}
          onChange={(event) => {
            const checked = event.currentTarget.checked;
            req.spec!.identityResolution = checked
              ? undefined
              : AccessP.Integration_Spec_IdentityResolution.create({
                  disableEmailDiscovery: true,
                });
            updateReq();
          }}
        />
        {!emailMatching && (
          <Callout tone="warning" icon={AlertTriangle} title="Only already linked accounts work">
            Nobody new can be matched, so new people can neither receive DMs
            nor submit decisions from the provider.
          </Callout>
        )}
      </FormSection>

      <FormSection
        icon={Settings2}
        title="Status"
        description="A disabled Integration keeps its configuration but delivers nothing and rejects every inbound call."
      >
        <Switch
          label="Disable this Integration"
          checked={req.spec.isDisabled}
          onChange={(event) => {
            req.spec!.isDisabled = event.currentTarget.checked;
            updateReq();
          }}
        />
      </FormSection>
    </div>
  );
};

export default Edit;
