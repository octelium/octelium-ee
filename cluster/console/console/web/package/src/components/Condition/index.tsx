import { Condition } from "@/apis/corev1/corev1";
import { SegmentedControl } from "@mantine/core";
import * as React from "react";
import { match } from "ts-pattern";
import EditItem from "../EditItem";
import ItemMessage from "../ItemMessage";
import ConditionBuilderBtn from "../PolicyBuilder/ConditionBuilderBtn";
import { CELEditor, OPAEditor } from "./Editor";

const CONDITION_TYPES = [
  { value: "match", label: "Match" },
  { value: "not", label: "Not" },
  { value: "all", label: "All (AND)" },
  { value: "any", label: "Any (OR)" },
  { value: "none", label: "None (NOR)" },
  { value: "opa", label: "OPA" },
  { value: "matchAny", label: "Match All" },
] as const;

const makeCondition = (kind: string): Condition => {
  const base = (type: object) => Condition.create({ type: type as any });
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

const Cond = (props: {
  item?: Condition;
  onChange: (condition: Condition) => void;
}) => {
  const { item } = props;

  const [req, setReq] = React.useState<Condition>(
    item ?? makeCondition("match"),
  );
  const [init] = React.useState<Condition>(() => Condition.clone(req));

  const updateReq = (next: Condition) => {
    setReq(Condition.clone(next));
    props.onChange(next);
  };

  React.useEffect(() => {
    setReq(item ?? makeCondition("match"));
  }, [item]);

  const handleTypeChange = (v: string) => {
    const restored = Condition.clone(init);
    if (restored.type.oneofKind === v) {
      updateReq(Object.assign(Condition.clone(req), { type: restored.type }));
    } else {
      updateReq(makeCondition(v));
    }
  };

  const makeListItemMessage = (
    title: string,
    of: Condition[],
    setOf: (items: Condition[]) => void,
  ) => (
    <ItemMessage
      title={title}
      obj={of}
      isList
      onSet={() => setOf([makeCondition("match")])}
      onAddListItem={() => setOf([...of, makeCondition("match")])}
    >
      {of.map((x, idx) => (
        <EditItem
          key={idx}
          obj={of[idx]}
          onUnset={() => {
            const next = of.filter((_, i) => i !== idx);
            setOf(next);
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

  return (
    <div className="w-full my-3">
      <div className="flex flex-col gap-3">
        <SegmentedControl
          value={req.type.oneofKind}
          data={CONDITION_TYPES as any}
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
            onChange={(v) => updateReq(v ?? Condition.create())}
          />
        </div>
      </div>

      <div className="mt-2">
        {match(req.type)
          .when(
            (x) => x.oneofKind === "all",
            (t) =>
              makeListItemMessage("All conditions (AND)", t.all.of, (items) => {
                t.all.of = items;
                updateReq(Condition.clone(req));
              }),
          )
          .when(
            (x) => x.oneofKind === "any",
            (t) =>
              makeListItemMessage("Any condition (OR)", t.any.of, (items) => {
                t.any.of = items;
                updateReq(Condition.clone(req));
              }),
          )
          .when(
            (x) => x.oneofKind === "none",
            (t) =>
              makeListItemMessage("None of (NOR)", t.none.of, (items) => {
                t.none.of = items;
                updateReq(Condition.clone(req));
              }),
          )
          .when(
            (x) => x.oneofKind === "match",
            (t) => (
              <CELEditor
                exp={t.match}
                onChange={(v) => {
                  t.match = v;
                  updateReq(Condition.clone(req));
                }}
              />
            ),
          )
          .when(
            (x) => x.oneofKind === "not",
            (t) => (
              <CELEditor
                exp={t.not}
                onChange={(v) => {
                  t.not = v;
                  updateReq(Condition.clone(req));
                }}
              />
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
