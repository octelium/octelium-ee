import {
  Condition,
  Condition_Expression,
} from "@/apis/enterprisev1/enterprisev1";
import clsx from "clsx";
import { match } from "ts-pattern";
import itemList from "./ItemList";

const OpBadge = ({ kind }: { kind: "and" | "or" | "none" }) => {
  const labels = { and: "All of", or: "Any of", none: "None of" };
  const sublabels = {
    and: "All conditions must match",
    or: "At least one must match",
    none: "No conditions may match",
  };
  return (
    <div className="flex items-center gap-2 py-1.5 font-bold">
      <span
        className={clsx(
          "pcb-op text-[10px] font-bold uppercase tracking-wide px-2 py-0.5 rounded",
          {
            "bg-blue-50 text-blue-900 border border-blue-200 ": kind === "and",
            "bg-green-50 text-green-900 border border-green-200 ":
              kind === "or",
            "bg-orange-50 text-orange-900 border border-orange-200 ":
              kind === "none",
          },
        )}
      >
        {labels[kind]}
      </span>
      <span className="text-xs text-gray-400">{sublabels[kind]}</span>
    </div>
  );
};

export const ExprChip = ({ item }: { item: Condition_Expression }) => {
  const meta = itemList.find((x) => x.type === item.type.oneofKind);
  return (
    <div className="font-bold flex items-center gap-1.5 flex-wrap bg-white border border-gray-200 rounded-lg px-3 py-2 text-sm hover:border-gray-300 transition-colors">
      <span className="text-[14px] font-bold text-gray-800 bg-gray-100 rounded px-1.5 py-0.5 whitespace-nowrap">
        {meta?.title}
      </span>
      <span className="text-[12px] text-gray-700">is</span>
      <span className="text-[14px] bg-gray-50 border border-gray-200 rounded px-1.5 py-0.5 font-bold">
        {meta?.components.Value({ item })}
      </span>
    </div>
  );
};

const Separator = ({ kind }: { kind: "and" | "or" | "none" }) => {
  const labels = { and: "AND", or: "OR", none: "NOR" };
  return (
    <div className="flex items-center gap-2 py-0.5 relative">
      <span
        className={clsx(
          "text-[11px] font-bold uppercase tracking-widest px-1.5 py-0.5 rounded",
          {
            "text-blue-600 bg-blue-50 border border-blue-100": kind === "and",
            "text-green-700 bg-green-50 border border-green-100": kind === "or",
            "text-orange-700 bg-orange-50 border border-orange-100":
              kind === "none",
          },
        )}
      >
        {labels[kind]}
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
      (c) => (
        <div className="flex flex-col">
          <OpBadge kind="and" />
          <div className="ml-3.5 pl-4 border-l border-blue-200 flex flex-col gap-0.5">
            {c.all.of.map((x, idx) => (
              <div
                key={idx}
                className="relative flex flex-col before:absolute before:left-[-16px] before:top-5 before:w-3 before:h-px before:bg-gray-200"
              >
                <PrintCond item={x} depth={depth + 1} />
                {idx < c.all.of.length - 1 && <Separator kind="and" />}
              </div>
            ))}
          </div>
        </div>
      ),
    )
    .when(
      (x) => x.oneofKind === "any",
      (c) => (
        <div className="flex flex-col">
          <OpBadge kind="or" />
          <div className="ml-3.5 pl-4 border-l border-green-200 flex flex-col gap-0.5">
            {c.any.of.map((x, idx) => (
              <div
                key={idx}
                className="relative flex flex-col before:absolute before:left-[-16px] before:top-5 before:w-3 before:h-px before:bg-gray-200"
              >
                <PrintCond item={x} depth={depth + 1} />
                {idx < c.any.of.length - 1 && <Separator kind="or" />}
              </div>
            ))}
          </div>
        </div>
      ),
    )
    .when(
      (x) => x.oneofKind === "none",
      (c) => (
        <div className="flex flex-col">
          <OpBadge kind="none" />
          <div className="ml-3.5 pl-4 border-l border-orange-200 flex flex-col gap-0.5">
            {c.none.of.map((x, idx) => (
              <div
                key={idx}
                className="relative flex flex-col before:absolute before:left-[-16px] before:top-5 before:w-3 before:h-px before:bg-gray-200"
              >
                <PrintCond item={x} depth={depth + 1} />
                {idx < c.none.of.length - 1 && <Separator kind="none" />}
              </div>
            ))}
          </div>
        </div>
      ),
    )
    .otherwise(() => null);
};

export default PrintCond;
