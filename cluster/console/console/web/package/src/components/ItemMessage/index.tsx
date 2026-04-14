import * as React from "react";

import { Button, Collapse } from "@mantine/core";

import { ChevronDown, ChevronUp, Plus } from "lucide-react";
import { twMerge } from "tailwind-merge";

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
    <div className="mt-6 mb-16">
      <div
        className="pb-2 pd-2 font-bold text-sm text-black mb-8 border-b-[1px] border-b-gray-300 cursor-pointer flex items-center"
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
            isExpanded ? "text-black" : "text-gray-600",
            "mr-2",
          )}
        >
          {props.title}
        </span>{" "}
        {props.isList && (
          <Button
            size="xs"
            onClick={(e) => {
              e.stopPropagation();
              if (props.onAddListItem) {
                props.onAddListItem();
                setIsExpanded(true);
              }
            }}
          >
            <div className="flex items-center">
              <Plus />{" "}
              <span className="ml-1">
                {`Add Item (${arrLen == 1 ? "1 Item" : `${arrLen} Items`})`}
              </span>
            </div>
          </Button>
        )}
      </div>

      {props.obj && (
        <Collapse in={isExpanded}>
          <div className="ml-4 mb-4">
            <div>{props.children}</div>
          </div>
        </Collapse>
      )}

      <div className="my-4 flex items-center justify-center w-full">
        {props.isList && (
          <Button
            size="xs"
            onClick={(e) => {
              e.stopPropagation();
              if (props.onAddListItem) {
                props.onAddListItem();
                setIsExpanded(true);
              }
            }}
          >
            <div className="flex items-center">
              <Plus />{" "}
              <span className="ml-1">
                {`Add Item (${arrLen == 1 ? "1 Item" : `${arrLen} Items`})`}
              </span>
            </div>
          </Button>
        )}
      </div>
    </div>
  );
};

export default ItemMessage;
