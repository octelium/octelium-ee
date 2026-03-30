import React from "react";
import Meta from "../Meta";

export interface QryInterface {
  isSuccess?: boolean;
  isPending?: boolean;
  isError?: boolean;
}

const PageWrap = (props: {
  qry: QryInterface;
  title?: string;
  children?: React.ReactNode;
}) => {
  const { qry, children, title } = props;
  return (
    <div className="w-full">
      {title && title.length > 0 && <Meta title={title} />}
      {qry.isPending && (
        <div className="w-full font-extrabold text-4xl flex items-center"></div>
      )}
      {qry.isSuccess && <div>{children}</div>}
    </div>
  );
};

export default PageWrap;
