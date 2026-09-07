import React from "react";
import Meta from "../Meta";

export interface QryInterface {
  isSuccess?: boolean;
  isPending?: boolean;
  isError?: boolean;
}

const DefaultSkeleton = () => (
  <div
    className="w-full animate-pulse space-y-3"
    role="status"
    aria-label="Loading"
  >
    <div className="h-9 w-64 rounded-lg bg-slate-200" />
    <div className="h-40 w-full rounded-xl bg-slate-200/70" />
  </div>
);

const PageWrap = (props: {
  qry: QryInterface;
  title?: string;
  skeleton?: React.ReactNode;
  children?: React.ReactNode;
}) => {
  const { qry, children, title } = props;
  return (
    <div className="w-full">
      {title && title.length > 0 && <Meta title={title} />}
      {qry.isPending && (props.skeleton ?? <DefaultSkeleton />)}
      {qry.isSuccess && <div>{children}</div>}
    </div>
  );
};

export default PageWrap;
