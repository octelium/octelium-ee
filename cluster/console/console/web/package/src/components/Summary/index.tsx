import { formatNumber } from "@/utils";
import { Grid, Group } from "@mantine/core";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

export const SummaryItemCount = (props: {
  children?: React.ReactNode;
  count?: number;
  to?: string;
  active?: boolean;
}) => {
  const { count, children } = props;
  if (!count || count < 1) {
    return <></>;
  }

  return (
    <div
      className={twMerge(
        "flex flex-col items-start justify-start px-2 font-bold m-4 border-l-[4px]",
        props.active ? `border-1-black` : `border-l-slate-700`
      )}
    >
      <span
        className={twMerge(
          "text-2x mr-1",
          props.active ? `text-black` : `text-slate-700`
        )}
        style={{
          textShadow: "0 2px 8px rgba(0, 0, 0, 0.2)",
        }}
      >
        {formatNumber(count)}
      </span>
      {props.to ? (
        props.active ? (
          <div
            className={twMerge(
              "text-sm duration-500 transition-all",
              props.active
                ? `text-black`
                : `text-slate-500 hover:text-slate-800`
            )}
          >
            {children}
          </div>
        ) : (
          <Link to={props.to}>
            <div
              className={twMerge(
                "text-sm duration-500 transition-all",
                props.active
                  ? `text-black`
                  : `text-slate-500 hover:text-slate-800`
              )}
            >
              {children}
            </div>
          </Link>
        )
      ) : (
        <div
          className={twMerge(
            "text-sm",
            props.active ? `text-black` : `text-slate-500`
          )}
        >
          {children}
        </div>
      )}
    </div>
  );
};

export const SummaryItemCountWrap = (props: { children?: React.ReactNode }) => {
  return <Grid>{props.children}</Grid>;
};

export const SummaryNoItems = (props: { children?: React.ReactNode }) => {
  return (
    <div className="w-full flex items-center justify-center h-[200px]">
      <div className="font-bold text-2xl text-gray-600 text-shadow-2xs">
        No Items Found
      </div>
    </div>
  );
};
