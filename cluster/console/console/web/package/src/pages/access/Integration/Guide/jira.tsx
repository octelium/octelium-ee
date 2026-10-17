import {
  ArrowRightLeft,
  Info,
  ShieldCheck,
  TriangleAlert,
  UserRound,
} from "lucide-react";
import type { GuideContext, GuideDefinition } from ".";
import { ProviderMark } from "../../common";
import {
  Aside,
  Bullet,
  Bullets,
  Code,
  CodeBlock,
  Figure,
  P,
  Path,
  Step,
  Steps,
  Strong,
  Table,
  ValueRow,
} from "./blocks";
import { JiraIssueMock, JiraWebhookMock } from "./mocks";
import {
  ConsoleLink,
  ExternalLink,
  connectionStep,
  identitiesStatus,
  policyStep,
  policyYAML,
  secretsStep,
  testStep,
} from "./shared";

export const getJiraGuide = (ctx: GuideContext): GuideDefinition => {
  const jira =
    ctx.item.spec?.type?.oneofKind === "jira"
      ? ctx.item.spec.type.jira
      : undefined;
  const webhook = ctx.endpoints.find((entry) => entry.key === "webhook");
  const projectKey = jira?.projectKey || "ACCESS";
  const hasStatuses = !!jira?.approveStatus && !!jira?.rejectStatus;
  const hasWebhookSecret =
    jira?.webhookSecret?.type.oneofKind === "fromSecret" &&
    !!jira.webhookSecret.type.fromSecret;
  const site = (jira?.url || "https://your-site.atlassian.net").replace(/\/+$/, "");

  return {
    title: "Review access requests in Jira",
    intro: (
      <>
        Open a Jira issue for every request that reaches a review step, keep
        it up to date as the request moves on, and let reviewers decide by
        moving the issue to an approving or a rejecting status. Octelium
        stays the source of truth: a transition only counts when the Jira
        user who made it is an eligible reviewer.
      </>
    ),
    duration: "About 20 minutes",
    requirements: ["Jira Cloud site admin", "A Jira project"],
    flow: [
      {
        icon: <UserRound size={15} className="text-slate-600" />,
        title: "Someone requests access",
        caption: "A Policy rule with a Jira review surface matches it.",
      },
      {
        icon: <ProviderMark provider="jira" size={15} />,
        title: "Octelium opens an issue",
        caption: `In the ${projectKey} project, with every detail reviewers need.`,
      },
      {
        icon: <ArrowRightLeft size={15} className="text-slate-600" />,
        title: "A reviewer moves the issue",
        caption: `To ${jira?.approveStatus || "Approved"} or ${jira?.rejectStatus || "Declined"}.`,
      },
      {
        icon: <ShieldCheck size={15} className="text-slate-600" />,
        title: "Octelium decides",
        caption: "The webhook carries the transition; Octelium applies it as a review.",
      },
    ],
    steps: [
      {
        id: "account",
        title: "Prepare an automation account and an API token",
        summary:
          "Octelium calls the Jira Cloud REST API (v3) as this account, with HTTP basic authentication.",
        status: "manual",
        body: (
          <>
            <Steps>
              <Step>
                Use a dedicated Atlassian account, for example{" "}
                <Code>octelium-bot@company.com</Code>, so that issues and
                comments are clearly authored by Octelium.
              </Step>
              <Step>
                In the target project, give it the <Strong>Browse projects</Strong>,{" "}
                <Strong>Create issues</Strong>, <Strong>Edit issues</Strong> and{" "}
                <Strong>Add comments</Strong> permissions.
              </Step>
              <Step>
                Give it the global <Strong>Browse users and groups</Strong>{" "}
                permission. Octelium needs it to look people up by email.
              </Step>
              <Step>
                Signed in as that account, create an API token at{" "}
                <ExternalLink href="https://id.atlassian.com/manage-profile/security/api-tokens">
                  Atlassian account → Security → API tokens
                </ExternalLink>{" "}
                and copy it.
              </Step>
            </Steps>
            <Aside icon={Info} title="Jira Cloud only">
              The Integration authenticates with an email and an API token
              against <Code>/rest/api/3</Code>. Jira Data Center personal
              access tokens are not supported.
            </Aside>
          </>
        ),
      },
      secretsStep(ctx, {
        intro: (
          <>
            Store the API token, and a random webhook secret you generate
            yourself, as access Secrets. Jira signs every webhook delivery with
            that secret.
          </>
        ),
        secrets: [
          {
            name:
              jira?.apiToken?.type.oneofKind === "fromSecret" && jira.apiToken.type.fromSecret
                ? jira.apiToken.type.fromSecret
                : "jira-api-token",
            description: "The API token of the automation account.",
            example: "ATATT3xFfGF0…",
          },
          {
            name:
              jira?.webhookSecret?.type.oneofKind === "fromSecret" && jira.webhookSecret.type.fromSecret
                ? jira.webhookSecret.type.fromSecret
                : "jira-webhook-secret",
            description:
              "A random value shared with the Jira webhook. Without it, decisions made in Jira are rejected.",
            example: "openssl rand -hex 32",
          },
        ],
      }),
      {
        id: "workflow",
        title: "Prepare the project and its workflow",
        summary:
          "Reviewers decide by moving the issue, so the workflow needs an approving and a rejecting status.",
        status: jira?.projectKey && hasStatuses ? "done" : "attention",
        statusLabel:
          jira?.projectKey && hasStatuses
            ? `${projectKey} · ${jira.approveStatus} / ${jira.rejectStatus}`
            : !jira?.projectKey
              ? "No project key"
              : "No decision statuses",
        body: (
          <>
            <Steps>
              <Step>
                Pick the project that receives the issues, for example{" "}
                <Code>ACCESS</Code>, and an issue type that exists in it (
                <Code>Task</Code> is used when none is set).
              </Step>
              <Step>
                In{" "}
                <Path items={["Project settings", "Workflows"]} />, add two
                statuses such as <Code>Approved</Code> and{" "}
                <Code>Declined</Code>, each reachable from the initial status
                with a transition.
              </Step>
              <Step>
                Enter the project key and both status names in the
                Configuration tab. Names are compared case-insensitively and
                must differ.
              </Step>
            </Steps>
            <div className="flex max-w-3xl flex-col gap-1.5">
              <ValueRow label="Project key" value={jira?.projectKey} placeholder="Not set: no issue can be created" />
              <ValueRow label="Issue type" value={jira?.issueTypeName || "Task"} />
              <ValueRow label="Approving status" value={jira?.approveStatus} placeholder="Not set: approving from Jira is disabled" />
              <ValueRow label="Rejecting status" value={jira?.rejectStatus} placeholder="Not set: rejecting from Jira is disabled" />
            </div>
            <Aside icon={Info} title="Only the transition counts">
              Octelium reads the status change carried by the webhook event,
              not the current status of the issue. Editing other fields or
              commenting never decides anything.
            </Aside>
          </>
        ),
      },
      connectionStep(ctx, {
        summary:
          "Octelium calls /rest/api/3/myself with the API token and records the site.",
        checks: (
          <P>
            <Strong>Ready</Strong> proves that the email and the API token
            work. It does not prove that the project, the issue type or the
            permissions are right: failed issue creations show up in the
            Activity tab.
          </P>
        ),
      }),
      {
        id: "webhook",
        title: "Create the Jira webhook",
        summary: "Jira tells Octelium about every issue transition in the project.",
        status: !hasWebhookSecret ? "attention" : webhook ? "manual" : "attention",
        statusLabel: !hasWebhookSecret
          ? "No webhook secret"
          : webhook
            ? undefined
            : "Synchronize first",
        body: (
          <>
            <Steps>
              <Step>
                As a Jira admin, open{" "}
                <Path items={["Settings", "System", "WebHooks"]} /> and select{" "}
                <Strong>Create a WebHook</Strong>.
              </Step>
              <Step>
                Paste the URL below, and the same value you stored in the
                webhook Secret as the <Strong>Secret</Strong>.
              </Step>
              <Step>
                Under <Strong>Issue related events</Strong>, enter the JQL{" "}
                <Code>project = {projectKey}</Code> and only select{" "}
                <Strong>Issue → updated</Strong>.
              </Step>
              <Step>Save. Leave the webhook enabled.</Step>
            </Steps>
            <div className="flex max-w-3xl flex-col gap-1.5">
              <ValueRow
                label="Jira webhook URL"
                value={webhook?.url}
                placeholder="Available once the Cluster assigns an Integration ID"
              />
              <ValueRow label="JQL filter" value={`project = ${projectKey}`} />
            </div>
            <Figure caption="Jira webhook settings for this Integration.">
              <JiraWebhookMock url={webhook?.url} projectKey={projectKey} />
            </Figure>
            <Aside icon={Info} title="How deliveries are verified">
              Jira adds an <Code>X-Hub-Signature: sha256=…</Code> header, an
              HMAC-SHA256 of the body with your secret. Octelium rejects any
              delivery whose signature does not match.
            </Aside>
          </>
        ),
      },
      {
        id: "identities",
        title: "Make sure Jira users map to Octelium Users",
        summary:
          "A transition only counts when its Jira user resolves to an eligible Octelium User.",
        status: identitiesStatus(ctx),
        statusLabel:
          ctx.identities.count > 0
            ? `${ctx.identities.count} linked account${ctx.identities.count === 1 ? "" : "s"}`
            : undefined,
        body: (
          <>
            <P>
              Octelium matches the email that Jira reports for the user who
              made the transition with the <Code>spec.email</Code> of exactly
              one Octelium User.
            </P>
            <Bullets>
              <Bullet>
                Atlassian only reveals emails it is allowed to: accounts
                managed by your organization (verified domain) work out of the
                box; other people must set their profile email visibility to{" "}
                <Strong>Anyone</Strong>.
              </Bullet>
              <Bullet>
                The automation account needs <Strong>Browse users and groups</Strong>.
              </Bullet>
              <Bullet>
                Test one person with <Strong>Resolve identity</Strong> in the
                header, using their account ID from the profile URL{" "}
                <Code>{site}/jira/people/&lt;account ID&gt;</Code>.
              </Bullet>
            </Bullets>
          </>
        ),
      },
      policyStep(ctx, {
        intro: (
          <>
            Add this Integration as an <Code>INTERACTIVE</Code> review surface
            with the <Code>SHARED</Code> audience: the issue lives in the
            project. Jira cannot deliver to individual people, so the
            Reviewers, Requester and Subject audiences are not available.
          </>
        ),
        yaml: policyYAML({
          integration: ctx.name,
          reviewAudience: "SHARED",
        }),
        notes: (
          <Table
            head={["Field", "Effect in Jira"]}
            rows={[
              [<Code>interactionMode: INTERACTIVE</Code>, "Transitions to the decision statuses are applied as reviews."],
              [<Code>interactionMode: DEEP_LINK_ONLY</Code>, "The issue is informational; decisions are made in the access portal."],
              ["Outcome notifications", "Open an issue that records the outcome, in the same project."],
            ]}
          />
        ),
      }),
      testStep(ctx, {
        body: (
          <>
            <Steps>
              <Step>
                Request a resource matched by your Policy from the{" "}
                <ExternalLink href={ctx.portalURL}>access portal</ExternalLink>.
              </Step>
              <Step>
                An issue named{" "}
                <Code>Octelium access Request … by … for …</Code> appears in{" "}
                <Code>{projectKey}</Code>.
              </Step>
              <Step>
                As an eligible reviewer, move it to{" "}
                <Code>{jira?.approveStatus || "Approved"}</Code>. The request
                moves on in{" "}
                <ConsoleLink to="/access/requests">Access → Requests</ConsoleLink>;
                when it is closed, Octelium updates the issue and adds a final
                comment.
              </Step>
            </Steps>
            <Figure caption="A request presented as a Jira issue. Moving it to the approving status approves the review step.">
              <JiraIssueMock
                projectKey={jira?.projectKey}
                approveStatus={jira?.approveStatus}
                issueType={jira?.issueTypeName}
              />
            </Figure>
          </>
        ),
      }),
    ],
    faq: [
      {
        question: "No issue is created",
        answer: (
          <P>
            Open the Activity tab: a failing delivery shows the HTTP status
            returned by Jira. <Code>400</Code> usually means a wrong issue type
            or a required field in the project; <Code>401</Code> a wrong email
            or token; <Code>403</Code> or <Code>404</Code> a missing project
            permission or project key.
          </P>
        ),
      },
      {
        question: "Moving the issue does not approve the request",
        answer: (
          <Bullets>
            <Bullet>The webhook exists, is enabled, listens to <Strong>Issue → updated</Strong> and its JQL includes the project.</Bullet>
            <Bullet>The webhook secret equals the value of the access Secret.</Bullet>
            <Bullet>The status name equals the approving status of the Integration.</Bullet>
            <Bullet>The surface is <Code>INTERACTIVE</Code> and the request is still at that step.</Bullet>
            <Bullet>The Jira user resolves to an eligible reviewer who is not the requester or the subject.</Bullet>
          </Bullets>
        ),
      },
      {
        question: "A Jira user cannot be resolved",
        answer: (
          <P>
            Their email is probably hidden by their Atlassian profile
            visibility, or the automation account lacks{" "}
            <Strong>Browse users and groups</Strong>. Resolve identity shows the
            exact reason.
          </P>
        ),
      },
      {
        question: "Probe the webhook URL",
        answer: (
          <>
            <P>An unsigned POST must be rejected with 401.</P>
            <CodeBlock
              language="sh"
              code={`curl -sS -o /dev/null -w '%{http_code}\\n' -H 'Content-Type: application/json' -d '{}' '${webhook?.url ?? "<Jira webhook URL>"}'`}
            />
            <Aside icon={TriangleAlert} title="Only a real transition is a full test" tone="warning">
              The probe does not check the secret, the statuses or the
              reviewer mapping.
            </Aside>
          </>
        ),
      },
    ],
  };
};
