import * as AccessP from "@/apis/accessv1/accessv1";
import { twMerge } from "@/lib/utils";
import { getDomain } from "@/utils";
import {
  AlertTriangle,
  BookOpen,
  Check,
  CircleDashed,
  Clock3,
  Hand,
  LifeBuoy,
  MinusCircle,
} from "lucide-react";
import * as React from "react";
import { ProviderTile, getProviderKind, ProviderKind } from "../../common";
import {
  getIntegrationUsages,
  IntegrationUsage,
  useAccessPolicies,
  useAccessSecretNames,
  useIntegrationDeliverySummary,
  useIntegrationIdentities,
} from "../queries";
import {
  getInboundEndpoints,
  getIntegrationPresentation,
  InboundEndpoint,
} from "../utils";
import { Faq, FlowDiagram, FlowNode } from "./blocks";
import { getJiraGuide } from "./jira";
import { getSlackGuide } from "./slack";
import { getWebhookGuide } from "./webhook";

export type GuideStepStatus = "done" | "attention" | "todo" | "manual";

export type GuideStep = {
  id: string;
  title: string;
  summary: string;
  status: GuideStepStatus;
  statusLabel?: string;
  optional?: boolean;
  body: React.ReactNode;
};

export type GuideDefinition = {
  title: string;
  intro: React.ReactNode;
  duration: string;
  requirements: string[];
  flow: FlowNode[];
  steps: GuideStep[];
  faq: { question: string; answer: React.ReactNode }[];
};

export type GuideContext = {
  item: AccessP.Integration;
  provider: ProviderKind;
  domain: string;
  name: string;
  endpoints: InboundEndpoint[];
  presentation: ReturnType<typeof getIntegrationPresentation>;
  secrets: {
    names: string[];
    missing: string[];
    isLoaded: boolean;
  };
  usages: {
    items: IntegrationUsage[];
    interactive: number;
    isLoaded: boolean;
  };
  identities: { count: number; isLoaded: boolean };
  deliveries: { total: number; failing: number; isLoaded: boolean };
  portalURL: string;
};

const STATUS_META: Record<
  GuideStepStatus,
  { label: string; className: string; icon: React.ReactNode; dot: string }
> = {
  done: {
    label: "Done",
    className: "border-emerald-200 bg-emerald-50 text-emerald-700",
    icon: <Check size={13} strokeWidth={3} />,
    dot: "border-emerald-300 bg-emerald-500 text-white",
  },
  attention: {
    label: "Needs attention",
    className: "border-amber-200 bg-amber-50 text-amber-700",
    icon: <AlertTriangle size={12} strokeWidth={2.5} />,
    dot: "border-amber-300 bg-amber-50 text-amber-700",
  },
  todo: {
    label: "To do",
    className: "border-slate-200 bg-white text-slate-600",
    icon: <CircleDashed size={12} strokeWidth={2.5} />,
    dot: "border-slate-300 bg-white text-slate-500",
  },
  manual: {
    label: "Outside Octelium",
    className: "border-blue-200 bg-blue-50 text-blue-700",
    icon: <Hand size={12} strokeWidth={2.4} />,
    dot: "border-blue-200 bg-blue-50 text-blue-700",
  },
};

