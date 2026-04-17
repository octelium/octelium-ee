import {
  Condition,
  Condition_Expression,
} from "@/apis/enterprisev1/enterprisev1";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import itemList from "./ItemList";

const kindMeta = {
  and: {
    label: "All of",
    sub: "All must match",
    badge: "bg-blue-50 text-blue-800 border-blue-200",
    border: "border-blue-200",
    sep: "AND",
    sepColor: "text-blue-600 bg-blue-50 border-blue-100",
  },
  or: {
    label: "Any of",
    sub: "At least one must match",
    badge: "bg-green-50 text-green-800 border-green-200",
    border: "border-green-200",
    sep: "OR",
    sepColor: "text-green-700 bg-green-50 border-green-100",
  },
  none: {
    label: "None of",
    sub: "No conditions may match",
    badge: "bg-orange-50 text-orange-800 border-orange-200",
    border: "border-orange-200",
    sep: "NOR",
    sepColor: "text-orange-700 bg-orange-50 border-orange-100",
  },
};

const OpBadge = ({ kind }: { kind: "and" | "or" | "none" }) => {
  const m = kindMeta[kind];
  return (
    <div className="flex items-center gap-2 py-1">
      <span
        className={twMerge(
          "text-[0.65rem] font-bold uppercase tracking-[0.08em] px-2 py-0.5 rounded border",
          m.badge,
        )}
      >
        {m.label}
      </span>
      <span className="text-[0.7rem] font-semibold text-slate-400">
        {m.sub}
      </span>
    </div>
  );
};

const Separator = ({ kind }: { kind: "and" | "or" | "none" }) => {
  const m = kindMeta[kind];
  return (
    <div className="flex items-center gap-1.5 py-0.5 pl-1">
      <span
        className={twMerge(
          "text-[0.6rem] font-bold uppercase tracking-widest px-1.5 py-px rounded border",
          m.sepColor,
        )}
      >
        {m.sep}
      </span>
    </div>
  );
};

export const ExprChip = ({ item }: { item: Condition_Expression }) => {
  const meta = itemList.find((x) => x.type === item.type.oneofKind);
  return (
    <div className="inline-flex items-center gap-1.5 flex-wrap">
      <span className="text-[0.72rem] font-bold text-slate-700 bg-slate-100 rounded px-1.5 py-0.5 border border-slate-200 whitespace-nowrap">
        {meta?.title}
      </span>
      <span className="text-[0.68rem] font-semibold text-slate-400">is</span>
      <span className="text-[0.72rem] font-bold bg-white border border-slate-200 rounded px-1.5 py-0.5 text-slate-700">
        {meta?.components.Value({ item })}
      </span>
    </div>
  );
};

const PrintCond = ({
  item,
  depth = 0,
}: {
  item: Condition;
  depth?: number;
}) => {
  return match(item.type)
    .when(
      (x) => x.oneofKind === "expression",
      (c) => <ExprChip item={c.expression} />,
    )
    .when(
      (x) => x.oneofKind === "all",
      (c) => {
        const m = kindMeta["and"];
        return (
          <div className="flex flex-col gap-0.5">
            <OpBadge kind="and" />
            <div
              className={twMerge(
                "ml-3 pl-3 border-l-2 flex flex-col gap-0.5",
                m.border,
              )}
            >
              {c.all.of.map((x, idx) => (
                <div key={idx} className="flex flex-col">
                  <PrintCond item={x} depth={depth + 1} />
                  {idx < c.all.of.length - 1 && <Separator kind="and" />}
                </div>
              ))}
            </div>
          </div>
        );
      },
    )
    .when(
      (x) => x.oneofKind === "any",
      (c) => {
        const m = kindMeta["or"];
        return (
          <div className="flex flex-col gap-0.5">
            <OpBadge kind="or" />
            <div
              className={twMerge(
                "ml-3 pl-3 border-l-2 flex flex-col gap-0.5",
                m.border,
              )}
            >
              {c.any.of.map((x, idx) => (
                <div key={idx} className="flex flex-col">
                  <PrintCond item={x} depth={depth + 1} />
                  {idx < c.any.of.length - 1 && <Separator kind="or" />}
                </div>
              ))}
            </div>
          </div>
        );
      },
    )
    .when(
      (x) => x.oneofKind === "none",
      (c) => {
        const m = kindMeta["none"];
        return (
          <div className="flex flex-col gap-0.5">
            <OpBadge kind="none" />
            <div
              className={twMerge(
                "ml-3 pl-3 border-l-2 flex flex-col gap-0.5",
                m.border,
              )}
            >
              {c.none.of.map((x, idx) => (
                <div key={idx} className="flex flex-col">
                  <PrintCond item={x} depth={depth + 1} />
                  {idx < c.none.of.length - 1 && <Separator kind="none" />}
                </div>
              ))}
            </div>
          </div>
        );
      },
    )
    .otherwise(() => null);
};

export default PrintCond;
