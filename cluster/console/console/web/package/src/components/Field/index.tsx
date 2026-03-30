import { twMerge } from "tailwind-merge";
import { TextInput } from "@mantine/core";

const Field = (props: {
  val: string | number;
  onChange: (val: string | number) => void;
  isRequired?: boolean;
  label: string;
  multiLine?: boolean;
  rows?: number;
  maxRows?: number;
  isNumber?: boolean;
  description?: string | undefined;
  placeholder?: string | undefined;
}) => {
  if (props.rows) {
    return (
      <textarea
        value={props.val}
        rows={props.rows}
        onChange={(e) => {
          if (props.isNumber && e.target.value == "") {
            props.onChange(0);
            return;
          } else if (props.isNumber) {
            props.onChange(parseInt(e.target.value));
            return;
          }

          props.onChange(e.target.value);
        }}
        placeholder={props.label}
        className={twMerge(
          "inline-flex w-full outline-none p-2 border-[3px]",
          "rounded-lg border-gray-400 focus:border-gray-900",
          "transition-all duration-300 focus:shadow-md",
          "resize-none",
          "font-semibold"
        )}
      />
    );
  }

  return (
    <TextInput
      label={props.label}
      required={props.isRequired}
      description={props.description}
      placeholder={props.placeholder}
    />
  );
  return (
    <input
      value={props.val}
      required={props.isRequired}
      type={props.isNumber ? "number" : undefined}
      onChange={(e) => {
        if (props.isNumber && e.target.value == "") {
          props.onChange(0);
          return;
        } else if (props.isNumber) {
          props.onChange(parseInt(e.target.value));
          return;
        }

        props.onChange(e.target.value);
      }}
      placeholder={props.label}
      className={twMerge(
        "inline-flex w-full outline-none p-2 border-[2px]",
        "rounded-lg border-gray-400 focus:border-gray-900",
        "transition-all duration-300 focus:shadow-md",
        "font-semibold"
      )}
    />
  );

  /*
  return (
    <TextField
      defaultValue={props.val}
      variant="outlined"
      size="small"
      required={props.isRequired}
      fullWidth={true}
      multiline={props.multiLine}
      label={props.label}
      rows={props.rows}
      maxRows={props.rows}
      type={props.isNumber ? "number" : undefined}
      onChange={(e) => {
        props.onChange(e.target.value);
      }}
    />
  );
  */
};

export default Field;
