import { FilterChips, buildFilterChips } from "@/components/Paginator";
import { TextInput } from "@mantine/core";
import { Activity, Search } from "lucide-react";
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
  const queryParam = searchParams.get("common.query") ?? "";
  const [query, setQuery] = React.useState(queryParam);

  React.useEffect(() => setQuery(queryParam), [queryParam]);

  React.useEffect(() => {
    if (query.trim() === queryParam) return;
    const timeout = window.setTimeout(() => {
      const next = new URLSearchParams(searchParams);
      const value = query.trim();
      if (value) next.set("common.query", value);
      else next.delete("common.query");
      next.delete("common.page");
      setSearchParams(next, { replace: true, preventScrollReset: true });
    }, 350);
    return () => window.clearTimeout(timeout);
  }, [query, queryParam, searchParams, setSearchParams]);

  const removeFilter = (key: string) => {
    const next = new URLSearchParams(searchParams);
    next.delete(key);
    next.delete("common.page");
    const search = next.toString();
    navigate(
      `${location.pathname}${search ? `?${search}` : ""}`,
      { replace: true, preventScrollReset: true },
    );
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
        <div className="mt-3 border-t border-slate-100 pt-3">
          <TextInput
            size="xs"
            value={query}
            onChange={(event) => setQuery(event.currentTarget.value)}
            leftSection={<Search size={13} />}
            placeholder={`Search ${props.title.toLowerCase()}`}
            aria-label={`Search ${props.title.toLowerCase()}`}
            className="max-w-md"
          />
        </div>
        {chips.length > 0 && (
          <div className="mt-3">
            <FilterChips chips={chips} onRemove={removeFilter} />
          </div>
        )}
      </header>
      {props.children}
    </main>
  );
};

export default LogPageShell;
