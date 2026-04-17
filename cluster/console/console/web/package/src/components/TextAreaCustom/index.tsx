import { FileInput, Textarea } from "@mantine/core";
import { Paperclip } from "lucide-react";

const TextAreaCustom = (props: {
  value?: string;
  onChange: (value?: string) => void;
  required?: boolean;
  rows?: number;
  label?: string;
  placeholder?: string;
  description?: string;
}) => {
  const handleFile = (file: File | null) => {
    if (!file) return;

    const reader = new FileReader();

    reader.onload = (e: ProgressEvent<FileReader>) => {
      const result = e.target?.result;
      if (!(result instanceof ArrayBuffer)) return;

      try {
        const value = new TextDecoder("utf-8", { fatal: true }).decode(result);
        props.onChange(value);
      } catch {
        // non-UTF-8 file — silently ignore
      }
    };

    reader.readAsArrayBuffer(file);
  };

  return (
    <div className="w-full flex flex-col gap-2">
      <Textarea
        rows={props.rows ?? 5}
        label={props.label}
        placeholder={props.placeholder}
        description={props.description}
        required={props.required}
        value={props.value}
        onChange={(e) => props.onChange(e.target.value)}
        autosize
        minRows={props.rows ?? 5}
      />

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

export default TextAreaCustom;
