import * as AccessP from "@/apis/accessv1/accessv1";
import TimeAgo from "@/components/TimeAgo";
import { Button } from "@mantine/core";
import {
  ArrowUpRight,
  FilePlus2,
  KeyRound,
  ShieldAlert,
  ShieldCheck,
} from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { RefChip } from "../../common";
import { getSyncStateMeta } from "../utils";
import type { GuideContext, GuideStep, GuideStepStatus } from ".";
import { Aside, Bullet, Bullets, Code, CodeBlock, P, Strong } from "./blocks";

export const ExternalLink = (props: { href: string; children: React.ReactNode }) => (
  <a
    href={props.href}
    target="_blank"
    rel="noreferrer noopener"
    className="inline-flex items-center gap-0.5 font-semibold text-blue-700 underline decoration-blue-300 underline-offset-2 hover:text-blue-800"
  >
    {props.children}
    <ArrowUpRight size={11} strokeWidth={2.5} aria-hidden="true" />
  </a>
);

export const ConsoleLink = (props: {
  to: string;
  children: React.ReactNode;
  state?: unknown;
}) => (
  <Link
    to={props.to}
    state={props.state}
    className="font-semibold text-blue-700 underline decoration-blue-300 underline-offset-2 hover:text-blue-800"
  >
    {props.children}
  </Link>
);

export const secretsStep = (
  ctx: GuideContext,
  opts: {
    secrets: { name: string; description: string; example: string }[];
    intro: React.ReactNode;
  },
): GuideStep => {
  const referenced = ctx.secrets.names;
  const missing = ctx.secrets.missing;
  const status: GuideStepStatus =
    referenced.length === 0
      ? "todo"
      : !ctx.secrets.isLoaded
        ? "todo"
        : missing.length > 0
          ? "attention"
          : "done";

  return {
    id: "secrets",
    title: "Store the credentials as access Secrets",
    summary:
      "The Integration only refers to Secrets by name, so credentials never appear in its configuration.",
    status,
    statusLabel:
      status === "done"
        ? `${referenced.length} Secret${referenced.length === 1 ? "" : "s"} found`
        : status === "attention"
          ? `${missing.length} missing`
          : undefined,
    body: (
      <>
        <P>{opts.intro}</P>
        <div className="grid max-w-3xl grid-cols-1 gap-2 md:grid-cols-2">
          {opts.secrets.map((secret) => (
            <div
              key={secret.name}
              className="flex flex-col gap-1 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5"
            >
              <span className="inline-flex items-center gap-1.5 font-mono text-xs font-semibold text-slate-800">
                <KeyRound size={12} strokeWidth={2.3} className="text-slate-500" />
                {secret.name}
              </span>
              <span className="text-xs leading-5 text-slate-600">
                {secret.description}
              </span>
              <span className="text-micro text-slate-500">
                Value looks like <Code>{secret.example}</Code>
              </span>
            </div>
          ))}
        </div>
        <Bullets>
          <Bullet>
            Create each one in{" "}
            <ConsoleLink to="/access/secrets/create">
              Access → Secrets → Create
            </ConsoleLink>{" "}
            and paste the raw value: no quotes, no base64, no{" "}
            <Code>Bearer</Code> prefix.
          </Bullet>
          <Bullet>
            The value is write-only. Editing a Secret always replaces its value.
          </Bullet>
          <Bullet>
            Only <Strong>access</Strong> Secrets can be used here, not core or
            enterprise Secrets.
          </Bullet>
        </Bullets>
        {missing.length > 0 && (
          <Aside icon={ShieldAlert} title="Missing Secrets" tone="warning">
            This Integration refers to{" "}
            {missing.map((name, index) => (
              <span key={name}>
                {index > 0 && ", "}
                <Code>{name}</Code>
              </span>
            ))}{" "}
            which {missing.length === 1 ? "does" : "do"} not exist yet.
          </Aside>
        )}
      </>
    ),
  };
};

export const connectionStep = (
  ctx: GuideContext,
  opts: { summary: string; checks: React.ReactNode },
): GuideStep => {
  const p = ctx.presentation;
  const state = ctx.item.status?.state;
  const status: GuideStepStatus = ctx.item.spec?.isDisabled
    ? "attention"
    : state === AccessP.Integration_Status_State.READY
      ? "done"
      : state === AccessP.Integration_Status_State.ERROR ||
          state === AccessP.Integration_Status_State.DEGRADED
        ? "attention"
        : "todo";

  return {
    id: "connect",
    title: "Connect and verify",
    summary: opts.summary,
    status,
    statusLabel: ctx.item.spec?.isDisabled ? "Disabled" : p.stateMeta.label,
    body: (
      <>
        <P>
          Select <Strong>Synchronize now</Strong> in the header of this page.
          The Cluster checks the credentials against the provider and records
          the result below.
        </P>
        <div className="grid max-w-3xl grid-cols-1 gap-2 sm:grid-cols-3">
          <div className="flex flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2">
            <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
              Health
            </span>
            <span className="text-sm font-semibold text-slate-800">
              {p.stateMeta.label}
            </span>
          </div>
          <div className="flex flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2">
            <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
              Last synchronization
            </span>
            <span className="text-sm font-semibold text-slate-800">
              {p.currentSync ? getSyncStateMeta(p.currentSync.state).label : "Never"}
            </span>
            {p.currentSync?.createdAt && (
              <span className="text-micro text-slate-500">
                <TimeAgo rfc3339={p.currentSync.createdAt} />
              </span>
            )}
          </div>
          <div className="flex flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2">
            <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
              Provider tenant
            </span>
            <span className="truncate text-sm font-semibold text-slate-800">
              {ctx.item.status?.externalTenantName ||
                ctx.item.status?.externalTenantID ||
                "Not discovered yet"}
            </span>
          </div>
        </div>
        {opts.checks}
        {ctx.item.status?.lastError && (
          <Aside icon={ShieldAlert} title="Last error reported by the provider" tone="warning">
            <Code>{ctx.item.status.lastError}</Code>
          </Aside>
        )}
        {ctx.item.spec?.isDisabled && (
          <Aside icon={ShieldAlert} title="This Integration is disabled" tone="warning">
            A disabled Integration delivers nothing and rejects every inbound
            request. Enable it from the Configuration tab.
          </Aside>
        )}
      </>
    ),
  };
};

