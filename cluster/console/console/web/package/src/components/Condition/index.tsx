import { Condition } from "@/apis/corev1/corev1";
import * as React from "react";
import ConditionBuilder from "./Builder";
import { looksComplete } from "./cel";
import type { CELContext } from "./celSchema";
import {
  compileNode,
  conditionIssues,
  parseCondition,
  type ConditionNode,
} from "./model";
import type { PickerScope } from "./Picker";

export const isConditionComplete = (condition: Condition): boolean => {
  switch (condition.type.oneofKind) {
    case "matchAny":
      return condition.type.matchAny;
    case "match":
      return looksComplete(condition.type.match);
    case "not":
      return looksComplete(condition.type.not);
    case "opa":
      return (
        condition.type.opa.type.oneofKind === "inline" &&
        condition.type.opa.type.inline.trim().length > 0
      );
    case "all":
      return (
        condition.type.all.of.length > 0 &&
        condition.type.all.of.every(isConditionComplete)
      );
    case "any":
      return (
        condition.type.any.of.length > 0 &&
        condition.type.any.of.every(isConditionComplete)
      );
    case "none":
      return (
        condition.type.none.of.length > 0 &&
        condition.type.none.of.every(isConditionComplete)
      );
    default:
      return false;
  }
};

const parseItem = (item?: Condition) =>
  item?.type.oneofKind === "match" && !item.type.match.trim()
    ? undefined
    : parseCondition(item);

const Cond = (props: {
  item?: Condition;
  scope?: PickerScope;
  context?: CELContext;
  onChange: (condition: Condition) => void;
}) => {
  const [node, setNode] = React.useState<ConditionNode | undefined>(() =>
    parseItem(props.item),
  );
  const received = React.useRef(props.item);

  React.useEffect(() => {
    const item = props.item;
    if (received.current && item && Condition.equals(received.current, item)) {
      return;
    }
    if (!received.current && !item) return;
    received.current = item;
    setNode(parseItem(item));
  }, [props.item]);

  const change = (next?: ConditionNode) => {
    setNode(next);
    if (next && conditionIssues(next).length === 0) {
      const compiled = compileNode(next);
      received.current = compiled;
      props.onChange(Condition.clone(compiled));
    }
  };

  return (
    <div className="my-3 w-full">
      <ConditionBuilder
        value={node}
        onChange={change}
        scope={props.scope ?? "basic"}
        context={props.context}
      />
    </div>
  );
};

export default Cond;
