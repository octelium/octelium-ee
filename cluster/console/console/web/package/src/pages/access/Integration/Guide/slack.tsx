import {
  BellRing,
  Hash,
  Info,
  MousePointerClick,
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
import {
  SlackInteractivityMock,
  SlackMessageMock,
  SlackScopesMock,
} from "./mocks";
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

export const getSlackGuide = (ctx: GuideContext): GuideDefinition => {
  const slack =
    ctx.item.spec?.type?.oneofKind === "slack"
      ? ctx.item.spec.type.slack
      : undefined;
  const interactions = ctx.endpoints.find((entry) => entry.key === "interactions");
  const commands = ctx.endpoints.find((entry) => entry.key === "commands");
  const events = ctx.endpoints.find((entry) => entry.key === "events");

  return {
    title: "Review access requests in Slack",
    intro: (
      <>
        Present access requests in a Slack channel or in direct messages, let
        reviewers approve or reject them with a click, notify people of the
        outcome, and optionally request access with a slash command. Octelium
        stays the source of truth: every click is authorized exactly like a
        decision made in the access portal.
      </>
    ),
    duration: "About 15 minutes",
    requirements: ["Slack workspace admin", "Access to this Console"],
    flow: [
      {
        icon: <UserRound size={15} className="text-slate-600" />,
        title: "Someone requests access",
        caption: "From the access portal, the CLI or a Slack slash command.",
      },
      {
        icon: <ShieldCheck size={15} className="text-slate-600" />,
        title: "A Policy rule matches",
        caption: "Its review step lists this Integration as a surface.",
      },
      {
        icon: <ProviderMark provider="slack" size={15} />,
        title: "Slack shows the request",
        caption: "In the review channel or as a DM to each eligible reviewer.",
      },
      {
        icon: <MousePointerClick size={15} className="text-slate-600" />,
        title: "A reviewer clicks Approve",
        caption: "Octelium checks they may review, then grants the access.",
      },
    ],
    steps: [
      {
        id: "app",
        title: "Create a Slack app",
        summary: "One app can serve every Octelium Integration of a workspace.",
        status: "manual",
        body: (
          <>
            <Steps>
              <Step>
                Open{" "}
                <ExternalLink href="https://api.slack.com/apps">
                  Your Slack apps
                </ExternalLink>{" "}
                and select <Strong>Create New App</Strong> →{" "}
                <Strong>From scratch</Strong>.
              </Step>
              <Step>
                Name it, for example <Code>Octelium Access</Code>, pick your
                workspace and create it.
              </Step>
              <Step>
                Open <Path items={["Settings", "Socket Mode"]} /> and make sure{" "}
                <Strong>Enable Socket Mode</Strong> is off. Octelium receives
                signed HTTPS callbacks, and Slack hides the Request URL fields
                while Socket Mode is on.
              </Step>
            </Steps>
          </>
        ),
      },
      {
        id: "scopes",
        title: "Add bot scopes and install the app",
        summary:
          "The scopes decide which features work: channel posts, DMs, slash commands and matching people by email.",
        status: "manual",
        body: (
          <>
            <P>
              In <Path items={["OAuth & Permissions", "Bot Token Scopes"]} />{" "}
              add:
            </P>
            <Table
              head={["Scope", "Needed for"]}
              rows={[
                [<Code>chat:write</Code>, "Posting and updating request messages. Always required."],
                [<Code>users:read</Code>, "Matching Slack members to Octelium Users."],
                [<Code>users:read.email</Code>, "Reading the email used for that match. Required together with users:read."],
                [<Code>im:write</Code>, "Direct messages to reviewers, the requester or the subject."],
                [<Code>commands</Code>, "The optional request slash command."],
              ]}
            />
            <Figure caption="The five bot scopes and the Bot User OAuth Token in the Slack app settings.">
              <SlackScopesMock />
            </Figure>
            <Steps>
              <Step>
                Select <Strong>Install to Workspace</Strong> at the top of{" "}
                <Strong>OAuth &amp; Permissions</Strong>. If you add scopes
                later, use <Strong>Reinstall to Workspace</Strong>: new scopes
                are not granted to an already installed token.
              </Step>
              <Step>
                Copy the <Strong>Bot User OAuth Token</Strong> (it starts with{" "}
                <Code>xoxb-</Code>).
              </Step>
              <Step>
                Open <Path items={["Basic Information", "App Credentials"]} />{" "}
                and copy the <Strong>Signing Secret</Strong>. It is not the
                Client Secret nor the legacy Verification Token.
              </Step>
            </Steps>
          </>
        ),
      },
      secretsStep(ctx, {
        intro: (
          <>
            Store both values as access Secrets and select them in the
            Integration&apos;s <Strong>Configuration</Strong> tab.
          </>
        ),
        secrets: [
          {
            name: slack?.botToken?.type.oneofKind === "fromSecret" && slack.botToken.type.fromSecret
              ? slack.botToken.type.fromSecret
              : "slack-bot-token",
            description: "The Bot User OAuth Token. Octelium uses it to call the Slack Web API.",
            example: "xoxb-1234-5678-abcd",
          },
          {
            name: slack?.signingSecret?.type.oneofKind === "fromSecret" && slack.signingSecret.type.fromSecret
              ? slack.signingSecret.type.fromSecret
              : "slack-signing-secret",
            description: "The Signing Secret. Octelium uses it to verify every callback that claims to come from Slack.",
            example: "8f742231b10e8888abcd99yyyzzz85a5",
          },
        ],
      }),
      {
        id: "channel",
        title: "Invite the bot to the review channel",
        summary:
          "Shared deliveries go to one channel. Direct messages do not need a channel.",
        status: slack?.channelID ? "done" : "attention",
        statusLabel: slack?.channelID ? slack.channelID : "No channel ID",
        body: (
          <>
            <Steps>
              <Step>
                Create or pick a channel, for example{" "}
                <Code>#access-reviews</Code>, and invite the bot:
              </Step>
            </Steps>
            <CodeBlock code="/invite @Octelium Access" language="slack" />
            <Steps>
              <Step>
                Open the channel details and copy its <Strong>Channel ID</Strong>{" "}
                (it starts with <Code>C</Code>). In a channel link such as{" "}
                <Code>https://app.slack.com/client/T0123/C0456</Code> it is the
                last part.
              </Step>
              <Step>
                Paste it in <Strong>Channel ID</Strong> on the Configuration
                tab. Private channels work too, as long as the bot was invited.
              </Step>
            </Steps>
            <ValueRow
              label="Configured channel ID"
              value={slack?.channelID}
              placeholder="Not set: shared channel deliveries are disabled"
            />
            <Aside icon={Info} title="Optional: mention a user group">
              Set <Strong>Mention user group ID</Strong> (it starts with{" "}
              <Code>S</Code>) to ping a Slack user group such as{" "}
              <Code>@sre-oncall</Code> in every shared message. Mentions only
              notify people: who may approve is always decided by the Policy.
            </Aside>
          </>
        ),
      },
      connectionStep(ctx, {
        summary:
          "Octelium calls Slack's auth.test with the bot token and records the workspace.",
        checks: (
          <P>
            <Strong>Ready</Strong> proves that the bot token works. It does not
            prove that the channel ID, the signing secret or the callback URL
            are right: the end-to-end test below does.
          </P>
        ),
      }),
      {
        id: "interactivity",
        title: "Turn on interactivity",
        summary:
          "Point the Slack app at this Integration so button clicks reach Octelium.",
        status: interactions ? "manual" : "attention",
        statusLabel: interactions ? undefined : "Synchronize first",
        body: (
          <>
            <Steps>
              <Step>
                In the same Slack app, open{" "}
                <Path items={["Interactivity & Shortcuts"]} /> and turn{" "}
                <Strong>Interactivity</Strong> on.
              </Step>
              <Step>
                Paste the URL below in <Strong>Request URL</Strong> and save.
                Leave <Strong>Options Load URL</Strong> empty.
              </Step>
            </Steps>
            <ValueRow
              label="Interactivity request URL"
              value={interactions?.url}
              placeholder="Available once the Cluster assigns an Integration ID"
            />
            <Figure caption="Slack app settings with the Request URL of this Integration.">
              <SlackInteractivityMock url={interactions?.url} />
            </Figure>
            <Aside icon={Info} title="Several channels, one Slack app">
              Create one Integration per channel with the same credentials.
              Slack only accepts one Request URL per app, so configure the URL
              of any one of them: Octelium routes every click to the right
              Integration of the same workspace. Keep that Integration enabled.
            </Aside>
          </>
        ),
      },
      {
        id: "command",
        title: "Add a request slash command",
        summary: "Let people request access without leaving Slack.",
        status: "manual",
        optional: true,
        body: (
          <>
            <Steps>
              <Step>
                Open <Path items={["Slash Commands", "Create New Command"]} />.
              </Step>
              <Step>
                Use <Code>/octelium</Code> as the command, the URL below as
                its <Strong>Request URL</Strong> and{" "}
                <Code>request svc:NAME --duration=1h reason</Code> as the
                usage hint. Reinstall the app if Slack asks you to.
              </Step>
            </Steps>
            <ValueRow
              label="Slash command request URL"
              value={commands?.url}
              placeholder="Available once the Cluster assigns an Integration ID"
            />
            <CodeBlock
              language="slack"
              title="Usage"
              code={`/octelium request svc:prod-db --duration=1h --urgency=HIGH Investigating INC-4821
/octelium request cat:production-support --duration=30m Planned maintenance`}
            />
            <Table
              head={["Part", "Meaning"]}
              rows={[
                [<Code>svc:NAME</Code>, "A Service (service: also works)."],
                [<Code>cat:NAME</Code>, "A Catalog (catalog: also works)."],
                [<Code>--duration</Code>, "30m, 1h, 1h30m… Days are not supported here."],
                [<Code>--urgency</Code>, "VERY_LOW, LOW, NORMAL, HIGH, VERY_HIGH or HIGHEST."],
                ["Remaining text", "The justification, up to 1,500 bytes."],
              ]}
            />
            {events && (
              <P>
                Event Subscriptions are not needed. If you enable them anyway,
                use <Code>{events.url}</Code>.
              </P>
            )}
          </>
        ),
      },
      {
        id: "identities",
        title: "Make sure Slack members map to Octelium Users",
        summary:
          "A click is only accepted from a Slack member that resolves to an eligible Octelium User.",
        status: identitiesStatus(ctx),
        statusLabel:
          ctx.identities.count > 0
            ? `${ctx.identities.count} linked account${ctx.identities.count === 1 ? "" : "s"}`
            : undefined,
        body: (
          <>
            <P>
              Octelium reads the email of the Slack member and matches it with
              the <Code>spec.email</Code> of exactly one Octelium User. The
              link is remembered as a linked account and reused afterwards.
            </P>
            <Bullets>
              <Bullet>
                Every requester and reviewer needs an Octelium User with the
                same email as in Slack, and no other User may share it.
              </Bullet>
              <Bullet>
                The app needs both <Code>users:read</Code> and{" "}
                <Code>users:read.email</Code>.
              </Bullet>
              <Bullet>
                Use <Strong>Resolve identity</Strong> in the header with a
                Slack member ID (<Code>U…</Code>, from the member&apos;s
                profile → <Strong>Copy member ID</Strong>) to check one person.
              </Bullet>
            </Bullets>
            <P>
              Linked accounts are listed in the <Strong>Activity</Strong> tab.
            </P>
          </>
        ),
      },
      policyStep(ctx, {
        intro: (
          <>
            Add this Integration as a <Strong>review surface</Strong> of a
            review step to present pending requests, and as an{" "}
            <Strong>outcome notification</Strong> of a rule to announce the
            result. Use <Code>INTERACTIVE</Code> so reviewers get the buttons.
          </>
        ),
        yaml: policyYAML({
          integration: ctx.name,
          reviewAudience: "SHARED",
          notificationAudience: "REQUESTER",
        }),
        notes: (
          <Table
            head={["Audience", "Where it is delivered", "Allowed in"]}
            rows={[
              [<span className="inline-flex items-center gap-1"><Hash size={12} />Shared</span>, "The Integration's channel", "Review steps and notifications"],
              [<span className="inline-flex items-center gap-1"><ShieldCheck size={12} />Reviewers</span>, "A DM to every eligible reviewer", "Review steps"],
              [<span className="inline-flex items-center gap-1"><BellRing size={12} />Requester</span>, "A DM to whoever asked", "Notifications"],
              [<span className="inline-flex items-center gap-1"><UserRound size={12} />Subject</span>, "A DM to whoever gets the access", "Notifications"],
            ]}
          />
        ),
      }),
      testStep(ctx, {
        body: (
          <>
            <Steps>
              <Step>
                Sign in to the{" "}
                <ExternalLink href={ctx.portalURL}>access portal</ExternalLink>{" "}
                as a requester and request a resource matched by your Policy,
                or use the slash command.
              </Step>
              <Step>
                Check that a message appears in the channel or in the
                reviewers&apos; DMs.
              </Step>
              <Step>
                As another eligible reviewer, click <Strong>Approve</Strong>.
                The requester and the subject cannot review their own request
                unless the step allows it.
              </Step>
              <Step>
                The message updates, the requester is notified, and the request
                shows the decision in{" "}
                <ConsoleLink to="/access/requests">Access → Requests</ConsoleLink>.
              </Step>
            </Steps>
            <Figure caption="What reviewers see. Approving the first step moves the request to the next one.">
              <SlackMessageMock channel={slack?.channelID ? "access-reviews" : undefined} />
            </Figure>
          </>
        ),
      }),
    ],
    faq: [
      {
        question: "Deliveries fail with channel_not_found or not_in_channel",
        answer: (
          <P>
            The channel ID is wrong, belongs to another workspace, or the bot
            was not invited (private channels always need an explicit invite).
            Fix it and wait for the next retry: failed deliveries are retried
            with backoff, see the Activity tab.
          </P>
        ),
      },
      {
        question: "Clicking a button shows a warning or dispatch_failed",
        answer: (
          <Bullets>
            <Bullet>Socket Mode is off in the same app that posted the message.</Bullet>
            <Bullet>Interactivity is on and its Request URL ends in <Code>/slack/interactions</Code> with this Integration&apos;s ID.</Bullet>
            <Bullet>The public host is reachable from the internet without a login redirect.</Bullet>
            <Bullet>The Signing Secret belongs to this app and the Integration is enabled.</Bullet>
          </Bullets>
        ),
      },
      {
        question: "“You are not allowed to review this Request”",
        answer: (
          <P>
            The click reached Octelium, so Slack is configured correctly. The
            Slack member resolved to a User who is not a reviewer of the
            current step, is disabled, or is the requester or the subject while
            separation of duties forbids it. Policy changes only apply to new
            requests: create a fresh one after editing the Policy.
          </P>
        ),
      },
      {
        question: "A person cannot be matched or does not get DMs",
        answer: (
          <P>
            Check both <Code>users:read</Code> scopes, that the app was
            reinstalled after adding them, that the Octelium User email matches
            Slack exactly and is unique, that <Code>im:write</Code> is granted,
            and that email discovery is enabled on the Integration. Use{" "}
            <Strong>Resolve identity</Strong> to see the exact reason.
          </P>
        ),
      },
      {
        question: "The message has no Approve and Reject buttons",
        answer: (
          <P>
            It is either an outcome notification, the surface uses{" "}
            <Code>DEEP_LINK_ONLY</Code>, or the request already left that
            step. Only <Code>INTERACTIVE</Code> review surfaces of the current
            step carry buttons.
          </P>
        ),
      },
      {
        question: "I rotated the bot token or signing secret",
        answer: (
          <P>
            Update the access Secret with the new value, then select{" "}
            <Strong>Synchronize now</Strong> on every Integration that uses it.
            Do not recreate the Integration: that changes its ID and its
            callback URLs.
          </P>
        ),
      },
      {
        question: "Probe the callback URL from a terminal",
        answer: (
          <>
            <P>
              A GET must return <Strong>405</Strong> and an unsigned POST must
              return <Strong>401</Strong>. Anything else points to DNS, TLS or
              routing problems.
            </P>
            <CodeBlock
              language="sh"
              code={`curl -sS -o /dev/null -w '%{http_code}\\n' '${interactions?.url ?? "<interactivity request URL>"}'
curl -sS -o /dev/null -w '%{http_code}\\n' --data-urlencode 'payload={}' '${interactions?.url ?? "<interactivity request URL>"}'`}
            />
            <Aside icon={TriangleAlert} title="Only a real click is a full test" tone="warning">
              These probes do not check the signing secret, the reviewer
              mapping or the Policy.
            </Aside>
          </>
        ),
      },
    ],
  };
};
