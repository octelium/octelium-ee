import { ListResponseMeta } from "@/apis/metav1/metav1";
import { Button, Pagination } from "@mantine/core";
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

  const itemCountLabel =
    meta.totalCount === 1
      ? "1 item"
      : `${meta.totalCount.toLocaleString()} items`;

  const btnStyles = (active: boolean) => ({
    root: {
      height: "28px",
      fontSize: "0.72rem",
      fontWeight: 700,
      padding: "0 10px",
      backgroundColor: active ? "#0f172a" : "#ffffff",
      color: active ? "#ffffff" : "#64748b",
      border: "none",
      borderRadius: 0,
      transition: "background-color 150ms, color 150ms",
      "&:hover": {
        backgroundColor: active ? "#1e293b" : "#f8fafc",
        color: active ? "#ffffff" : "#0f172a",
      },
    },
  });

  const iconBtnStyles = (active: boolean) => ({
    root: {
      height: "28px",
      width: "28px",
      padding: 0,
      backgroundColor: active ? "#0f172a" : "#ffffff",
      color: active ? "#ffffff" : "#64748b",
      border: "none",
      borderRadius: 0,
      transition: "background-color 150ms, color 150ms",
      "&:hover": {
        backgroundColor: active ? "#1e293b" : "#f8fafc",
        color: active ? "#ffffff" : "#0f172a",
      },
    },
  });

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
              "!border-slate-200 !bg-white !text-slate-700",
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
          {itemCountLabel}
        </span>
      )}

      <div className="flex items-center gap-1.5">
        <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
          {[
            { label: "Name", value: "NAME" },
            { label: "Created", value: "CREATED_AT" },
          ].map((opt) => (
            <Button
              key={opt.value}
              onClick={() => setParam("common.orderBy.type", opt.value)}
              styles={btnStyles(currentOrderBy === opt.value)}
            >
              {opt.label}
            </Button>
          ))}
        </Button.Group>

        <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
          {(
            [
              { value: "ASC", Icon: ArrowUpWideNarrow },
              { value: "DESC", Icon: ArrowDownWideNarrow },
            ] as const
          ).map(({ value, Icon }) => (
            <Button
              key={value}
              onClick={() => setParam("common.orderBy.mode", value)}
              styles={iconBtnStyles(currentOrderMode === value)}
              title={value === "ASC" ? "Ascending" : "Descending"}
            >
              <Icon size={13} />
            </Button>
          ))}
        </Button.Group>

        {hasMultiplePages && (
          <span
            className="text-[0.75rem] font-bold text-slate-500 tracking-wide whitespace-nowrap ml-1"
            style={{ textShadow: "0 1px 2px rgba(15,23,42,0.08)" }}
          >
            {itemCountLabel}
          </span>
        )}
      </div>
    </div>
  );
};

export default Paginator;
