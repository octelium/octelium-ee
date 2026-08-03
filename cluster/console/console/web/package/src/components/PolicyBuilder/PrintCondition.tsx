import {
  Condition,
  Condition_Expression,
} from "@/apis/enterprisev1/enterprisev1";
import { Braces, Check, Sparkles } from "lucide-react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import itemList from "./ItemList";

const kindMeta = {
  and: {
    label: "All of",
    description: "Every condition must match",
    badge: "border-blue-200 bg-blue-50 text-blue-700",
    border: "border-blue-200",
    separator: "AND",
    separatorStyle: "border-blue-100 bg-blue-50 text-blue-600",
  },
  or: {
    label: "Any of",
    description: "At least one condition must match",
    badge: "border-emerald-200 bg-emerald-50 text-emerald-700",
    border: "border-emerald-200",
    separator: "OR",
    separatorStyle: "border-emerald-100 bg-emerald-50 text-emerald-700",
  },
  none: {
    label: "None of",
    description: "No conditions may match",
    badge: "border-amber-200 bg-amber-50 text-amber-700",
    border: "border-amber-200",
    separator: "NOR",
    separatorStyle: "border-amber-100 bg-amber-50 text-amber-700",
  },
};

const GroupHeading = ({ kind }: { kind: "and" | "or" | "none" }) => {
  const meta = kindMeta[kind];
  return (
    <div className="flex flex-wrap items-center gap-2 py-1">
      <span
        className={twMerge(
          "rounded-md border px-2 py-0.5 text-[0.6rem] font-bold uppercase tracking-[0.07em]",
          meta.badge,
        )}
      >
        {meta.label}
      </span>
      <span className="text-[0.65rem] font-semibold text-slate-400">
        {meta.description}
      </span>
    </div>
  );
};

const Separator = ({ kind }: { kind: "and" | "or" | "none" }) => {
  const meta = kindMeta[kind];
  return (
    <div className="flex items-center py-1 pl-1">
      <span
        className={twMerge(
          "rounded border px-1.5 py-px text-[0.56rem] font-bold uppercase tracking-widest",
          meta.separatorStyle,
        )}
      >
        {meta.separator}
      </span>
    </div>
  );
};

export const ExprChip = ({ item }: { item: Condition_Expression }) => {
  const definition = itemList.find(
    (candidate) => candidate.type === item.type.oneofKind,
  );

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-1.5">
      <span className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-slate-100 px-2 py-1 text-[0.7rem] font-bold text-slate-700">
        <Braces size={11} strokeWidth={2.25} className="text-slate-400" />
        {definition?.title ?? "Unknown expression"}
      </span>
      <span className="text-[0.65rem] font-semibold text-slate-400">is</span>
      <span className="min-w-0 rounded-md border border-slate-200 bg-white px-2 py-1 text-[0.7rem] font-bold text-slate-700">
        {definition?.components.Value({ item }) ?? "Not configured"}
      </span>
    </div>
  );
};

const duplicateOccurrence = (
  items: Condition[],
  index: number,
  serialized: string,
) =>
  items
    .slice(0, index)
    .filter((item) => Condition.toJsonString(item) === serialized).length;

const GroupPreview = ({
  kind,
  items,
  depth,
}: {
  kind: "and" | "or" | "none";
  items: Condition[];
  depth: number;
}) => {
  const meta = kindMeta[kind];
  return (
    <div className="flex flex-col gap-1">
      <GroupHeading kind={kind} />
      <div
        className={twMerge(
          "ml-2 flex flex-col border-l-2 pl-3",
          meta.border,
        )}
      >
        {items.map((item, index) => {
          const serialized = Condition.toJsonString(item);
          return (
            <div
              key={`${serialized}-${duplicateOccurrence(items, index, serialized)}`}
              className="flex flex-col"
            >
              <PrintCond item={item} depth={depth + 1} />
              {index < items.length - 1 && <Separator kind={kind} />}
            </div>
          );
        })}
      </div>
    </div>
  );
};

const PrintCond = ({
  item,
  depth = 0,
}: {
  item: Condition;
  depth?: number;
}) =>
  match(item.type)
    .with({ oneofKind: "expression" }, (type) => (
      <ExprChip item={type.expression} />
    ))
    .with({ oneofKind: "all" }, (type) => (
      <GroupPreview kind="and" items={type.all.of} depth={depth} />
    ))
    .with({ oneofKind: "any" }, (type) => (
      <GroupPreview kind="or" items={type.any.of} depth={depth} />
    ))
    .with({ oneofKind: "none" }, (type) => (
      <GroupPreview kind="none" items={type.none.of} depth={depth} />
    ))
    .with({ oneofKind: "not" }, (type) => (
      <div className="rounded-lg border border-red-100 bg-red-50/50 p-2.5">
        <div className="mb-2 flex items-center gap-2">
          <span className="rounded-md border border-red-200 bg-white px-2 py-0.5 text-[0.58rem] font-bold uppercase tracking-[0.07em] text-red-600">
            NOT
          </span>
          <span className="text-[0.63rem] font-semibold text-red-700/70">
            Result is inverted
          </span>
        </div>
        {type.not.expression ? (
          <ExprChip item={type.not.expression} />
        ) : (
          <span className="text-[0.68rem] font-semibold text-red-600/70">
            Negated expression is not configured
          </span>
        )}
      </div>
    ))
    .with({ oneofKind: "matchAny" }, () => (
      <div className="flex items-center gap-2.5 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
        <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-slate-900 text-white">
          <Sparkles size={11} strokeWidth={2.25} />
        </span>
        <div>
          <p className="text-[0.7rem] font-bold text-slate-700">
            Match everything
          </p>
          <p className="mt-0.5 text-[0.62rem] font-semibold text-slate-400">
            No restrictions are applied
          </p>
        </div>
        <Check size={12} className="ml-auto text-emerald-600" />
      </div>
    ))
    .otherwise(() => (
      <div className="rounded-lg border border-dashed border-slate-200 px-3 py-3 text-center text-[0.68rem] font-semibold text-slate-400">
        Condition is not configured
      </div>
    ));

export default PrintCond;
