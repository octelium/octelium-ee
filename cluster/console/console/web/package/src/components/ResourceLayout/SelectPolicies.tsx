import SelectResourceMultiple from "./SelectResourceMultiple";

const SelectPolicies = (props: {
  policies?: string[];
  onUpdate: (policies?: string[]) => void;
}) => {
  return (
    <div className="w-full">
      {
        <SelectResourceMultiple
          api="core"
          kind="Policy"
          label="Policies"
          description="Choose one or more Policies"
          defaultValue={props.policies}
          clearable
          onChange={(v) => {
            props.onUpdate(v?.map((x) => x.metadata!.name) ?? undefined);
          }}
        />
      }
    </div>
  );
};

export default SelectPolicies;
