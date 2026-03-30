import * as React from "react";

import { ActionIcon, TextInput, Transition } from "@mantine/core";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

import { useClickOutside } from "@mantine/hooks";
import { Search, SearchX } from "lucide-react";

const SearchList = (props: { btnSize?: "xs" | "small" }) => {
  const navigate = useNavigate();
  let [searchParams, _] = useSearchParams();

  let [qry, setQry] = React.useState(searchParams.get("common.query") ?? "");
  const loc = useLocation();

  let [enabled, setEnabled] = React.useState(false);
  const ref = useClickOutside(() => setEnabled(false));

  return (
    <div className="flex items-center h-[30px] my-4" ref={ref}>
      <ActionIcon
        onClick={() => {
          setEnabled(!enabled);
        }}
        variant="transparent"
        aria-label="Settings"
      >
        {enabled ? <SearchX size={"28px"} /> : <Search size={"28px"} />}
      </ActionIcon>

      <Transition
        mounted={enabled}
        transition="fade-right"
        duration={400}
        timingFunction="ease"
      >
        {(style) => (
          <div style={style} className="ml-4 w-full">
            <TextInput
              className="shadow-sm w-[300px]"
              value={qry}
              onChange={(v) => {
                setQry(v.target.value);

                if (v.target.value.length === 0) {
                  searchParams.delete("common.query");
                } else {
                  searchParams.set("common.query", v.target.value);
                }

                navigate(`${loc.pathname}?${searchParams.toString()}`);
              }}
            />
          </div>
        )}
      </Transition>
    </div>
  );
};

export default SearchList;
