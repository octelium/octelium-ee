import { Condition } from "@/apis/corev1/corev1";
import { Condition as Policy_Spec_Condition } from "@/apis/enterprisev1/enterprisev1";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import { Button, Modal, ScrollArea } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import ConditionC from "./Condition";
import PrintCond from "./PrintCondition";
import { SlidersHorizontal } from "lucide-react";

const ConditionBuilderBtn = (props: {
  onChange: (condition?: Condition) => void;
}) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [cur, setCur] = useState<Policy_Spec_Condition | undefined>(undefined);

  const mutation = useMutation({
    mutationFn: async (req?: Policy_Spec_Condition) => {
      const r = await getClientEnterprise().getCoreCondition(
        req ?? Policy_Spec_Condition.create(),
      );
      return r.response;
    },
    onSuccess: (r) => props.onChange(r),
    onError,
  });

  return (
    <>
      <button
        onClick={open}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.72rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
      >
        <SlidersHorizontal size={12} strokeWidth={2.5} />
        Condition builder
      </button>

      <Modal
        opened={opened}
        onClose={close}
        size="100vw"
        padding={0}
        withCloseButton={false}
        styles={{
          body: { padding: 0 },
          content: {
            borderRadius: "12px",
            overflow: "hidden",
            border: "1px solid #e2e8f0",
          },
        }}
      >
        <div className="flex flex-col h-[90vh]">
          <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 bg-white shrink-0">
            <div className="flex items-center gap-2">
              <SlidersHorizontal size={16} className="text-slate-500" />
              <span className="text-[0.85rem] font-bold uppercase tracking-[0.05em] text-slate-800">
                Condition Builder
              </span>
            </div>
            <button
              onClick={close}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[0.72rem] font-bold text-slate-500 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer"
            >
              Done
            </button>
          </div>

          <div className="flex flex-1 overflow-hidden">
            <div className="flex-1 overflow-auto p-6">
              <ScrollArea h="100%">
                <ConditionC
                  onChange={(v) => {
                    setCur(v);
                    mutation.mutate(v);
                  }}
                />
              </ScrollArea>
            </div>

            {cur && (
              <div className="w-80 shrink-0 border-l border-slate-200 bg-slate-50/50 overflow-auto p-5">
                <p className="text-[0.68rem] font-bold uppercase tracking-[0.08em] text-slate-400 mb-3">
                  Preview
                </p>
                <PrintCond item={cur} />
              </div>
            )}
          </div>
        </div>
      </Modal>
    </>
  );
};

export default ConditionBuilderBtn;