import * as AccessP from "@/apis/accessv1/accessv1";
import EditItem from "@/components/EditItem";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { Alert, SegmentedControl, Switch, TextInput } from "@mantine/core";
import { AlertTriangle, MessageSquare, SquareKanban, Webhook } from "lucide-react";
import * as React from "react";

type IntegrationType = AccessP.Integration_Spec["type"];

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

  const changeType = (value: string) => {
    if (isExisting) return;
    if (!req.spec || !["slack", "jira", "webhook"].includes(value)) return;
    const currentKind = req.spec.type.oneofKind;
    if (currentKind) {
      configurations.current[currentKind] = structuredClone(req.spec.type);
    }
    req.spec.type = structuredClone(
      configurations.current[value] ??
        createType(value as IntegrationType["oneofKind"]),
    );
    updateReq();
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="text-xs font-semibold text-slate-700">
            Integration status
          </div>
          <div className="mt-0.5 text-xs font-normal text-slate-500">
            A disabled Integration neither delivers anything nor accepts any
            inbound request.
          </div>
        </div>
        <Switch
          label="Disabled"
          description="Disable every delivery and every inbound request of this Integration."
          checked={req.spec.isDisabled}
          onChange={(event) => {
            req.spec!.isDisabled = event.currentTarget.checked;
            updateReq();
          }}
        />
      </div>

      <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-3">
        <div className="mb-2 text-xs font-semibold text-slate-700">
          Provider
        </div>
        <SegmentedControl
          fullWidth
          disabled={isExisting}
          value={type.oneofKind ?? "slack"}
          onChange={changeType}
          data={[
            { label: "Slack", value: "slack" },
            { label: "Jira", value: "jira" },
            { label: "Webhook", value: "webhook" },
          ]}
        />
        {isExisting && (
          <p className="mt-2 text-micro font-normal leading-4 text-slate-500">
            The provider of an existing Integration cannot be changed. Create a
            new Integration to use a different one.
          </p>
        )}
      </div>

      {type.oneofKind === "slack" && (
        <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 text-body font-semibold text-slate-800">
            <MessageSquare size={15} /> Slack
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <SecretSelect
              required
              label="Bot token Secret"
              description="access Secret containing the bot token (i.e. xoxb-…) of the Slack app."
              value={
                type.slack.botToken?.type.oneofKind === "fromSecret"
                  ? type.slack.botToken.type.fromSecret
                  : undefined
              }
              onChange={(name) => {
                type.slack.botToken = {
                  type: { oneofKind: "fromSecret", fromSecret: name },
                };
                updateReq();
              }}
            />
            <SecretSelect
              required
              label="Signing secret Secret"
              description="access Secret containing the signing secret used to verify the requests that Slack sends to the Cluster."
              value={
                type.slack.signingSecret?.type.oneofKind === "fromSecret"
                  ? type.slack.signingSecret.type.fromSecret
                  : undefined
              }
              onChange={(name) => {
                type.slack.signingSecret = {
                  type: { oneofKind: "fromSecret", fromSecret: name },
                };
                updateReq();
              }}
            />
            <TextInput
              label="Channel ID"
              description="Slack channel (i.e. C…) that the shared presentations are delivered to. Required by the shared Surfaces."
              placeholder="C01ABC234DE"
              value={type.slack.channelID}
              onChange={(event) => {
                type.slack.channelID = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              label="Workspace ID"
              description="Slack workspace (i.e. T…) that this Integration is bound to. It is discovered by the Cluster when it is unset."
              placeholder="T01ABC234DE"
              value={type.slack.teamID}
              onChange={(event) => {
                type.slack.teamID = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              label="Mention user group ID"
              description="Slack user group (i.e. S…) that is mentioned in the shared presentations."
              placeholder="S01ABC234DE"
              value={type.slack.mentionUserGroupID}
              onChange={(event) => {
                type.slack.mentionUserGroupID = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              type="url"
              label="API base URL"
              description="Override the base URL of the Slack API. The public API URL is used when it is unset."
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
          </div>
        </section>
      )}

      {type.oneofKind === "jira" && (
        <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 text-body font-semibold text-slate-800">
            <SquareKanban size={15} /> Jira
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput
              required
              type="url"
              label="Site URL"
              description="Base URL of the Jira site."
              placeholder="https://example.atlassian.net"
              value={type.jira.url}
              error={
                isValidURL(type.jira.url)
                  ? undefined
                  : "Enter a valid HTTP or HTTPS URL"
              }
              onChange={(event) => {
                type.jira.url = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              required
              type="email"
              label="Account email"
              description="Email of the Atlassian account whose API token accesses the Jira REST API."
              placeholder="automation@example.com"
              value={type.jira.email}
              error={
                isValidEmail(type.jira.email)
                  ? undefined
                  : "Enter a valid email address"
              }
              onChange={(event) => {
                type.jira.email = event.target.value;
                updateReq();
              }}
            />
            <SecretSelect
              required
              label="API token Secret"
              description="access Secret containing the API token of the Atlassian account."
              value={
                type.jira.apiToken?.type.oneofKind === "fromSecret"
                  ? type.jira.apiToken.type.fromSecret
                  : undefined
              }
              onChange={(name) => {
                type.jira.apiToken = {
                  type: { oneofKind: "fromSecret", fromSecret: name },
                };
                updateReq();
              }}
            />
            <SecretSelect
              label="Webhook secret Secret"
              description="access Secret containing the shared secret that Jira signs its webhook deliveries with. It is required in order to accept any inbound Jira request."
              value={
                type.jira.webhookSecret?.type.oneofKind === "fromSecret"
                  ? type.jira.webhookSecret.type.fromSecret
                  : undefined
              }
              onChange={(name) => {
                type.jira.webhookSecret = name
                  ? { type: { oneofKind: "fromSecret", fromSecret: name } }
                  : undefined;
                updateReq();
              }}
            />
            <TextInput
              label="Project key"
              description="Jira project that the issues are created in. Required by the shared Surfaces."
              placeholder="ACCESS"
              value={type.jira.projectKey}
              onChange={(event) => {
                type.jira.projectKey = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              label="Issue type"
              description="Jira issue type of the created issues. It defaults to Task."
              placeholder="Task"
              value={type.jira.issueTypeName}
              onChange={(event) => {
                type.jira.issueTypeName = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              label="Approving status"
              description="Jira status which, once an issue transitions to it, submits an approving decision. Only used by the interactive Surfaces."
              placeholder="Approved"
              value={type.jira.approveStatus}
              onChange={(event) => {
                type.jira.approveStatus = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              label="Rejecting status"
              description="Jira status which, once an issue transitions to it, submits a rejecting decision. It must differ from the approving status."
              placeholder="Rejected"
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
          </div>

          {!type.jira.webhookSecret && (
            <Alert
              color="amber"
              icon={<AlertTriangle size={15} />}
              title="Inbound requests are rejected"
            >
              Without a webhook secret the Cluster cannot verify the Jira
              deliveries, so no decision can be submitted from Jira.
            </Alert>
          )}
        </section>
      )}

      {type.oneofKind === "webhook" && (
        <section className="space-y-4 rounded-xl border border-slate-200 bg-white p-4">
          <div className="flex items-center gap-2 text-body font-semibold text-slate-800">
            <Webhook size={15} /> Webhook
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <TextInput
              required
              type="url"
              label="Delivery URL"
              description="URL that the Cluster delivers its presentations to."
              placeholder="https://example.com/octelium/access"
              value={type.webhook.url}
              error={
                isValidURL(type.webhook.url)
                  ? undefined
                  : "Enter a valid HTTP or HTTPS URL"
              }
              onChange={(event) => {
                type.webhook.url = event.target.value;
                updateReq();
              }}
            />
            <TextInput
              label="Name"
              description="Opaque label that is included in the delivered payloads so that the receiving system can route them."
              placeholder="access-requests"
              value={type.webhook.name}
              onChange={(event) => {
                type.webhook.name = event.target.value;
                updateReq();
              }}
            />
            <SecretSelect
              required
              label="Signing secret Secret"
              description="access Secret containing the secret that the Cluster signs its outbound deliveries with."
              value={
                type.webhook.signingSecret?.type.oneofKind === "fromSecret"
                  ? type.webhook.signingSecret.type.fromSecret
                  : undefined
              }
              onChange={(name) => {
                type.webhook.signingSecret = {
                  type: { oneofKind: "fromSecret", fromSecret: name },
                };
                updateReq();
              }}
            />
            <SecretSelect
              label="Inbound secret Secret"
              description="access Secret containing the secret that the external system signs its requests to the Cluster with. It is required in order to accept any inbound request."
              value={
                type.webhook.inboundSecret?.type.oneofKind === "fromSecret"
                  ? type.webhook.inboundSecret.type.fromSecret
                  : undefined
              }
              onChange={(name) => {
                type.webhook.inboundSecret = name
                  ? { type: { oneofKind: "fromSecret", fromSecret: name } }
                  : undefined;
                updateReq();
              }}
            />
          </div>

          {!type.webhook.inboundSecret && (
            <Alert
              color="amber"
              icon={<AlertTriangle size={15} />}
              title="Inbound requests are rejected"
            >
              Without an inbound secret the Cluster cannot verify the requests
              of the external system, so it only delivers to it.
            </Alert>
          )}
        </section>
      )}

      <EditItem
        title="Identity resolution"
        description="How the external actors of this Integration are resolved to the Cluster Users."
        obj={req.spec.identityResolution}
        onUnset={() => {
          req.spec!.identityResolution = undefined;
          updateReq();
        }}
        onSet={() => {
          req.spec!.identityResolution =
            AccessP.Integration_Spec_IdentityResolution.create();
          updateReq();
        }}
      >
        {req.spec.identityResolution && (
          <div className="space-y-3">
            <Switch
              label="Disable email discovery"
              description="Stop resolving an external actor from the email that the provider reports for it."
              checked={req.spec.identityResolution.disableEmailDiscovery}
              onChange={(event) => {
                req.spec!.identityResolution!.disableEmailDiscovery =
                  event.currentTarget.checked;
                updateReq();
              }}
            />
            {req.spec.identityResolution.disableEmailDiscovery && (
              <Alert
                color="amber"
                icon={<AlertTriangle size={15} />}
                title="No new external actor is resolved"
              >
                Email discovery is the only way the Cluster establishes an
                IntegrationIdentity. The already established ones keep working,
                but no new external actor can be resolved, so it can neither be
                delivered to directly nor submit a decision.
              </Alert>
            )}
          </div>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
