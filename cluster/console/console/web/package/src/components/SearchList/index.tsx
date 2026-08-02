import { TextInput } from "@mantine/core";
import { Search, X } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

const SearchList = () => {
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
      const search = next.toString();
      navigate(`${location.pathname}${search ? `?${search}` : ""}`, {
        replace: true,
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

  const clear = () => {
    setQuery("");
    inputRef.current?.focus();
  };

  return (
    <div className="w-full max-w-xl">
      <TextInput
        ref={inputRef}
        value={query}
        onChange={(event) => setQuery(event.currentTarget.value)}
        aria-label="Search resources"
        placeholder="Search…"
        size="md"
        radius="md"
        leftSection={<Search size={17} strokeWidth={2.2} />}
        leftSectionPointerEvents="none"
        rightSectionWidth={query ? 40 : 32}
        rightSection={
          query ? (
            <button
              type="button"
              onClick={clear}
              aria-label="Clear search"
              title="Clear search"
              className="flex h-7 w-7 items-center justify-center rounded-md text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500"
            >
              <X size={15} strokeWidth={2.4} />
            </button>
          ) : (
            <kbd className="hidden min-w-6 rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-center font-mono text-[0.68rem] font-bold text-slate-400 sm:inline-block">
              /
            </kbd>
          )
        }
        styles={{
          root: { width: "100%" },
          input: {
            height: "40px",
            minHeight: "40px",
            paddingLeft: "42px",
            fontSize: "0.84rem",
            fontWeight: 600,
            backgroundColor: "rgba(255, 255, 255, 0.48)",
            backdropFilter: "blur(6px)",
            border: "1px solid rgba(203, 213, 225, 0.8)",
            color: "#1e293b",
            boxShadow: "0 1px 2px rgba(15, 23, 42, 0.04)",
            transition:
              "background-color 500ms, border-color 500ms, box-shadow 500ms",
            "&:hover": {
              backgroundColor: "rgba(255, 255, 255, 0.7)",
              borderColor: "#cbd5e1",
            },
            "&:focus": {
              backgroundColor: "#ffffff",
              borderColor: "#0f172a",
              boxShadow: "0 0 0 2px rgba(15, 23, 42, 0.12)",
            },
          },
          section: { color: "#64748b" },
        }}
      />
    </div>
  );
};

export default SearchList;
