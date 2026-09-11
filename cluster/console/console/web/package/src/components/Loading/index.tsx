import ClipLoader from "react-spinners/ClipLoader";
import { Loader } from "@mantine/core";
import { motion } from "framer-motion";
import { twMerge } from "tailwind-merge";

const Loading = () => {
  return (
    <div>
      <div>
        <ClipLoader
          color={"#111"}
          loading={true}
          size={150}
          aria-label="Loading Spinner"
          data-testid="loader"
        />
      </div>
    </div>
  );
};

export default Loading;

export const ListLoading = (props: { label: string }) => (
  <motion.div
    role="status"
    aria-live="polite"
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    transition={{ duration: 0.5 }}
    className="flex min-h-72 w-full flex-col items-center justify-center gap-3"
  >
    <motion.div
      animate={{ opacity: [0.45, 1, 0.45], scale: [0.96, 1, 0.96] }}
      transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
      className="flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm"
    >
      <Loader size="sm" color="ink" type="oval" />
    </motion.div>
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
    <motion.div
      animate={{ opacity: [0.45, 1, 0.45], scale: [0.96, 1, 0.96] }}
      transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
      className="flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm"
    >
      <Loader size="sm" color="ink" type="oval" />
    </motion.div>
  </motion.div>
);
