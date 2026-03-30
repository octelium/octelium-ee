import { FileInput, Textarea } from "@mantine/core";

const TextAreaCustom = (props: {
  value?: string;
  onChange: (value?: string) => void;
  required?: boolean;
  rows?: number;
  label?: string;
  placeholder?: string;
}) => {
  return (
    <div className="w-full">
      <div>
        <Textarea
          rows={props.rows ? props.rows : 5}
          label={props.label}
          placeholder={props.placeholder}
          required={props.required}
          value={props.value}
          onChange={(v) => {
            props.onChange(v.target.value);
          }}
        />
        <div className="flex items-end justify-end my-4">
          <FileInput
            label="Set from a file"
            placeholder="Click to select a file"
            accept="text/*"
            onChange={(file) => {
              if (!file) {
                return;
              }

              const reader = new FileReader();

              reader.onload = (e: ProgressEvent<FileReader>) => {
                const result = e.target?.result;

                if (!(result instanceof ArrayBuffer)) {
                  return;
                }

                const decoder = new TextDecoder("utf-8", { fatal: true });
                try {
                  const value = decoder.decode(result);
                  props.onChange(value);
                } catch {}
              };

              reader.onerror = () => {};

              reader.readAsArrayBuffer(file);
            }}
          />
        </div>
      </div>
    </div>
  );
};

export default TextAreaCustom;
