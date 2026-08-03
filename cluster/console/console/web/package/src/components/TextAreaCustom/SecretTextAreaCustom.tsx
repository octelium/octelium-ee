import {
  ActionIcon,
  Button,
  CopyButton,
  Input,
  Textarea,
  Tooltip,
} from "@mantine/core";
import {
  Check,
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  Paperclip,
} from "lucide-react";
import { useRef, useState } from "react";

const SecretTextAreaCustom = (props: {
  value?: string;
  onChange: (value?: string) => void;
  required?: boolean;
  rows?: number;
  label?: string;
  placeholder?: string;
  description?: string;
}) => {
  const [revealed, setRevealed] = useState(false);
  const [fileError, setFileError] = useState<string>();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const value = props.value ?? "";

  const handleFile = (file?: File) => {
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (event: ProgressEvent<FileReader>) => {
      const result = event.target?.result;
      if (!(result instanceof ArrayBuffer)) return;

      try {
        props.onChange(
          new TextDecoder("utf-8", { fatal: true }).decode(result),
        );
        setFileError(undefined);
      } catch {
        setFileError("The selected file is not valid UTF-8 text.");
      }
    };
    reader.onerror = () => setFileError("The selected file could not be read.");
    reader.readAsArrayBuffer(file);
  };

  return (
    <Input.Wrapper
      label={props.label}
      description={props.description}
      required={props.required}
      error={fileError}
      styles={{
        label: {
          color: "#475569",
          fontSize: "0.72rem",
          fontWeight: 700,
          letterSpacing: "0.02em",
          marginBottom: "5px",
        },
        description: {
          color: "#94a3b8",
          fontSize: "0.68rem",
          fontWeight: 600,
          marginBottom: "6px",
        },
        error: { fontSize: "0.68rem", fontWeight: 600, marginTop: "6px" },
      }}
    >
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_3px_rgba(15,23,42,0.05)] transition-[border-color,box-shadow] duration-500 focus-within:border-slate-400 focus-within:shadow-[0_0_0_3px_rgba(148,163,184,0.15)]">
        <div className="flex min-h-10 flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-slate-50/70 px-2.5 py-1.5">
          <div className="flex min-w-0 items-center gap-2">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-slate-900 text-white">
              <KeyRound size={12} strokeWidth={2.5} />
            </span>
            <span className="truncate text-[0.68rem] font-bold text-slate-600">
              Secret value
            </span>
            {value && (
              <span className="hidden text-[0.6rem] font-semibold text-slate-400 sm:inline">
                {value.length.toLocaleString()} characters
              </span>
            )}
          </div>

          <div className="flex items-center gap-1">
            <input
              ref={fileInputRef}
              type="file"
              accept="text/*"
              className="hidden"
              onChange={(event) => {
                handleFile(event.target.files?.[0]);
                event.currentTarget.value = "";
              }}
            />
            <Button
              type="button"
              size="compact-xs"
              variant="subtle"
              color="gray"
              leftSection={<Paperclip size={11} strokeWidth={2.5} />}
              onClick={() => fileInputRef.current?.click()}
              styles={{ root: { fontSize: "0.65rem", fontWeight: 700 } }}
            >
              Load file
            </Button>

            <CopyButton value={value} timeout={2_000}>
              {({ copied, copy }) => (
                <Tooltip label={copied ? "Copied" : "Copy value"} withArrow>
                  <ActionIcon
                    type="button"
                    size="sm"
                    variant="subtle"
                    color={copied ? "teal" : "gray"}
                    disabled={!value}
                    aria-label={copied ? "Secret copied" : "Copy secret"}
                    onClick={copy}
                  >
                    {copied ? (
                      <Check size={13} strokeWidth={2.5} />
                    ) : (
                      <Copy size={13} strokeWidth={2.25} />
                    )}
                  </ActionIcon>
                </Tooltip>
              )}
            </CopyButton>

            <Tooltip
              label={revealed ? "Hide secret" : "Reveal secret"}
              withArrow
            >
              <ActionIcon
                type="button"
                size="sm"
                variant={revealed ? "light" : "subtle"}
                color={revealed ? "blue" : "gray"}
                aria-label={revealed ? "Hide secret" : "Reveal secret"}
                aria-pressed={revealed}
                onClick={() => setRevealed((current) => !current)}
              >
                {revealed ? (
                  <EyeOff size={13} strokeWidth={2.5} />
                ) : (
                  <Eye size={13} strokeWidth={2.5} />
                )}
              </ActionIcon>
            </Tooltip>
          </div>
        </div>

        <Textarea
          aria-label={props.label ?? "Secret value"}
          placeholder={props.placeholder ?? "Enter or load a secret value"}
          required={props.required}
          value={value}
          rows={props.rows ?? 4}
          onChange={(event) => {
            setFileError(undefined);
            props.onChange(event.target.value);
          }}
          styles={{
            input: {
              WebkitTextSecurity: revealed ? "none" : "disc",
              backgroundColor: "#ffffff",
              border: 0,
              borderRadius: 0,
              boxShadow: "none",
              color: "#1e293b",
              fontFamily: "Ubuntu, sans-serif",
              fontSize: "0.8rem",
              fontWeight: 600,
              letterSpacing: 0,
              lineHeight: 1.6,
              minHeight: 0,
              padding: "12px 14px",
              resize: "vertical",
              transition: "none",
              "&:focus": { outline: "none" },
            },
          }}
        />
      </div>
    </Input.Wrapper>
  );
};

export default SecretTextAreaCustom;
