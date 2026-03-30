import * as React from "react";

import { Collapse } from "@mantine/core";

import { twMerge } from "tailwind-merge";
import Label from "../Label";
import { Plus, ChevronDown, ChevronUp } from "lucide-react";

interface Props {
  children?: React.ReactNode;
  title: string;
  obj?: object | Array<any> | undefined;
  onSet?: () => void;
  isList?: boolean;
  onAddListItem?: () => void;
}

const ItemMessage = (props: Props) => {
  let arrLen = 0;
  let arr = props.obj as Array<any> | undefined;
  if (props.isList) {
    arrLen = arr?.length ?? 0;
  }

  const isObjNull = !props.obj;

  let [isExpanded, setIsExpanded] = React.useState(arrLen > 0 ? true : false);

  return (
    <div className="mt-6">
      <div
        className="font-bold text-sm text-black mb-4 border-b-[1px] border-b-gray-300 cursor-pointer flex items-center"
        onClick={() => {
          if (isObjNull) {
            if (props.onSet) {
              props.onSet();
              setIsExpanded(true);
            }
          } else {
            setIsExpanded(!isExpanded);
          }

          if (props.isList && arrLen < 1 && props.onAddListItem) {
            props.onAddListItem();
          }
        }}
      >
        <button
          className="text-gray-600 font-bold text-sm hover:text-gray-800 mr-2 transition-all duration-300"
          onClick={() => {
            setIsExpanded(!isExpanded);
          }}
        >
          {isExpanded ? <ChevronUp /> : <ChevronDown />}
        </button>
        <span
          className={twMerge(
            isObjNull ? "text-gray-600" : "text-black",
            "mr-2"
          )}
        >
          {props.title}
        </span>{" "}
        {props.isList && (
          <Label>{`List (${
            arrLen == 1 ? "1 Item" : `${arrLen} Items`
          })`}</Label>
        )}
        {props.isList && props.onAddListItem && (
          <button
            className={twMerge(
              "inline-flex items-center mx-2",
              "bg-black text-white font-bold p-1 rounded-full text-xs",
              "shadow",
              "transition-all duration-300",
              "hover:bg-gray-800 cursor-pointer",
              "mr-2"
            )}
            onClick={(e) => {
              e.stopPropagation();
              if (props.onAddListItem) {
                props.onAddListItem();
                setIsExpanded(true);
              }
            }}
          >
            <Plus />
          </button>
        )}
        {/*
        {!props.obj && (
          <Button
            onClick={(e) => {
              e.stopPropagation();
              if (props.onSet) {
                props.onSet();
                setIsExpanded(true);
              }
            }}
          >
            Not set (Set!)
          </Button>
        )}
        */}
      </div>

      {props.obj && (
        <Collapse in={isExpanded}>
          <div className="ml-4 mb-4">
            <div>{props.children}</div>
          </div>
        </Collapse>
      )}
    </div>
  );
};

export default ItemMessage;
