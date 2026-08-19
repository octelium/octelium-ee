import { Condition } from "@/apis/corev1/corev1";
import { SegmentedControl } from "@mantine/core";
import * as React from "react";
import { match } from "ts-pattern";
import EditItem from "../EditItem";
import ItemMessage from "../ItemMessage";
import ConditionBuilderBtn from "../PolicyBuilder/ConditionBuilderBtn";
import { CELEditor, OPAEditor } from "./Editor";

type ConditionKind =
  | "match"
  | "not"
  | "all"
  | "any"
  | "none"
  | "opa"
  | "matchAny";

const CONDITION_TYPES: Array<{ value: ConditionKind; label: string }> = [
  { value: "match", label: "Match" },
  { value: "not", label: "Not" },
  { value: "all", label: "All (AND)" },
  { value: "any", label: "Any (OR)" },
  { value: "none", label: "None (NOR)" },
  { value: "opa", label: "OPA" },
  { value: "matchAny", label: "Match everything" },
];

const makeCondition = (kind: ConditionKind): Condition => {
  const base = (type: Condition["type"]) => Condition.create({ type });
  switch (kind) {
    case "match":
      return base({ oneofKind: "match", match: "" });
    case "not":
      return base({ oneofKind: "not", not: "" });
    case "all":
      return base({ oneofKind: "all", all: { of: [makeCondition("match")] } });
    case "any":
      return base({ oneofKind: "any", any: { of: [makeCondition("match")] } });
    case "none":
      return base({
        oneofKind: "none",
        none: { of: [makeCondition("match")] },
      });
    case "opa":
      return base({
        oneofKind: "opa",
        opa: { type: { oneofKind: "inline", inline: "" } },
      });
    case "matchAny":
      return base({ oneofKind: "matchAny", matchAny: true });
    default:
      return base({ oneofKind: "match", match: "" });
  }
};

const expressionLooksComplete = (value: string) => {
  if (!value.trim()) return false;
  const stack: string[] = [];
  let quote: string | undefined;
  let escaped = false;
  for (const char of value) {
    if (quote) {
      if (escaped) {
        escaped = false;
      } else if (char === "\\") {
        escaped = true;
      } else if (char === quote) {
        quote = undefined;
      }
      continue;
    }
    if (char === '"' || char === "'") {
      quote = char;
      continue;
    }
    if (char === "(" || char === "[" || char === "{") {
      stack.push(char);
    } else if (char === ")" || char === "]" || char === "}") {
      const opening = stack.pop();
      if (
        (char === ")" && opening !== "(") ||
        (char === "]" && opening !== "[") ||
        (char === "}" && opening !== "{")
      ) {
        return false;
      }
    }
  }
  return !quote && stack.length === 0;
};

const isConditionKind = (value: string): value is ConditionKind =>
  CONDITION_TYPES.some((item) => item.value === value);

