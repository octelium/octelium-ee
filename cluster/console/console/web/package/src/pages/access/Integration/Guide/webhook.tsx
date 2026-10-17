import { SegmentedControl } from "@mantine/core";
import { Info, Send, ShieldCheck, UserRound } from "lucide-react";
import * as React from "react";
import type { GuideContext, GuideDefinition } from ".";
import { ProviderMark } from "../../common";
import {
  Aside,
  Bullet,
  Bullets,
  Code,
  CodeBlock,
  P,
  Step,
  Steps,
  Strong,
  Table,
  ValueRow,
} from "./blocks";
import {
  ConsoleLink,
  connectionStep,
  identitiesStatus,
  policyStep,
  policyYAML,
  secretsStep,
  testStep,
} from "./shared";

const VERIFY_SAMPLES: Record<string, string> = {
  "Node.js": `import crypto from "node:crypto";

export const verify = (secret, req, rawBody) => {
  const timestamp = req.headers["x-octelium-timestamp"];
  const signature = req.headers["x-octelium-signature"] ?? "";
  if (Math.abs(Date.now() / 1000 - Number(timestamp)) > 300) return false;

  const expected =
    "sha256=" +
    crypto.createHmac("sha256", secret).update(\`\${timestamp}:\`).update(rawBody).digest("hex");

  return (
    expected.length === signature.length &&
    crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature))
  );
};`,
  Go: `func verify(secret []byte, r *http.Request, body []byte) bool {
	timestamp := r.Header.Get("X-Octelium-Timestamp")
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(timestamp + ":"))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(r.Header.Get("X-Octelium-Signature")))
}`,
  Python: `import hashlib, hmac, time

def verify(secret: bytes, headers, body: bytes) -> bool:
    timestamp = headers.get("X-Octelium-Timestamp", "")
    if not timestamp.isdigit() or abs(time.time() - int(timestamp)) > 300:
        return False

    expected = "sha256=" + hmac.new(
        secret, timestamp.encode() + b":" + body, hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(expected, headers.get("X-Octelium-Signature", ""))`,
};

const VerifySamples = () => {
  const [language, setLanguage] = React.useState("Node.js");
  return (
    <div className="flex max-w-3xl flex-col gap-2">
      <SegmentedControl
        size="xs"
        value={language}
        onChange={setLanguage}
        data={Object.keys(VERIFY_SAMPLES)}
        className="w-fit"
      />
      <CodeBlock
        code={VERIFY_SAMPLES[language]}
        language={language}
        title={`Verify a delivery · ${language}`}
      />
    </div>
  );
};