const useGuideContext = (item: AccessP.Integration): GuideContext | undefined => {
  const provider = getProviderKind(item);
  const presentation = getIntegrationPresentation(item);
  const domain = getDomain();
  const secretNames = useAccessSecretNames();
  const policies = useAccessPolicies();
  const identities = useIntegrationIdentities(item);
  const deliveries = useIntegrationDeliverySummary(item);

  if (!provider) return undefined;

  const usages = getIntegrationUsages(policies.data?.items ?? [], item);

  return {
    item,
    provider,
    domain,
    name: item.metadata?.name ?? "my-integration",
    endpoints: getInboundEndpoints(item, domain),
    presentation,
    secrets: {
      names: presentation.secretNames,
      missing: secretNames.data
        ? presentation.secretNames.filter((name) => !secretNames.data.has(name))
        : [],
      isLoaded: !!secretNames.data,
    },
    usages: {
      items: usages,
      interactive: usages.filter(
        (usage) =>
          !usage.isNotification &&
          usage.surface.interactionMode ===
            AccessP.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE,
      ).length,
      isLoaded: !!policies.data,
    },
    identities: {
      count: identities.data?.listResponseMeta?.totalCount ?? 0,
      isLoaded: !!identities.data,
    },
    deliveries: {
      total: Number(deliveries.data?.totalNumber ?? 0),
      failing: Number(deliveries.data?.totalFailing ?? 0),
      isLoaded: !!deliveries.data,
    },
    portalURL: `https://access.octelium.${domain}`,
  };
};

const StepCard = (props: {
  step: GuideStep;
  index: number;
  isLast: boolean;
}) => {
  const { step } = props;
  const meta = STATUS_META[step.status];

  return (
    <li id={`guide-${step.id}`} className="relative flex scroll-mt-4 gap-3 pb-6 last:pb-0">
      {!props.isLast && (
        <span
          aria-hidden="true"
          className={twMerge(
            "absolute bottom-0 left-[15px] top-9 w-px",
            step.status === "done" ? "bg-emerald-200" : "bg-slate-200",
          )}
        />
      )}
      <span
        aria-hidden="true"
        className={twMerge(
          "relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border text-xs font-bold",
          meta.dot,
        )}
      >
        {step.status === "done" ? meta.icon : props.index + 1}
      </span>

      <section className="flex min-w-0 flex-1 flex-col gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3.5 shadow-card">
        <header className="flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0">
            <h3 className="text-sm font-semibold text-slate-900">
              <span className="sr-only">Step {props.index + 1}: </span>
              {step.title}
              {step.optional && (
                <span className="ml-2 align-middle text-micro font-normal text-slate-400">
                  Optional
                </span>
              )}
            </h3>
            <p className="mt-0.5 text-xs font-normal leading-5 text-slate-500">
              {step.summary}
            </p>
          </div>
          <span
            className={twMerge(
              "inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-micro font-semibold",
              meta.className,
            )}
          >
            {meta.icon}
            {step.statusLabel ?? meta.label}
          </span>
        </header>
        <div className="flex min-w-0 flex-col gap-3">{step.body}</div>
      </section>
    </li>
  );
};

const Progress = (props: { steps: GuideStep[] }) => {
  const tracked = props.steps.filter((step) => step.status !== "manual");
  const done = tracked.filter((step) => step.status === "done").length;
  const attention = tracked.filter((step) => step.status === "attention").length;
  const share = tracked.length === 0 ? 0 : (done / tracked.length) * 100;

  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
          Verified by the Cluster
        </span>
        <span className="text-xs font-semibold tabular-nums text-slate-700">
          {done} of {tracked.length}
        </span>
      </div>
      <div
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={tracked.length}
        aria-valuenow={done}
        aria-label="Setup progress"
        className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100"
      >
        <span
          className="block h-full rounded-full bg-emerald-500 transition-[width] duration-500"
          style={{ width: `${share}%` }}
        />
      </div>
      {attention > 0 && (
        <span className="text-micro font-semibold text-amber-700">
          {attention} step{attention === 1 ? " needs" : "s need"} attention
        </span>
      )}
    </div>
  );
};

