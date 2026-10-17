import { twMerge } from "@/lib/utils";
import { motion } from "framer-motion";
import { Check, ChevronDown, Hash, Lock, Search } from "lucide-react";
import { ProviderMark } from "../../common";

const Field = (props: { label: string; value: string }) => (
  <div className="flex min-w-0 flex-col">
    <span className="text-micro font-bold text-slate-800">{props.label}</span>
    <span className="truncate text-xs text-slate-600">{props.value}</span>
  </div>
);

export const SlackMessageMock = (props: {
  channel?: string;
  interactive?: boolean;
}) => (
  <div className="mx-auto w-full max-w-xl overflow-hidden rounded-lg border border-slate-200 bg-white shadow-card">
    <div className="flex items-center gap-2 border-b border-slate-100 px-3 py-2">
      <Hash size={13} strokeWidth={2.5} className="text-slate-500" />
      <span className="text-xs font-bold text-slate-800">
        {props.channel || "access-reviews"}
      </span>
    </div>
    <div className="flex gap-2.5 px-3 py-3">
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-slate-900">
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
          <circle cx="12" cy="12" r="7" fill="none" stroke="white" strokeWidth="2.5" />
          <circle cx="12" cy="12" r="2.5" fill="white" />
        </svg>
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex items-center gap-1.5">
          <span className="text-xs font-bold text-slate-900">
            Octelium Access
          </span>
          <span className="rounded-sm bg-slate-100 px-1 text-[9px] font-bold uppercase text-slate-500">
            App
          </span>
          <span className="text-micro text-slate-400">10:42 AM</span>
        </div>
        <span className="text-xs text-slate-800">
          <span className="font-semibold text-blue-700">@sre-oncall</span>{" "}
          Erin Lopez requests access to{" "}
          <span className="font-semibold">production-support</span>
        </span>
        <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 border-l-4 border-slate-300 pl-2.5">
          <Field label="Duration" value="2 hours" />
          <Field label="Urgency" value="Highest" />
          <Field label="Step" value="team-lead (1 of 2)" />
          <Field label="Quorum" value="1 of SRE on-call" />
          <div className="col-span-2">
            <Field
              label="Justification"
              value="Investigating elevated 5xx rates on the billing API (INC-4821)."
            />
          </div>
        </div>
        {props.interactive !== false ? (
          <div className="flex flex-wrap items-center gap-1.5">
            <motion.span
              className="relative inline-flex items-center rounded border border-[#007a5a] bg-[#007a5a] px-3 py-1 text-xs font-bold text-white"
              animate={{ scale: [1, 1, 0.95, 1] }}
              transition={{
                duration: 2.4,
                times: [0, 0.7, 0.8, 1],
                repeat: Infinity,
                repeatDelay: 1.2,
              }}
            >
              Approve
            </motion.span>
            <span className="inline-flex items-center rounded border border-slate-300 bg-white px-3 py-1 text-xs font-bold text-slate-700">
              Reject
            </span>
            <span className="inline-flex items-center rounded border border-slate-300 bg-white px-3 py-1 text-xs font-bold text-slate-700">
              Open in portal
            </span>
          </div>
        ) : (
          <span className="inline-flex w-fit items-center rounded border border-slate-300 bg-white px-3 py-1 text-xs font-bold text-slate-700">
            Open in portal
          </span>
        )}
        <motion.span
          className="inline-flex w-fit items-center gap-1 text-micro font-semibold text-[#007a5a]"
          animate={{ opacity: [0, 0, 1, 1] }}
          transition={{
            duration: 2.4,
            times: [0, 0.8, 0.85, 1],
            repeat: Infinity,
            repeatDelay: 1.2,
          }}
        >
          <Check size={11} strokeWidth={3} />
          Approved by Carol Diaz · the request moves to the security step
        </motion.span>
      </div>
    </div>
  </div>
);

const SettingsShell = (props: {
  title: string;
  provider: "slack" | "jira";
  nav: string[];
  active: string;
  children: React.ReactNode;
}) => (
  <div className="mx-auto flex w-full max-w-2xl overflow-hidden rounded-lg border border-slate-200 bg-white shadow-card">
    <div className="hidden w-40 shrink-0 flex-col gap-0.5 border-r border-slate-100 bg-slate-50 px-2 py-2.5 sm:flex">
      <div className="mb-1.5 flex items-center gap-1.5 px-1.5">
        <ProviderMark provider={props.provider} size={13} />
        <span className="truncate text-micro font-bold text-slate-700">
          {props.title}
        </span>
      </div>
      {props.nav.map((item) => (
        <span
          key={item}
          className={twMerge(
            "truncate rounded px-1.5 py-1 text-micro",
            item === props.active
              ? "bg-white font-bold text-slate-900 shadow-card"
              : "text-slate-500",
          )}
        >
          {item}
        </span>
      ))}
    </div>
    <div className="flex min-w-0 flex-1 flex-col gap-3 px-4 py-3.5">
      {props.children}
    </div>
  </div>
);

