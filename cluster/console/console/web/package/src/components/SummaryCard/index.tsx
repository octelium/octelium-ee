import { motion } from "framer-motion";
import { ChevronRight } from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const SummaryCard = (props: {
  title: string;
  link?: string;
  children?: React.ReactNode;
  className?: string;
}) => (
  <motion.div
    initial={{ opacity: 0, y: 8 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.22, ease: "easeOut" }}
    className={twMerge(
      "overflow-hidden rounded-xl border border-slate-200 bg-white",
      props.className,
    )}
  >
    <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
      <span className="text-body font-semibold uppercase tracking-[0.05em] text-slate-800">
        {props.title}
      </span>
      {props.link && (
        <Link
          to={props.link}
          className="flex items-center gap-1 text-xs font-normal text-slate-500 transition-colors duration-150 hover:text-slate-900"
        >
          View all
          <ChevronRight size={11} strokeWidth={2.5} />
        </Link>
      )}
    </div>
    <div className="px-4 py-3.5">{props.children}</div>
  </motion.div>
);

export default SummaryCard;
