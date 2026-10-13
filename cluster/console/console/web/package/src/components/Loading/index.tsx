import { Loader } from "@mantine/core";
import { motion } from "framer-motion";
import { twMerge } from "@/lib/utils";

export const ListLoading = (props: { label: string }) => (
  <motion.div
    role="status"
    aria-live="polite"
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    transition={{ duration: 0.5 }}
    className="flex min-h-72 w-full flex-col items-center justify-center gap-3"
  >
    <div className="motion-safe:animate-pulse flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm">
      <Loader size="sm" color="ink" type="oval" />
    </div>
    <span className="text-xs font-semibold tracking-wide text-slate-500">
      Loading {props.label}…
    </span>
  </motion.div>
);

export const PageLoading = (props: { className?: string }) => (
  <motion.div
    role="status"
    aria-live="polite"
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    transition={{ duration: 0.3 }}
    className={twMerge(
      "flex min-h-[40vh] w-full flex-col items-center justify-center gap-3",
      props.className,
    )}
  >
    <div className="motion-safe:animate-pulse flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm">
      <Loader size="sm" color="ink" type="oval" />
    </div>
  </motion.div>
);
