import { FilterChips, buildFilterChips } from "@/components/Paginator";
import SearchList from "@/components/SearchList";
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

const LogPageShell = (props: { title: string; children: React.ReactNode }) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const chips = buildFilterChips(searchParams);
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
    <div className="flex w-full flex-col gap-4 py-4">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-3">
        <div className="w-full max-w-md">
          <SearchList
            placeholder={`Search ${props.title.toLowerCase()}`}
            label={`Search ${props.title.toLowerCase()}`}
            trimQuery
          />
        </div>
        {chips.length > 0 && (
          <FilterChips
            chips={chips}
            onRemove={removeFilter}
            onClear={clearFilters}
          />
        )}
      </div>
      {props.children}
    </div>
  );
};

export default LogPageShell;
