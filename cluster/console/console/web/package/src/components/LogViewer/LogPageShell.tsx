import { FilterChips, buildFilterChips } from "@/components/Paginator";
import SearchList from "@/components/SearchList";
import { Activity } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

export const useLogPageParams = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const value = Number(searchParams.get("common.page"));
  const page = Number.isInteger(value) && value > 0 ? value - 1 : 0;

  const setPage = (nextPage: number) => {
    const next = new URLSearchParams(searchParams);
    if (nextPage <= 0) next.delete("common.page");
    else next.set("common.page", String(nextPage + 1));
    setSearchParams(next, { preventScrollReset: true });
  };

  return { page, setPage };
};

const LogPageShell = (props: {
  title: string;
  description: string;
  children: React.ReactNode;
  icon?: React.ElementType<{ size?: number; strokeWidth?: number }>;
}) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const chips = buildFilterChips(searchParams);
  const Icon = props.icon ?? Activity;
  const clearFilters = () => {
    const next = new URLSearchParams(searchParams);
    for (const chip of chips) next.delete(chip.key);
    next.delete("common.query");
    next.delete("common.page");
    setSearchParams(next, { replace: true, preventScrollReset: true });
  };

  const removeFilter = (key: string) => {
    const next = new URLSearchParams(searchParams);
    next.delete(key);
    next.delete("common.page");
    const search = next.toString();
    navigate(`${location.pathname}${search ? `?${search}` : ""}`, {
      replace: true,
      preventScrollReset: true,
    });
  };

  return (
    <main className="flex w-full flex-col gap-4 py-4">
      <header className="rounded-xl border border-slate-200 bg-white px-4 py-3.5 shadow-card">
        <div className="flex items-start gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
            <Icon size={16} strokeWidth={2.25} />
          </span>
          <div className="min-w-0">
            <h1 className="text-lg font-semibold tracking-tight text-slate-950">
              {props.title}
            </h1>
            <p className="mt-0.5 text-xs leading-5 text-slate-500">
              {props.description}
            </p>
          </div>
        </div>
        <div className="mt-3 max-w-md border-t border-slate-100 pt-3">
          <SearchList
            placeholder={`Search ${props.title.toLowerCase()}`}
            label={`Search ${props.title.toLowerCase()}`}
            trimQuery
          />
        </div>
        {chips.length > 0 && (
          <div className="mt-3">
            <FilterChips
              chips={chips}
              onRemove={removeFilter}
              onClear={clearFilters}
            />
          </div>
        )}
      </header>
      {props.children}
    </main>
  );
};

export default LogPageShell;