export const getWebhookGuide = (ctx: GuideContext): GuideDefinition => {
  const webhook =
    ctx.item.spec?.type?.oneofKind === "webhook"
      ? ctx.item.spec.type.webhook
      : undefined;
  const inbound = ctx.endpoints.find((entry) => entry.key === "inbound");
  const hasInboundSecret =
    webhook?.inboundSecret?.type.oneofKind === "fromSecret" &&
    !!webhook.inboundSecret.type.fromSecret;
  const label = webhook?.name || ctx.name;

  return {
    title: "Connect any system with a signed webhook",
    intro: (
      <>
        Send every presentation of a request (created, updated and closed)
        to your own HTTPS endpoint, for example a SIEM, a chat bot or a ticket
        system Octelium does not support natively. Optionally let that system
        submit decisions and create requests back, with signed calls.
      </>
    ),
    duration: "About 30 minutes",
    requirements: ["An HTTPS endpoint you control"],
    flow: [
      {
        icon: <UserRound size={15} className="text-slate-600" />,
        title: "A request needs presenting",
        caption: "A Policy rule lists this Integration as a surface.",
      },
      {
        icon: <Send size={15} className="text-slate-600" />,
        title: "Octelium POSTs JSON",
        caption: "Signed with HMAC-SHA256 and retried with backoff until 2xx.",
      },
      {
        icon: <ProviderMark provider="webhook" size={15} />,
        title: "Your system reacts",
        caption: "Store, route or show the request however you need.",
      },
      {
        icon: <ShieldCheck size={15} className="text-slate-600" />,
        title: "Optionally, decide back",
        caption: "Signed calls to the inbound URL are authorized like any review.",
      },
    ],
    steps: [
      {
        id: "receiver",
        title: "Build the receiving endpoint",
        summary:
          "Accept a JSON POST, verify its signature, answer with any 2xx status.",
        status: "manual",
        body: (
          <>
            <P>
              Octelium sends one POST per change of a presentation. The{" "}
              <Code>type</Code> is <Code>created</Code>, <Code>updated</Code>{" "}
              or <Code>closed</Code>, and <Code>bindingName</Code> identifies
              the presentation for its whole life.
            </P>
            <CodeBlock
              language="json"
              title="Delivery body"
              code={`{
  "type": "created",
  "bindingName": "b-5n2k8q0x7d1m4wzh9c3e",
  "name": "${label}",
  "presentation": {
    "purpose": "REVIEW_SURFACE",
    "requestName": "req-incident-4821",
    "state": "PENDING",
    "requester": "erin (erin@acme.example.com)",
    "subject": "erin (erin@acme.example.com)",
    "resource": "Catalog production-support",
    "urgency": "HIGHEST",
    "justification": "Investigating elevated 5xx rates (INC-4821).",
    "requestedDuration": "2h0m0s",
    "effectiveDuration": "2h0m0s",
    "policyName": "production-access",
    "ruleName": "production-review",
    "stepName": "team-lead",
    "stepIndex": 1,
    "stepCount": 2,
    "approvalSummary": "0 of 1 approval",
    "portalURL": "https://access.octelium.${ctx.domain}/reviewer/requests/req-incident-4821",
    "isActionable": true
  },
  "sentAt": "2026-10-17T10:42:00Z"
}`}
            />
            <Table
              head={["Header", "Value"]}
              rows={[
                [<Code>X-Octelium-Timestamp</Code>, "Unix time in seconds of the delivery."],
                [<Code>X-Octelium-Signature</Code>, <><Code>sha256=</Code> + hex HMAC-SHA256 of <Code>{"{timestamp}:{body}"}</Code> with the signing secret.</>],
              ]}
            />
            <P>
              Reject deliveries older than 5 minutes and compare signatures in
              constant time:
            </P>
            <VerifySamples />
            <Aside icon={Info} title="Retries">
              Any non-2xx answer, timeout (30 s) or redirect counts as a
              failure and is retried with backoff. Make your handler
              idempotent by <Code>bindingName</Code> and <Code>type</Code>.
            </Aside>
          </>
        ),
      },
      secretsStep(ctx, {
        intro: (
          <>
            Generate random secrets (for example with{" "}
            <Code>openssl rand -hex 32</Code>) and store them as access
            Secrets. Share the same values with your system.
          </>
        ),
        secrets: [
          {
            name:
              webhook?.signingSecret?.type.oneofKind === "fromSecret" && webhook.signingSecret.type.fromSecret
                ? webhook.signingSecret.type.fromSecret
                : "webhook-signing-secret",
            description: "Signs the deliveries Octelium sends to your endpoint.",
            example: "9f2c…e81a",
          },
          {
            name:
              webhook?.inboundSecret?.type.oneofKind === "fromSecret" && webhook.inboundSecret.type.fromSecret
                ? webhook.inboundSecret.type.fromSecret
                : "webhook-inbound-secret",
            description:
              "Optional. Signs the calls your system makes to Octelium. Without it, inbound calls are rejected.",
            example: "4b7d…02cf",
          },
        ],
      }),
      {
        id: "destination",
        title: "Point the Integration at your endpoint",
        summary: "Octelium validates the URL when it synchronizes the Integration.",
        status: webhook?.url ? "done" : "attention",
        statusLabel: webhook?.url ? "URL set" : "No URL",
        body: (
          <div className="flex max-w-3xl flex-col gap-1.5">
            <ValueRow label="Delivery URL" value={webhook?.url} placeholder="Not set" />
            <ValueRow label="Name in payloads" value={label} />
          </div>
        ),
      },
      connectionStep(ctx, {
        summary: "Octelium checks that the delivery URL is a valid HTTP(S) URL.",
        checks: (
          <P>
            Synchronization does not call your endpoint. The first delivery
            does: watch the Activity tab for failures.
          </P>
        ),
      }),
      {
        id: "inbound",
        title: "Accept decisions and requests from your system",
        summary:
          "Your system can approve, reject and request access on behalf of its users.",
        status: hasInboundSecret ? "done" : "todo",
        statusLabel: hasInboundSecret ? "Inbound secret set" : undefined,
        optional: true,
        body: (
          <>
            <P>
              Sign the calls exactly like the deliveries, with the inbound
              secret, and POST them to:
            </P>
            <ValueRow
              label="Inbound URL"
              value={inbound?.url}
              placeholder="Available once the Cluster assigns an Integration ID"
            />
            <P>
              People are identified by <Code>externalActorID</Code>, which must
              be the <Strong>email</Strong> of an Octelium User. Octelium links
              it to that User the first time, and applies the same reviewer
              checks as the access portal.
            </P>
            <CodeBlock
              language="json"
              title="Approve or reject the presentation"
              code={`{
  "action": "setReviewDecision",
  "externalActorID": "carol@acme.example.com",
  "eventID": "evt-0193",
  "bindingName": "b-5n2k8q0x7d1m4wzh9c3e",
  "decision": "APPROVE",
  "justification": "Matches INC-4821"
}`}
            />
            <CodeBlock
              language="json"
              title="Request access"
              code={`{
  "action": "createRequest",
  "externalActorID": "erin@acme.example.com",
  "eventID": "evt-0194",
  "resource": "cat:production-support",
  "duration": "2h",
  "urgency": "HIGH",
  "justification": "Investigating INC-4821"
}`}
            />
            <Bullets>
              <Bullet>
                <Code>decision</Code> is <Code>APPROVE</Code> or{" "}
                <Code>REJECT</Code>; an empty one withdraws a previous
                decision.
              </Bullet>
              <Bullet>
                <Code>resource</Code> is <Code>svc:NAME</Code> or{" "}
                <Code>cat:NAME</Code>; <Code>duration</Code> uses Go syntax
                such as <Code>45m</Code> or <Code>2h</Code>.
              </Bullet>
              <Bullet>
                Reuse <Code>eventID</Code> when you retry a decision:
                duplicates are ignored.
              </Bullet>
              <Bullet>
                The answer is JSON:{" "}
                <Code>{`{"isAccepted":true,"message":"…","requestName":"…","portalURL":"…"}`}</Code>
                .
              </Bullet>
            </Bullets>
          </>
        ),
      },
      {
        id: "identities",
        title: "Check linked accounts",
        summary: "Every email seen in an accepted inbound call becomes a linked account.",
        status: hasInboundSecret ? identitiesStatus(ctx) : "todo",
        optional: true,
        statusLabel:
          ctx.identities.count > 0
            ? `${ctx.identities.count} linked account${ctx.identities.count === 1 ? "" : "s"}`
            : undefined,
        body: (
          <P>
            Use <Strong>Resolve identity</Strong> in the header with an email
            to check that it matches exactly one Octelium User.
          </P>
        ),
      },
      policyStep(ctx, {
        intro: (
          <>
            Add this Integration as a review surface to send pending requests,
            or as an outcome notification to stream every final decision, for
            example to a SIEM. Use <Code>INTERACTIVE</Code> only when your
            system submits decisions.
          </>
        ),
        yaml: policyYAML({
          integration: ctx.name,
          reviewAudience: "SHARED",
          notificationAudience: "SHARED",
        }),
        notes: null,
      }),
      testStep(ctx, {
        body: (
          <Steps>
            <Step>
              Create a request matched by your Policy from the access portal.
            </Step>
            <Step>
              Check that your endpoint received a signed <Code>created</Code>{" "}
              delivery, then an <Code>updated</Code> or <Code>closed</Code> one
              as the request moves on.
            </Step>
            <Step>
              Failures, with the status code your endpoint returned, are
              listed in the Activity tab and in{" "}
              <ConsoleLink to="/access">the Access overview</ConsoleLink>.
            </Step>
          </Steps>
        ),
      }),
    ],
    faq: [
      {
        question: "Deliveries fail with a 4xx or 5xx status",
        answer: (
          <P>
            The status comes from your endpoint. Check its logs, the signature
            verification (timestamp first, then <Code>:</Code>, then the raw
            body) and that it answers within 30 seconds without redirects.
          </P>
        ),
      },
      {
        question: "Inbound calls are rejected with 401",
        answer: (
          <P>
            The inbound secret is missing or different, the timestamp is more
            than 5 minutes off, or the signature was computed over a
            re-serialized body instead of the exact bytes sent.
          </P>
        ),
      },
      {
        question: "Inbound decisions are not accepted",
        answer: (
          <P>
            The response <Code>message</Code> explains why: the email does not
            match exactly one User, the User is not a reviewer of the current
            step, or the presentation is not <Code>INTERACTIVE</Code>.
          </P>
        ),
      },
    ],
  };
};
