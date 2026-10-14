import CodeMirror from "@uiw/react-codemirror";
import { regoHighlight, regoLanguage, regoTheme } from "./opa";

export { CELEditor } from "./CELEditor";

const regoExtensions = [regoLanguage, regoHighlight, regoTheme];

export const OPAEditor = (props: {
  exp: string;
  label?: string;
  invalid?: boolean;
  onChange: (val: string) => void;
}) => {
  return (
    <div className="mt-3 w-full rounded-lg overflow-hidden border border-slate-700">
      <CodeMirror
        value={props.exp}
        aria-label={props.label ?? "OPA/Rego policy"}
        aria-invalid={props.invalid || undefined}
        tabIndex={0}
        extensions={regoExtensions}
        minHeight="160px"
        maxHeight="420px"
        basicSetup={{
          lineNumbers: true,
          foldGutter: false,
          autocompletion: false,
          highlightActiveLine: true,
        }}
        onChange={(val) => props.onChange(val)}
      />
    </div>
  );
};
