import { Resource, resourceToYAML } from "@/utils/pb";

import Editor from "../Editor";

const ResourceEditor = (props: {
  item: Resource;
  size?: "xs" | "small";

  onResourceChange?: (arg: Resource) => void;
  readOnly?: boolean;
  mode?: "json" | "yaml" | undefined;
  onChange?: (arg: string) => void;
}) => {
  const { item } = props;

  return (
    <div className="rounded-none">
      <div className="w-full py-4 px-2">
        <div>
          <Editor
            item={props.item}
            mode={"yaml"}
            onResourceChange={(n) => {}}
            onChange={(v) => {
              if (props.onChange) {
                props.onChange(v);
              }
            }}
            value={resourceToYAML(item)}
            readOnly={props.readOnly}
          />
        </div>
      </div>
    </div>
  );
};

export default ResourceEditor;
