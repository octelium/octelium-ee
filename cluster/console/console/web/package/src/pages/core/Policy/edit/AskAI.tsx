import { twMerge } from "@/lib/utils";
import { Button, Modal, Textarea } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { ArrowDownToLine, SendHorizontal, Sparkles } from "lucide-react";
import * as React from "react";

export type AskAIAction = {
  id: string;
  label: string;
  hint: string;
  needsText: boolean;
};

const AskAIDialog = (props: {
  opened: boolean;
  title: string;
  subject: string;
  actions: AskAIAction[];
  placeholder: string;
  examples: string[];
  applyLabel: string;
  onClose: () => void;
  onSend: (action: string, text: string) => void;
}) => {
  const [action, setAction] = React.useState(props.actions[0]?.id ?? "");
  const [text, setText] = React.useState("");
  const mobile = useMediaQuery("(max-width: 40em)");
  const current =
    props.actions.find((item) => item.id === action) ?? props.actions[0];
  const canSend = !current?.needsText || text.trim().length > 0;

  const send = () => {
    if (!canSend || !current) return;
    props.onSend(current.id, text);
  };

  return (
    <Modal
      opened={props.opened}
      onClose={props.onClose}
      fullScreen={!!mobile}
      size="lg"
      radius={mobile ? 0 : "lg"}
      padding={0}
      withCloseButton={false}
      styles={{ body: { padding: 0 } }}
    >
      <div className="flex flex-col">
        <header className="flex items-start gap-3 border-b border-slate-200 px-5 py-4">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white shadow-sm">
            <Sparkles size={16} strokeWidth={2.25} />
          </span>
          <div className="min-w-0">
            <Modal.Title className="text-sm font-bold tracking-tight text-slate-900">
              {props.title}
            </Modal.Title>
            <p className="mt-0.5 text-xs text-slate-500">{props.subject}</p>
          </div>
        </header>

        <div className="flex flex-col gap-4 px-5 py-4">
          <div
            role="radiogroup"
            aria-label="What to ask"
            className="grid grid-cols-2 gap-1.5 sm:grid-cols-4"
          >
            {props.actions.map((item) => {
              const active = item.id === current?.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  role="radio"
                  aria-checked={active}
                  onClick={() => setAction(item.id)}
                  className={twMerge(
                    "flex min-w-0 cursor-pointer flex-col gap-0.5 rounded-lg border px-2.5 py-2 text-left outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
                    active
                      ? "border-slate-800 bg-slate-800 text-white"
                      : "border-slate-200 bg-white text-slate-700 hover:border-slate-300",
                  )}
                >
                  <span className="text-xs font-semibold">{item.label}</span>
                  <span
                    className={twMerge(
                      "text-micro leading-4",
                      active ? "text-slate-300" : "text-slate-500",
                    )}
                  >
                    {item.hint}
                  </span>
                </button>
              );
            })}
          </div>

          <Textarea
            data-autofocus
            label={
              current?.needsText ? "Describe it" : "Anything to add? (optional)"
            }
            placeholder={
              current?.needsText
                ? props.placeholder
                : "e.g. focus on contractors"
            }
            autosize
            minRows={current?.needsText ? 4 : 2}
            maxRows={10}
            value={text}
            onChange={(event) => setText(event.currentTarget.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
                event.preventDefault();
                send();
              }
            }}
          />

          {current?.needsText && props.examples.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Examples
              </span>
              <div className="flex flex-wrap gap-1.5">
                {props.examples.map((example) => (
                  <button
                    key={example}
                    type="button"
                    onClick={() => setText(example)}
                    className="cursor-pointer rounded-full border border-slate-200 bg-white px-2.5 py-1 text-left text-micro text-slate-600 outline-none transition-colors hover:border-slate-300 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
                  >
                    {example}
                  </button>
                ))}
              </div>
            </div>
          )}

          <p className="flex items-start gap-2 rounded-lg bg-slate-50 px-3 py-2 text-micro leading-4 text-slate-600">
            <ArrowDownToLine
              size={12}
              className="mt-0.5 shrink-0 text-slate-500"
            />
            <span>
              The answer appears in the AI panel with this draft as context. Use{" "}
              <b>Apply to {props.applyLabel}</b> on its code block to bring the
              proposal back here, then review it before saving.
            </span>
          </p>
        </div>

        <footer className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/70 px-5 py-3">
          <Button variant="default" onClick={props.onClose}>
            Cancel
          </Button>
          <Button
            color="ink"
            leftSection={<SendHorizontal size={13} />}
            disabled={!canSend}
            onClick={send}
          >
            Ask AI
          </Button>
        </footer>
      </div>
    </Modal>
  );
};

export default AskAIDialog;