const Toc = (props: { steps: GuideStep[] }) => (
  <nav aria-label="Guide steps" className="flex flex-col gap-0.5">
    <span className="mb-1 px-2 text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
      Steps
    </span>
    {props.steps.map((step, index) => {
      const meta = STATUS_META[step.status];
      return (
        <a
          key={step.id}
          href={`#guide-${step.id}`}
          onClick={(event) => {
            event.preventDefault();
            document
              .getElementById(`guide-${step.id}`)
              ?.scrollIntoView({ behavior: "smooth", block: "start" });
          }}
          className="group flex items-center gap-2 rounded-md px-2 py-1.5 text-xs font-normal text-slate-600 outline-none transition-colors duration-150 hover:bg-slate-50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <span
            aria-hidden="true"
            className={twMerge(
              "flex h-4 w-4 shrink-0 items-center justify-center rounded-full border text-[9px] font-bold",
              meta.dot,
            )}
          >
            {step.status === "done" ? (
              <Check size={9} strokeWidth={3.5} />
            ) : (
              index + 1
            )}
          </span>
          <span className="min-w-0 truncate">{step.title}</span>
        </a>
      );
    })}
  </nav>
);

const IntegrationGuide = (props: { item: AccessP.Integration }) => {
  const ctx = useGuideContext(props.item);

  if (!ctx) {
    return (
      <div className="px-4 py-10 text-center text-sm text-slate-500">
        Choose a provider for this Integration to see its setup guide.
      </div>
    );
  }

  const guide =
    ctx.provider === "slack"
      ? getSlackGuide(ctx)
      : ctx.provider === "jira"
        ? getJiraGuide(ctx)
        : getWebhookGuide(ctx);

  return (
    <div className="@container flex flex-col gap-5 bg-slate-50/40 px-3 py-4 sm:px-4">
      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
        <div className="flex flex-col gap-4 px-4 py-4 @3xl:flex-row @3xl:items-start @3xl:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <ProviderTile provider={ctx.provider} size="lg" />
            <div className="min-w-0">
              <h2 className="text-base font-bold text-slate-900">
                {guide.title}
              </h2>
              <div className="mt-1 max-w-2xl text-body leading-6 text-slate-600">
                {guide.intro}
              </div>
              <div className="mt-2.5 flex flex-wrap gap-1.5">
                <span className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-micro font-semibold text-slate-600">
                  <Clock3 size={11} strokeWidth={2.4} />
                  {guide.duration}
                </span>
                {guide.requirements.map((requirement) => (
                  <span
                    key={requirement}
                    className="inline-flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-micro font-semibold text-slate-600"
                  >
                    <BookOpen size={11} strokeWidth={2.4} />
                    {requirement}
                  </span>
                ))}
              </div>
            </div>
          </div>
          <div className="w-full shrink-0 @3xl:w-64">
            <Progress steps={guide.steps} />
          </div>
        </div>
        <div className="border-t border-slate-100 bg-slate-50/60 px-4 py-4">
          <span className="mb-2.5 block text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
            How it works
          </span>
          <FlowDiagram nodes={guide.flow} />
        </div>
      </section>

      <div className="grid grid-cols-1 gap-5 @5xl:grid-cols-[12rem_minmax(0,1fr)]">
        <aside className="hidden @5xl:block">
          <div className="sticky top-2 rounded-xl border border-slate-200 bg-white p-2 shadow-card">
            <Toc steps={guide.steps} />
          </div>
        </aside>
        <ol className="flex min-w-0 list-none flex-col">
          {guide.steps.map((step, index) => (
            <StepCard
              key={step.id}
              step={step}
              index={index}
              isLast={index === guide.steps.length - 1}
            />
          ))}
        </ol>
      </div>

      {guide.faq.length > 0 && (
        <section className="flex flex-col gap-2.5">
          <div className="flex items-center gap-2">
            <LifeBuoy size={15} strokeWidth={2.2} className="text-slate-500" />
            <h3 className="text-sm font-semibold text-slate-900">
              Troubleshooting
            </h3>
          </div>
          <Faq items={guide.faq} />
        </section>
      )}

      <p className="flex items-center gap-1.5 text-micro font-normal text-slate-500">
        <MinusCircle size={11} strokeWidth={2.4} />
        Steps marked “Outside Octelium” happen in the provider and cannot be
        verified by the Cluster.
      </p>
    </div>
  );
};

export default IntegrationGuide;