const MockInput = (props: {
  label: string;
  value: string;
  mono?: boolean;
  highlight?: boolean;
  secret?: boolean;
}) => (
  <div className="flex min-w-0 flex-col gap-1">
    <span className="text-micro font-bold text-slate-700">{props.label}</span>
    <motion.span
      className={twMerge(
        "flex min-h-7 min-w-0 items-center gap-1.5 rounded border bg-white px-2 py-1 text-xs text-slate-800",
        props.mono && "font-mono text-[0.7rem]",
        props.highlight ? "border-blue-400" : "border-slate-300",
      )}
      animate={
        props.highlight
          ? {
              boxShadow: [
                "0 0 0 0 rgba(59,130,246,0)",
                "0 0 0 3px rgba(59,130,246,0.25)",
                "0 0 0 0 rgba(59,130,246,0)",
              ],
            }
          : undefined
      }
      transition={{ duration: 2.2, repeat: Infinity }}
    >
      {props.secret && (
        <Lock size={11} strokeWidth={2.4} className="shrink-0 text-slate-400" />
      )}
      <span className="min-w-0 truncate">
        {props.secret ? "••••••••••••••••••••" : props.value}
      </span>
    </motion.span>
  </div>
);

const MockToggle = (props: { label: string; on: boolean }) => (
  <div className="flex items-center justify-between gap-3">
    <span className="text-xs font-bold text-slate-800">{props.label}</span>
    <span
      className={twMerge(
        "relative inline-flex h-4 w-7 items-center rounded-full",
        props.on ? "bg-[#007a5a]" : "bg-slate-300",
      )}
    >
      <span
        className={twMerge(
          "absolute h-3 w-3 rounded-full bg-white shadow",
          props.on ? "right-0.5" : "left-0.5",
        )}
      />
    </span>
  </div>
);

export const SlackInteractivityMock = (props: { url?: string }) => (
  <SettingsShell
    title="Octelium Access"
    provider="slack"
    nav={[
      "Basic Information",
      "Socket Mode",
      "Interactivity & Shortcuts",
      "Slash Commands",
      "OAuth & Permissions",
      "Event Subscriptions",
    ]}
    active="Interactivity & Shortcuts"
  >
    <span className="text-sm font-bold text-slate-900">
      Interactivity &amp; Shortcuts
    </span>
    <MockToggle label="Interactivity" on />
    <MockInput
      label="Request URL"
      value={
        props.url ??
        "https://public.octelium.example.com/integration/v1/callback/<id>/slack/interactions"
      }
      mono
      highlight
    />
    <span className="text-micro text-slate-500">
      Slack sends a signed POST to this URL whenever a reviewer clicks a
      button in an Octelium message.
    </span>
    <div className="flex justify-end">
      <span className="rounded bg-[#007a5a] px-2.5 py-1 text-micro font-bold text-white">
        Save Changes
      </span>
    </div>
  </SettingsShell>
);

export const SlackScopesMock = () => (
  <SettingsShell
    title="Octelium Access"
    provider="slack"
    nav={[
      "Basic Information",
      "Socket Mode",
      "Interactivity & Shortcuts",
      "Slash Commands",
      "OAuth & Permissions",
      "Event Subscriptions",
    ]}
    active="OAuth & Permissions"
  >
    <span className="text-sm font-bold text-slate-900">Bot Token Scopes</span>
    <div className="flex flex-col divide-y divide-slate-100 rounded border border-slate-200">
      {[
        ["chat:write", "Send messages as Octelium Access"],
        ["users:read", "View people in a workspace"],
        ["users:read.email", "View email addresses of people in a workspace"],
        ["im:write", "Start direct messages with people"],
        ["commands", "Add shortcuts and/or slash commands"],
      ].map(([scope, description]) => (
        <div key={scope} className="flex items-center justify-between gap-2 px-2.5 py-1.5">
          <span className="font-mono text-[0.7rem] font-semibold text-slate-800">
            {scope}
          </span>
          <span className="truncate text-micro text-slate-500">
            {description}
          </span>
        </div>
      ))}
    </div>
    <MockInput
      label="Bot User OAuth Token"
      value="xoxb-…"
      mono
      secret
    />
  </SettingsShell>
);

