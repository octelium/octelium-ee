import { formatDuration } from "@/pages/Agent/format";
import { CopyButton, Shimmer } from "@/pages/Agent/ui";
import { Loader } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import {
  Brain,
  Check,
  ChevronDown,
  ChevronRight,
  ListChecks,
  Wrench,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";

export type Step =
  | { kind: "thinking"; id: string; text: string; duration?: number }
  | {
      kind: "tool";
      id: string;
      name: string;
      input: string;
      duration?: number;
    };

const ease = [0.22, 1, 0.36, 1] as const;

const Collapse = (props: { opened: boolean; children: React.ReactNode }) => (
  <AnimatePresence initial={false}>
    {props.opened && (
      <motion.div
        initial={{ height: 0, opacity: 0 }}
        animate={{ height: "auto", opacity: 1 }}
        exit={{ height: 0, opacity: 0 }}
        transition={{ duration: 0.25, ease }}
        className="overflow-hidden"
      >
        {props.children}
      </motion.div>
    )}
  </AnimatePresence>
);

const prettyInput = (input: string) => {
  try {
    return JSON.stringify(JSON.parse(input), null, 2);
  } catch {
    return input || "{}";
  }
};

const thinkingLabel = (live: boolean, duration?: number) => {
  if (live) return "Thinking";
  const formatted = duration !== undefined ? formatDuration(duration) : "";
  return formatted && duration! >= 1000
    ? `Thought for ${formatted}`
    : "Thought briefly";
};

const stepsSummary = (steps: Step[]) => {
  const tools = steps.filter((step) => step.kind === "tool").length;
  if (tools === 0) return "Reasoning";
  const thought = steps.some((step) => step.kind === "thinking");
  return `${thought ? "Thought and requested" : "Requested"} ${tools} tool call${tools === 1 ? "" : "s"}`;
};

const StepRow = (props: { step: Step; live: boolean; isLast: boolean }) => {
  const { step, live } = props;
  const [opened, setOpened] = React.useState(false);
  const title =
    step.kind === "thinking" ? thinkingLabel(live, step.duration) : step.name;
  const hasDetails = step.kind === "tool" || step.text.trim() !== "";
  const details = step.kind === "tool" ? prettyInput(step.input) : step.text;

  return (
    <li className="relative">
      {!props.isLast && (
        <span className="absolute bottom-0 left-[9.5px] top-7 w-px bg-slate-200" />
      )}
      <div className="flex items-start gap-2">
        <span className="relative z-[1] mt-[7px] flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-white ring-1 ring-slate-200">
          {live ? (
            <Loader size={10} color="gray" />
          ) : step.kind === "thinking" ? (
            <Brain size={11} strokeWidth={2.4} className="text-violet-600" />
          ) : (
            <Check size={11} strokeWidth={3} className="text-emerald-600" />
          )}
        </span>
        <div className="min-w-0 flex-1 pb-1">
          <button
            type="button"
            disabled={!hasDetails}
            aria-expanded={opened}
            onClick={() => setOpened((value) => !value)}
            className="group/step flex min-h-8 w-full cursor-pointer items-center gap-2 rounded-lg px-2 py-1 text-left outline-none transition-colors duration-150 hover:bg-slate-100 focus-visible:bg-slate-100 disabled:cursor-default disabled:hover:bg-transparent"
          >
            {step.kind === "tool" && (
              <Wrench size={13} className="shrink-0 text-slate-400" />
            )}
            <span
              className={twMerge(
                "min-w-0 flex-1 truncate text-xs font-semibold text-slate-700",
                step.kind === "tool" && "font-mono",
              )}
            >
              {live ? <Shimmer>{title}</Shimmer> : title}
            </span>
            {step.kind === "tool" && step.duration !== undefined && (
              <span className="shrink-0 font-mono text-micro text-slate-400">
                {formatDuration(step.duration)}
              </span>
            )}
            {hasDetails && (
              <ChevronRight
                size={13}
                strokeWidth={2.4}
                className={twMerge(
                  "shrink-0 text-slate-400 transition-transform duration-200",
                  opened && "rotate-90",
                )}
              />
            )}
          </button>
          <Collapse opened={opened}>
            <div className="mb-1.5 ml-2 mt-1 rounded-lg border border-slate-200 bg-white p-3">
              {step.kind === "thinking" ? (
                <p className="max-h-[320px] overflow-y-auto whitespace-pre-wrap text-xs leading-5 text-slate-600">
                  {details}
                </p>
              ) : (
                <>
                  <p className="mb-1 text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
                    Arguments
                  </p>
                  <div className="group/code relative">
                    <pre className="max-h-[320px] overflow-auto rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs leading-5">
                      <code className="whitespace-pre font-mono text-slate-700">
                        {details}
                      </code>
                    </pre>
                    <span className="absolute right-1.5 top-1.5 rounded-md bg-white opacity-0 shadow-sm transition-opacity focus-within:opacity-100 group-hover/code:opacity-100 [@media(hover:none)]:opacity-100">
                      <CopyButton value={() => details} />
                    </span>
                  </div>
                </>
              )}
            </div>
          </Collapse>
        </div>
      </div>
    </li>
  );
};

const Steps = (props: { steps: Step[]; live: boolean }) => {
  const { steps, live } = props;
  const [manual, setManual] = React.useState<boolean>();
  const opened = manual ?? live;
  const last = steps.at(-1)!;

  if (steps.length === 1) {
    return (
      <ol className="my-1.5">
        <StepRow step={last} live={live} isLast />
      </ol>
    );
  }

  return (
    <div className="my-1.5">
      <button
        type="button"
        aria-expanded={opened}
        onClick={() => setManual(!opened)}
        className="flex min-h-8 w-full cursor-pointer items-center gap-2 rounded-lg px-1 py-1 text-left outline-none transition-colors duration-150 hover:bg-slate-100 focus-visible:bg-slate-100"
      >
        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-white ring-1 ring-slate-200">
          {live ? (
            <Loader size={10} color="gray" />
          ) : (
            <ListChecks
              size={11}
              strokeWidth={2.4}
              className="text-slate-500"
            />
          )}
        </span>
        <span className="min-w-0 flex-1 truncate pl-1 text-xs font-semibold text-slate-600">
          {live ? (
            <Shimmer>
              {last.kind === "thinking" ? "Thinking" : last.name}
            </Shimmer>
          ) : (
            stepsSummary(steps)
          )}
        </span>
        <ChevronDown
          size={13}
          strokeWidth={2.4}
          className={twMerge(
            "mr-1 shrink-0 text-slate-400 transition-transform duration-200",
            opened && "rotate-180",
          )}
        />
      </button>
      <Collapse opened={opened}>
        <ol className="pb-1 pt-1">
          {steps.map((step, idx) => (
            <StepRow
              key={step.id}
              step={step}
              live={live && idx === steps.length - 1}
              isLast={idx === steps.length - 1}
            />
          ))}
        </ol>
      </Collapse>
    </div>
  );
};

export default Steps;
