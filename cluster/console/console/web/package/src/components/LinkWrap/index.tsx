import * as React from "react";
import { Link } from "react-router-dom";

const LinkWrap = (props: { to: string; children?: React.ReactNode }) => {
  return (
    <Link
      className="text-slate-600 hover:text-slate-800 rounded-full transition-all duration-200 shadow-2xl"
      to={props.to}
    >
      {props.children}
    </Link>
  );
};

export default LinkWrap;
