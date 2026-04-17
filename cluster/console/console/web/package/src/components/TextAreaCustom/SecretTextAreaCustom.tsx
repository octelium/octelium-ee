import { FileInput, Textarea } from "@mantine/core";
import { Eye, EyeOff, Paperclip } from "lucide-react";
import { useState } from "react";

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

  const handleFile = (file: File | null) => {
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (e: ProgressEvent<FileReader>) => {
      const result = e.target?.result;
      if (!(result instanceof ArrayBuffer)) return;
      try {
        props.onChange(
          new TextDecoder("utf-8", { fatal: true }).decode(result),
        );
      } catch {}
    };
    reader.readAsArrayBuffer(file);
  };

  return (
    <div className="w-full flex flex-col gap-2">
      <div className="relative">
        <Textarea
          label={props.label}
          placeholder={props.placeholder}
          description={props.description}
          required={props.required}
          value={props.value}
          onChange={(e) => props.onChange(e.target.value)}
          autosize
          minRows={props.rows ?? 5}
          styles={{
            input: revealed
              ? undefined
              : {
                  WebkitTextSecurity: "disc" as any,
                  fontFamily: "text-security-disc, monospace",
                  letterSpacing: "0.15em",
                },
          }}
        />
        <button
          type="button"
          onClick={() => setRevealed((v) => !v)}
          className="absolute right-2.5 top-[34px] flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer"
          title={revealed ? "Hide value" : "Reveal value"}
        >
          {revealed ? (
            <EyeOff size={13} strokeWidth={2.5} />
          ) : (
            <Eye size={13} strokeWidth={2.5} />
          )}
        </button>
      </div>

      <div className="flex justify-end">
        <FileInput
          accept="text/*"
          placeholder="Load from file"
          leftSection={<Paperclip size={12} strokeWidth={2.5} />}
          onChange={handleFile}
          clearable
          styles={{
            root: { width: "auto" },
            input: {
              fontSize: "0.72rem",
              fontWeight: 600,
              height: "28px",
              minHeight: "28px",
              paddingTop: 0,
              paddingBottom: 0,
              cursor: "pointer",
            },
          }}
        />
      </div>
    </div>
  );
};

export default SecretTextAreaCustom;