export const isConditionComplete = (condition: Condition): boolean => {
  switch (condition.type.oneofKind) {
    case "matchAny":
      return condition.type.matchAny;
    case "match":
      return expressionLooksComplete(condition.type.match);
    case "not":
      return expressionLooksComplete(condition.type.not);
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

const conditionValidationMessage = (condition: Condition): string | undefined => {
  switch (condition.type.oneofKind) {
    case "match":
      return expressionLooksComplete(condition.type.match)
        ? undefined
        : "Enter a complete CEL expression.";
    case "not":
      return expressionLooksComplete(condition.type.not)
        ? undefined
        : "Enter a complete CEL expression to negate.";
    case "opa":
      return condition.type.opa.type.oneofKind === "inline" &&
        condition.type.opa.type.inline.trim()
        ? undefined
        : "Enter an inline OPA/Rego policy.";
    case "all":
    case "any":
    case "none": {
      const items =
        condition.type.oneofKind === "all"
          ? condition.type.all.of
          : condition.type.oneofKind === "any"
            ? condition.type.any.of
            : condition.type.none.of;
      return items.length === 0
        ? "Add at least one nested condition."
        : items.find((item) => !isConditionComplete(item))
          ? "Complete each nested condition."
          : undefined;
    }
    case "matchAny":
      return condition.type.matchAny ? undefined : "Enable match everything.";
    default:
      return "Choose a condition type.";
  }
};

const Cond = (props: {
  item?: Condition;
  onChange: (condition: Condition) => void;
}) => {
  const { item } = props;

  const [req, setReq] = React.useState<Condition>(
    () => (item ? Condition.clone(item) : makeCondition("match")),
  );
  const listIDs = React.useRef<string[]>([]);
  const listKind = React.useRef<"all" | "any" | "none">(undefined);

  const updateReq = (next: Condition) => {
    const cloned = Condition.clone(next);
    setReq(cloned);
    if (isConditionComplete(cloned)) {
      props.onChange(Condition.clone(cloned));
    }
  };

  React.useEffect(() => {
    setReq(item ? Condition.clone(item) : makeCondition("match"));
  }, [item]);

  const handleTypeChange = (v: string) => {
    if (isConditionKind(v) && req.type.oneofKind !== v) {
      listIDs.current = [];
      listKind.current = undefined;
      updateReq(makeCondition(v));
    }
  };

  const makeListItemMessage = (
    kind: "all" | "any" | "none",
    title: string,
    of: Condition[],
    setOf: (items: Condition[]) => void,
  ) => {
    if (listKind.current !== kind) {
      listKind.current = kind;
      listIDs.current = [];
    }
    while (listIDs.current.length < of.length) {
      listIDs.current.push(crypto.randomUUID());
    }
    if (listIDs.current.length > of.length) {
      listIDs.current = listIDs.current.slice(0, of.length);
    }

    return (
      <ItemMessage
        title={title}
        obj={of}
        isList
        onSet={() => {
          listIDs.current = [crypto.randomUUID()];
          setOf([makeCondition("match")]);
        }}
        onAddListItem={() => {
          listIDs.current.push(crypto.randomUUID());
          setOf([...of, makeCondition("match")]);
        }}
      >
        {of.map((x, idx) => (
          <EditItem
            key={listIDs.current[idx]}
            obj={of[idx]}
            onUnset={() => {
              listIDs.current.splice(idx, 1);
              setOf(of.filter((_, i) => i !== idx));
            }}
          >
            <Cond
              item={x}
              onChange={(v) => {
                const next = [...of];
                next[idx] = v;
                setOf(next);
              }}
            />
          </EditItem>
        ))}
      </ItemMessage>
    );
  };

  const validationMessage = conditionValidationMessage(req);

  return (
    <div className="w-full my-3">
      <div className="flex flex-col gap-3">
        <SegmentedControl
          value={req.type.oneofKind ?? "match"}
          data={CONDITION_TYPES}
          onChange={handleTypeChange}
          size="xs"
          styles={{
            root: {
              flexWrap: "wrap",
              height: "auto",
              gap: "2px",
            },
          }}
        />

        <div className="flex items-center">
          <ConditionBuilderBtn
            onChange={(v) => v && updateReq(v)}
          />
        </div>
      </div>

      {validationMessage && (
        <p className="mt-2 text-xs font-semibold text-amber-700" role="status">
          {validationMessage}
        </p>
      )}

      <div className="mt-2">
        {match(req.type)
          .when(
            (x) => x.oneofKind === "all",
            (t) =>
              makeListItemMessage("all", "All conditions (AND)", t.all.of, (items) => {
                t.all.of = items;
                updateReq(Condition.clone(req));
              }),
          )
          .when(
            (x) => x.oneofKind === "any",
            (t) =>
              makeListItemMessage("any", "Any condition (OR)", t.any.of, (items) => {
                t.any.of = items;
                updateReq(Condition.clone(req));
              }),
          )
          .when(
            (x) => x.oneofKind === "none",
            (t) =>
              makeListItemMessage("none", "None of (NOR)", t.none.of, (items) => {
                t.none.of = items;
                updateReq(Condition.clone(req));
              }),
          )
          .when(
            (x) => x.oneofKind === "match",
            (t) => (
              <div>
                <p className="text-[0.68rem] font-bold text-slate-600">CEL expression</p>
                <CELEditor
                  label="CEL expression"
                  invalid={!isConditionComplete(req)}
                  exp={t.match}
                  onChange={(v) => {
                    t.match = v;
                    updateReq(Condition.clone(req));
                  }}
                />
              </div>
            ),
          )
          .when(
            (x) => x.oneofKind === "not",
            (t) => (
              <div>
                <p className="text-[0.68rem] font-bold text-slate-600">CEL expression to negate</p>
                <CELEditor
                  label="CEL expression to negate"
                  invalid={!isConditionComplete(req)}
                  exp={t.not}
                  onChange={(v) => {
                    t.not = v;
                    updateReq(Condition.clone(req));
                  }}
                />
              </div>
            ),
          )
          .when(
            (x) => x.oneofKind === "opa",
            (t) =>
              match(t.opa.type)
                .when(
                  (x) => x.oneofKind === "inline",
                  (inline) => (
                    <OPAEditor
                      label="OPA/Rego policy"
                      invalid={!isConditionComplete(req)}
                      exp={inline.inline}
                      onChange={(v) => {
                        inline.inline = v;
                        updateReq(Condition.clone(req));
                      }}
                    />
                  ),
                )
                .otherwise(() => null),
          )
          .otherwise(() => null)}
      </div>
    </div>
  );
};

export default Cond;
