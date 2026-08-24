import ClipLoader from "react-spinners/ClipLoader";
import { Loader } from "@mantine/core";
import { motion } from "framer-motion";

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
      <Loader size="sm" color="dark" type="oval" />
    </motion.div>
    <span className="text-[0.7rem] font-semibold tracking-wide text-slate-400">
      Loading {props.label}…
    </span>
  </motion.div>
);
