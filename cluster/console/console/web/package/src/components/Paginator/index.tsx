import { ListResponseMeta } from "@/apis/metav1/metav1";
import { Pagination } from "@mantine/core";
import { ArrowDownWideNarrow, ArrowUpWideNarrow } from "lucide-react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const Paginator = (props: {
  meta?: ListResponseMeta;
  onPageChange?: (page: number) => void;
}) => {
  const { meta } = props;
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const loc = useLocation();

  if (!meta) return null;

  const totalPages = Math.ceil(meta.totalCount / meta.itemsPerPage);
  const hasMultiplePages = totalPages > 1;
  const hasItems = meta.totalCount > 0;

  if (!hasItems) return null;

  const currentOrderBy =
    searchParams.get("common.orderBy.type") ?? "CREATED_AT";
  const currentOrderMode = searchParams.get("common.orderBy.mode") ?? "DESC";

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams.toString());
    next.set(key, value);
    navigate(`${loc.pathname}?${next.toString()}`);
  };

  const sortOptions: { label: string; value: string }[] = [
    { label: "Name", value: "NAME" },
    { label: "Created", value: "CREATED_AT" },
  ];

  return (
    <div className="w-full flex items-center justify-between gap-4 my-5">
      {hasMultiplePages ? (
        <Pagination
          total={totalPages}
          value={meta.page + 1}
          withEdges
          radius="md"
          color="dark"
          classNames={{
            control: twMerge(
              "!font-bold !text-[0.78rem]",
              "!border-slate-200 !bg-white",
              "!text-slate-700",
              "!shadow-[0_1px_3px_rgba(15,23,42,0.07)]",
              "hover:!bg-slate-50 hover:!border-slate-300",
              "!transition-colors !duration-150",
              "data-[active]:!bg-slate-900 data-[active]:!border-slate-900 data-[active]:!text-white",
            ),
          }}
          onChange={(v) => {
            setParam("common.page", `${v}`);
            props.onPageChange?.(v);
          }}
        />
      ) : (
        <span
          className="text-[0.75rem] font-bold text-slate-500 tracking-wide"
          style={{ textShadow: "0 1px 2px rgba(15,23,42,0.08)" }}
        >
          {meta.totalCount === 1
            ? "1 item"
            : `${meta.totalCount.toLocaleString()} items`}
        </span>
      )}

      <div className="flex items-center gap-2">
        <div className="flex items-center rounded-lg border border-slate-200 bg-white overflow-hidden shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
          {sortOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setParam("common.orderBy.type", opt.value)}
              className={twMerge(
                "cursor-pointer px-3 py-1.5 text-[0.72rem] font-bold transition-colors duration-150",
                currentOrderBy === opt.value
                  ? "bg-slate-900 text-white"
                  : "text-slate-600 hover:text-slate-900 hover:bg-slate-50",
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>

        <div className="flex items-center rounded-lg border border-slate-200 bg-white overflow-hidden shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
          {(
            [
              { value: "ASC", Icon: ArrowUpWideNarrow },
              { value: "DESC", Icon: ArrowDownWideNarrow },
            ] as const
          ).map(({ value, Icon }) => (
            <button
              key={value}
              onClick={() => setParam("common.orderBy.mode", value)}
              className={twMerge(
                "cursor-pointer flex items-center justify-center px-2.5 py-1.5 transition-colors duration-150",
                currentOrderMode === value
                  ? "bg-slate-900 text-white"
                  : "text-slate-600 hover:text-slate-900 hover:bg-slate-50",
              )}
              title={value === "ASC" ? "Ascending" : "Descending"}
            >
              <Icon size={14} />
            </button>
          ))}
        </div>

        {hasMultiplePages && (
          <span
            className="text-[0.75rem] font-bold text-slate-500 tracking-wide whitespace-nowrap"
            style={{ textShadow: "0 1px 2px rgba(15,23,42,0.08)" }}
          >
            {meta.totalCount.toLocaleString()} items
          </span>
        )}
      </div>
    </div>
  );
};

export default Paginator;