export const JiraIssueMock = (props: {
  projectKey?: string;
  approveStatus?: string;
  issueType?: string;
}) => {
  const projectKey = props.projectKey || "ACCESS";
  const approveStatus = props.approveStatus || "Approved";

  return (
    <div className="mx-auto w-full max-w-xl overflow-hidden rounded-lg border border-slate-200 bg-white shadow-card">
      <div className="flex items-center gap-2 border-b border-slate-100 px-3 py-2">
        <ProviderMark provider="jira" size={13} />
        <span className="text-micro font-semibold text-slate-500">
          Projects / {projectKey} /
        </span>
        <span className="text-micro font-bold text-blue-700">
          {projectKey}-128
        </span>
      </div>
      <div className="flex flex-col gap-3 px-3.5 py-3">
        <span className="text-sm font-bold text-slate-900">
          Octelium access Request req-incident-4821 by erin for
          production-support
        </span>
        <div className="flex flex-wrap items-center gap-2">
          <span className="relative inline-flex h-6 items-center overflow-hidden rounded border border-slate-300">
            <motion.span
              className="flex h-full items-center gap-1 px-2 text-micro font-bold uppercase"
              animate={{
                backgroundColor: [
                  "rgba(148,163,184,0.18)",
                  "rgba(148,163,184,0.18)",
                  "rgba(16,185,129,0.22)",
                  "rgba(16,185,129,0.22)",
                ],
              }}
              transition={{
                duration: 3,
                times: [0, 0.45, 0.55, 1],
                repeat: Infinity,
                repeatDelay: 1,
              }}
            >
              <motion.span
                animate={{ opacity: [1, 1, 0, 0] }}
                transition={{
                  duration: 3,
                  times: [0, 0.45, 0.5, 1],
                  repeat: Infinity,
                  repeatDelay: 1,
                }}
                className="text-slate-700"
              >
                To Do
              </motion.span>
              <motion.span
                animate={{ opacity: [0, 0, 1, 1] }}
                transition={{
                  duration: 3,
                  times: [0, 0.5, 0.55, 1],
                  repeat: Infinity,
                  repeatDelay: 1,
                }}
                className="absolute left-2 text-emerald-700"
              >
                {approveStatus}
              </motion.span>
              <ChevronDown size={11} strokeWidth={2.5} className="ml-10 text-slate-500" />
            </motion.span>
          </span>
          <span className="text-micro text-slate-500">
            {props.issueType || "Task"} · Reporter: Octelium Bot
          </span>
        </div>
        <div className="flex flex-col gap-1 rounded border border-slate-200 bg-slate-50 px-2.5 py-2 text-xs text-slate-700">
          <span>The Octelium access Request req-incident-4821 is currently PENDING.</span>
          <span>• Requester: erin · Resource: production-support</span>
          <span>• Review Step: security (2 of 2) · Quorum: 1 of security</span>
        </div>
        <span className="text-micro text-slate-500">
          Moving the issue to <strong>{approveStatus}</strong> submits an
          approving decision on behalf of the Jira user who made the change.
        </span>
      </div>
    </div>
  );
};

export const JiraWebhookMock = (props: {
  url?: string;
  projectKey?: string;
}) => (
  <SettingsShell
    title="Jira settings"
    provider="jira"
    nav={["System", "General configuration", "WebHooks", "Mail", "Audit log"]}
    active="WebHooks"
  >
    <span className="text-sm font-bold text-slate-900">Create a WebHook</span>
    <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
      <MockInput label="Name" value="Octelium access reviews" />
      <MockInput label="Status" value="Enabled" />
    </div>
    <MockInput
      label="URL"
      value={
        props.url ??
        "https://public.octelium.example.com/integration/v1/callback/<id>/jira/webhook"
      }
      mono
      highlight
    />
    <MockInput label="Secret" value="secret" secret />
    <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
      <div className="flex flex-col gap-1">
        <span className="text-micro font-bold text-slate-700">Issue related events</span>
        <span className="flex min-h-7 items-center gap-1.5 rounded border border-slate-300 bg-white px-2 font-mono text-[0.7rem] text-slate-800">
          <Search size={11} strokeWidth={2.4} className="text-slate-400" />
          project = {props.projectKey || "ACCESS"}
        </span>
      </div>
      <div className="flex flex-col gap-1">
        <span className="text-micro font-bold text-slate-700">Issue</span>
        <span className="flex flex-wrap gap-1.5 text-xs text-slate-700">
          {["created", "updated", "deleted"].map((event) => (
            <span key={event} className="inline-flex items-center gap-1">
              <span
                className={twMerge(
                  "flex h-3.5 w-3.5 items-center justify-center rounded-sm border",
                  event === "updated"
                    ? "border-blue-600 bg-blue-600 text-white"
                    : "border-slate-300 bg-white",
                )}
              >
                {event === "updated" && <Check size={9} strokeWidth={3.5} />}
              </span>
              {event}
            </span>
          ))}
        </span>
      </div>
    </div>
  </SettingsShell>
);
