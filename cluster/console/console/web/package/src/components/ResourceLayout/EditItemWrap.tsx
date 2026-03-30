import { Button } from "@mantine/core";
import * as React from "react";
import { MdOutlineEdit, MdOutlineEditOff } from "react-icons/md";

const EditItemWrap = (props: {
  showComponent: React.ReactNode;
  editComponent: React.ReactNode;
}) => {
  let [enabled, setEnabled] = React.useState(false);

  return (
    <div className="w-full flex items-center">
      <Button
        variant="transparent"
        size="compact-xs"
        className="mr-[2px] hover:bg-slate-300 transition-all duration-500 py-1 px-1 rounded-full"
        onClick={() => {
          setEnabled(!enabled);
        }}
      >
        {enabled ? <MdOutlineEditOff size={14} /> : <MdOutlineEdit size={14} />}
      </Button>
      {enabled && <div>{props.editComponent}</div>}
      {!enabled && <div>{props.showComponent}</div>}
    </div>
  );
};

export default EditItemWrap;
