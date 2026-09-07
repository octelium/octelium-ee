import { TextInput } from "@mantine/core";
import { Search, X } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

const SearchList = (props: { placeholder?: string }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const currentQuery = searchParams.get("common.query") ?? "";
  const [query, setQuery] = React.useState(currentQuery);
  const inputRef = React.useRef<HTMLInputElement>(null);

  React.useEffect(() => {
    setQuery(currentQuery);
  }, [currentQuery]);

  React.useEffect(() => {
    if (query === currentQuery) return;

    const timeout = window.setTimeout(() => {
      const next = new URLSearchParams(location.search);
      if (query.trim()) {
        next.set("common.query", query);
      } else {
        next.delete("common.query");
      }
      next.delete("common.page");
      const search = next.toString();
      navigate(`${location.pathname}${search ? `?${search}` : ""}`, {
        replace: true,
        preventScrollReset: true,
      });
    }, 250);

    return () => window.clearTimeout(timeout);
  }, [currentQuery, location.pathname, location.search, navigate, query]);

  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const isTyping =
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.isContentEditable;

      if (event.key === "/" && !isTyping) {
        event.preventDefault();
        inputRef.current?.focus();
      }

      const input = inputRef.current;
      if (event.key === "Escape" && input && document.activeElement === input) {
        input.blur();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  return (
    <TextInput
      ref={inputRef}
      value={query}
      onChange={(event) => setQuery(event.currentTarget.value)}
      aria-label="Search resources"
      placeholder={props.placeholder ?? "Search…"}
      radius="md"
      leftSection={<Search size={15} strokeWidth={2.2} />}
      leftSectionPointerEvents="none"
      rightSectionWidth={query ? 36 : 30}
      rightSection={
        query ? (
          <button
            type="button"
            onClick={() => {
              setQuery("");
              inputRef.current?.focus();
            }}
            aria-label="Clear search"
            title="Clear search"
            className="flex h-6 w-6 items-center justify-center rounded-md text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
          >
            <X size={14} strokeWidth={2.4} />
          </button>
        ) : (
          <kbd className="hidden min-w-5 rounded border border-slate-200 bg-slate-50 px-1 py-0.5 text-center font-mono text-xs font-normal text-slate-500 sm:inline-block">
            /
          </kbd>
        )
      }
      styles={{ root: { width: "100%" } }}
    />
  );
};

export default SearchList;
