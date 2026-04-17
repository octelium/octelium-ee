import { TextInput, Transition } from "@mantine/core";
import { useClickOutside } from "@mantine/hooks";
import { Search, X } from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const SearchList = (props: { btnSize?: "xs" | "small" }) => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [qry, setQry] = React.useState(searchParams.get("common.query") ?? "");
  const loc = useLocation();
  const [enabled, setEnabled] = React.useState(false);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const ref = useClickOutside(() => setEnabled(false));

  const handleToggle = () => {
    setEnabled((v) => !v);
    if (!enabled) {
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  };

  const handleChange = (v: React.ChangeEvent<HTMLInputElement>) => {
    setQry(v.target.value);
    const next = new URLSearchParams(searchParams.toString());
    if (v.target.value.length === 0) {
      next.delete("common.query");
    } else {
      next.set("common.query", v.target.value);
    }
    navigate(`${loc.pathname}?${next.toString()}`);
  };

  const handleClear = () => {
    setQry("");
    const next = new URLSearchParams(searchParams.toString());
    next.delete("common.query");
    navigate(`${loc.pathname}?${next.toString()}`);
    inputRef.current?.focus();
  };

  return (
    <div className="flex items-center gap-2 my-4" ref={ref}>
      <button
        onClick={handleToggle}
        className={twMerge(
          "flex items-center justify-center w-7 h-7 rounded-md cursor-pointer",
          "border transition-colors duration-150",
          enabled
            ? "border-slate-300 bg-slate-900 text-white hover:bg-slate-800"
            : "border-slate-200 bg-white text-slate-500 hover:text-slate-900 hover:border-slate-300 shadow-[0_1px_3px_rgba(15,23,42,0.06)]",
        )}
        title={enabled ? "Close search" : "Search"}
      >
        <Search size={13} strokeWidth={2.5} />
      </button>

      <Transition
        mounted={enabled}
        transition="fade-right"
        duration={200}
        timingFunction="ease"
      >
        {(style) => (
          <div style={style}>
            <TextInput
              ref={inputRef}
              value={qry}
              onChange={handleChange}
              placeholder="Search…"
              size="xs"
              rightSection={
                qry.length > 0 ? (
                  <button
                    onClick={handleClear}
                    className="flex items-center justify-center text-slate-400 hover:text-slate-800 cursor-pointer transition-colors duration-150"
                  >
                    <X size={12} strokeWidth={2.5} />
                  </button>
                ) : null
              }
              styles={{
                input: {
                  width: "220px",
                  height: "28px",
                  minHeight: "28px",
                  fontSize: "0.78rem",
                  fontWeight: 600,
                  fontFamily: "Ubuntu, sans-serif",
                  backgroundColor: "#ffffff",
                  border: "1px solid #e2e8f0",
                  borderRadius: "6px",
                  color: "#1e293b",
                  boxShadow: "0 1px 3px rgba(15,23,42,0.06)",
                  "&:focus": {
                    borderColor: "#94a3b8",
                    boxShadow: "0 0 0 2px rgba(148,163,184,0.2)",
                  },
                  "::placeholder": {
                    color: "#94a3b8",
                    fontWeight: 600,
                  },
                },
              }}
            />
          </div>
        )}
      </Transition>
    </div>
  );
};

export default SearchList;
