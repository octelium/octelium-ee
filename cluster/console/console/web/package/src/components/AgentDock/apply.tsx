import { Tooltip } from "@mantine/core";
import { ArrowDownToLine, Check, X } from "lucide-react";
import * as React from "react";
import { useAgentApplyTarget } from "./applyTarget";

export const ApplyCodeButton = (props: { language?: string; code: string }) => {
  const target = useAgentApplyTarget();
  const [result, setResult] = React.useState<{ id: string; error?: string }>();
  const language = (props.language ?? "").toLowerCase();
  if (!target || !target.languages.includes(language)) return null;
  const done = result?.id === target.id && !result.error;
  const error = result?.id === target.id ? result.error : undefined;

  return (
    <Tooltip
      label={
        error ??
        (done ? "Applied" : `Replace the ${target.label} with this code`)
      }
      withArrow
      color={error ? "red" : undefined}
      opened={error ? true : undefined}
      multiline
      maw={280}
    >
      <button
        type="button"
        onClick={() =>
          setResult({ id: target.id, error: target.apply(props.code) })
        }
        className="inline-flex h-6 max-w-[220px] cursor-pointer items-center gap-1 rounded-md border border-terminal-line bg-terminal-hover px-2 text-micro font-semibold text-terminal-bright outline-none transition-colors duration-150 hover:border-terminal-muted focus-visible:ring-2 focus-visible:ring-terminal-accent"
      >
        {error ? (
          <X size={11} className="shrink-0 text-terminal-error" />
        ) : done ? (
          <Check size={11} className="shrink-0 text-terminal-success" />
        ) : (
          <ArrowDownToLine size={11} className="shrink-0" />
        )}
        <span className="truncate">
          {done ? "Applied" : `Apply to ${target.label}`}
        </span>
      </button>
    </Tooltip>
  );
};
