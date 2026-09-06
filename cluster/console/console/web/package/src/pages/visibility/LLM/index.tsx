import LLMVisibility from "@/components/LLMVisibility";
import { motion } from "framer-motion";
import { BrainCircuit } from "lucide-react";

const LLMVisibilityPage = () => (
  <div className="flex w-full flex-col gap-4 py-4">
    <motion.header
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.24, ease: "easeOut" }}
      className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_2px_10px_rgba(15,23,42,0.04)] sm:flex-row sm:items-center sm:justify-between"
    >
      <div className="flex min-w-0 items-center gap-3">
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
          <BrainCircuit size={18} strokeWidth={2.2} />
        </span>
        <div className="min-w-0">
          <h1 className="truncate text-base font-bold text-slate-900">
            LLM visibility
          </h1>
          <p className="mt-0.5 text-[0.72rem] font-semibold text-slate-500">
            Every inference request the Cluster served, with its tokens, models,
            tools, guardrails and cost controls.
          </p>
        </div>
      </div>
      <span className="inline-flex w-fit shrink-0 items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-[0.64rem] font-bold uppercase tracking-[0.06em] text-slate-500">
        Cluster scope
      </span>
    </motion.header>

    <LLMVisibility
      principalKinds={["user", "session", "service", "policy"]}
    />
  </div>
);

export default LLMVisibilityPage;