export const policyYAML = (opts: {
  integration: string;
  reviewAudience: "SHARED" | "REVIEWERS";
  notificationAudience?: "SHARED" | "REQUESTER";
}) => {
  const notifications = opts.notificationAudience
    ? `
      notifications:
        - destination:
            integrationRef:
              apiVersion: access/v1
              kind: Integration
              name: ${opts.integration}
            audience: ${opts.notificationAudience}`
    : "";

  return `apiVersion: access/v1
kind: Policy
metadata:
  name: production-access
spec:
  rules:
    - name: production-review
      condition:
        resource:
          catalogRef:
            apiVersion: access/v1
            kind: Catalog
            name: production-support
      effect: REVIEW
      action:
        review:
          steps:
            - name: team-lead
              reviewers:
                - group:
                    groupRef:
                      apiVersion: core/v1
                      kind: Group
                      name: sre-oncall
              approvalRequirement: ANY
              timeout:
                minutes: 30
              onTimeout: ON_TIMEOUT_REJECT
              surfaces:
                - destination:
                    integrationRef:
                      apiVersion: access/v1
                      kind: Integration
                      name: ${opts.integration}
                    audience: ${opts.reviewAudience}
                  interactionMode: INTERACTIVE
      authorization:
        maxAccessDuration:
          hours: 4${notifications}
`;
};

const CreatePolicyButton = (props: { yaml: string }) => {
  const navigate = useNavigate();
  return (
    <Button
      size="compact-sm"
      variant="default"
      leftSection={<FilePlus2 size={13} strokeWidth={2.3} />}
      onClick={() =>
        navigate("/access/policies/create", {
          state: { seedYAML: props.yaml, returnTo: "/access/policies" },
        })
      }
    >
      Open as a new Policy
    </Button>
  );
};

export const policyStep = (
  ctx: GuideContext,
  opts: {
    intro: React.ReactNode;
    yaml: string;
    notes: React.ReactNode;
  },
): GuideStep => {
  const used = ctx.usages.items;
  const status: GuideStepStatus = !ctx.usages.isLoaded
    ? "todo"
    : used.length > 0
      ? "done"
      : "todo";

  return {
    id: "policy",
    title: "Send requests to it from a Policy",
    summary:
      "An Integration does nothing until an access Policy rule presents its requests or outcomes on it.",
    status,
    statusLabel:
      used.length > 0
        ? `Used by ${used.length} surface${used.length === 1 ? "" : "s"}`
        : undefined,
    body: (
      <>
        <P>{opts.intro}</P>
        {used.length > 0 && (
          <div className="flex max-w-3xl flex-wrap gap-1.5">
            {used.map((usage, index) => (
              <RefChip
                key={`${usage.policy.metadata?.uid}-${usage.rule.name}-${index}`}
                itemRef={{
                  apiVersion: "access/v1",
                  kind: "Policy",
                  name: usage.policy.metadata?.name ?? "",
                  uid: usage.policy.metadata?.uid ?? "",
                  resourceVersion: "",
                }}
                icon={ShieldCheck}
                label={`${usage.policy.metadata?.name} · ${usage.rule.name}${
                  usage.stepName ? ` · ${usage.stepName}` : " · outcome"
                }`}
              />
            ))}
          </div>
        )}
        {opts.notes}
        <CodeBlock code={opts.yaml} language="yaml" title="Example Policy" />
        <div className="flex flex-wrap items-center gap-2">
          <CreatePolicyButton yaml={opts.yaml} />
          <span className="text-micro text-slate-500">
            Opens the Policy form prefilled with this example. Replace the
            Catalog and Group with your own before creating it.
          </span>
        </div>
      </>
    ),
  };
};

export const identitiesStatus = (ctx: GuideContext): GuideStepStatus =>
  !ctx.identities.isLoaded ? "todo" : ctx.identities.count > 0 ? "done" : "todo";

export const testStep = (
  ctx: GuideContext,
  opts: { body: React.ReactNode },
): GuideStep => ({
  id: "test",
  title: "Try it end to end",
  summary:
    "Create a request that matches your Policy and review it from the provider.",
  status: !ctx.deliveries.isLoaded
    ? "todo"
    : ctx.deliveries.failing > 0
      ? "attention"
      : ctx.deliveries.total > 0
        ? "done"
        : "todo",
  statusLabel:
    ctx.deliveries.failing > 0
      ? `${ctx.deliveries.failing} failing`
      : ctx.deliveries.total > 0
        ? `${ctx.deliveries.total} deliveries`
        : undefined,
  body: opts.body,
});
