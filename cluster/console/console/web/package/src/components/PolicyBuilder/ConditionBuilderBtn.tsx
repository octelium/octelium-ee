import { Condition } from "@/apis/corev1/corev1";
import { Condition as BuilderCondition } from "@/apis/enterprisev1/enterprisev1";
import { onError } from "@/utils";
import { getClientEnterprise } from "@/utils/client";
import { Button, Modal, SegmentedControl } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import {
  Check,
  Eye,
  Loader2,
  SlidersHorizontal,
  Sparkles,
  X,
} from "lucide-react";
import { useState } from "react";
import ConditionEditor from "./Condition";
import PrintCondition from "./PrintCondition";

const createDraft = () =>
  BuilderCondition.create({
    type: { oneofKind: "matchAny", matchAny: true },
  });

const ConditionBuilderBtn = (props: {
  onChange: (condition?: Condition) => void;
}) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [draft, setDraft] = useState<BuilderCondition>(createDraft);
  const [mobileView, setMobileView] = useState<"build" | "preview">("build");
  const [changed, setChanged] = useState(true);

  const mutation = useMutation({
    mutationFn: async (request: BuilderCondition) => {
      const result = await getClientEnterprise().getCoreCondition(request);
      return result.response;
    },
    onSuccess: (condition) => {
      props.onChange(condition);
      setChanged(false);
      close();
    },
    onError,
  });

  const closeBuilder = () => {
    if (mutation.isPending) return;
    mutation.reset();
    close();
  };

  return (
    <>
      <Button
        type="button"
        variant="default"
        leftSection={<SlidersHorizontal size={13} strokeWidth={2.5} />}
        onClick={() => {
          mutation.reset();
          open();
        }}
      >
        Visual condition builder
      </Button>

      <Modal
        opened={opened}
        onClose={closeBuilder}
        size="min(1240px, calc(100vw - 24px))"
        padding={0}
        withCloseButton={false}
        closeOnClickOutside={!mutation.isPending}
        closeOnEscape={!mutation.isPending}
        centered
        overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
        transitionProps={{ transition: "pop", duration: 300 }}
        styles={{
          body: { padding: 0 },
          content: {
            border: "1px solid #e2e8f0",
            borderRadius: "14px",
            boxShadow: "0 24px 64px rgba(15,23,42,0.18)",
            overflow: "hidden",
          },
        }}
      >
        <div className="flex h-[min(900px,calc(100dvh-32px))] min-h-[560px] flex-col bg-slate-50">
          <header className="flex shrink-0 items-center justify-between gap-4 border-b border-slate-200 bg-white px-4 py-3.5 sm:px-5">
            <div className="flex min-w-0 items-center gap-3">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white shadow-sm">
                <SlidersHorizontal size={16} strokeWidth={2.25} />
              </span>
              <div className="min-w-0">
                <h2 className="truncate text-sm font-bold tracking-tight text-slate-900">
                  Visual condition builder
                </h2>
                <p className="mt-0.5 hidden text-[0.68rem] font-semibold text-slate-400 sm:block">
                  Build nested policy logic without writing CEL or Rego.
                </p>
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <Button
                type="button"
                variant="default"
                size="compact-sm"
                leftSection={<X size={12} strokeWidth={2.5} />}
                disabled={mutation.isPending}
                onClick={closeBuilder}
              >
                Cancel
              </Button>
              <Button
                type="button"
                color="dark"
                size="compact-sm"
                leftSection={
                  mutation.isPending ? (
                    <Loader2 size={12} className="animate-spin" />
                  ) : (
                    <Check size={12} strokeWidth={2.5} />
                  )
                }
                loading={mutation.isPending}
                disabled={!changed || mutation.isPending}
                onClick={() => mutation.mutate(BuilderCondition.clone(draft))}
              >
                {mutation.isPending ? "Applying…" : "Apply"}
              </Button>
            </div>
          </header>

          <div className="shrink-0 border-b border-slate-200 bg-white p-3 lg:hidden">
            <SegmentedControl
              fullWidth
              value={mobileView}
              data={[
                {
                  value: "build",
                  label: (
                    <span className="flex items-center gap-1.5">
                      <SlidersHorizontal size={12} /> Build
                    </span>
                  ),
                },
                {
                  value: "preview",
                  label: (
                    <span className="flex items-center gap-1.5">
                      <Eye size={12} /> Preview
                    </span>
                  ),
                },
              ]}
              onChange={(value) =>
                setMobileView(value as "build" | "preview")
              }
            />
          </div>

          <div className="grid min-h-0 flex-1 lg:grid-cols-[minmax(0,1fr)_360px]">
            <main
              className={`${
                mobileView === "build" ? "block" : "hidden"
              } min-h-0 overflow-y-auto p-3 sm:p-5 lg:block`}
            >
              <div className="mx-auto max-w-4xl">
                <div className="mb-4 flex items-start gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm">
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600">
                    <Sparkles size={13} strokeWidth={2.25} />
                  </span>
                  <div>
                    <p className="text-[0.74rem] font-bold text-slate-700">
                      Define the authorization logic
                    </p>
                    <p className="mt-0.5 text-[0.67rem] font-semibold leading-relaxed text-slate-400">
                      Choose a condition type, configure expressions, and nest
                      AND, OR, or NOR groups where needed.
                    </p>
                  </div>
                </div>

                <ConditionEditor
                  item={draft}
                  onChange={(condition) => {
                    if (!condition) return;
                    setDraft(BuilderCondition.clone(condition));
                    setChanged(true);
                  }}
                />
              </div>
            </main>

            <aside
              className={`${
                mobileView === "preview" ? "block" : "hidden"
              } min-h-0 overflow-y-auto border-l border-slate-200 bg-white p-4 lg:block lg:p-5`}
            >
              <div className="sticky top-0">
                <div className="mb-4 flex items-center justify-between gap-3">
                  <div>
                    <p className="text-[0.7rem] font-bold uppercase tracking-[0.07em] text-slate-600">
                      Logic preview
                    </p>
                    <p className="mt-0.5 text-[0.64rem] font-semibold text-slate-400">
                      Live representation of this condition
                    </p>
                  </div>
                  <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                    <Eye size={14} strokeWidth={2.25} />
                  </span>
                </div>
                <div className="rounded-xl border border-slate-200 bg-slate-50/60 p-3.5">
                  <PrintCondition item={draft} />
                </div>
              </div>
            </aside>
          </div>

          <footer className="flex shrink-0 items-center border-t border-slate-200 bg-white px-4 py-3 sm:px-5">
            <div className="flex items-center gap-2">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  changed ? "bg-amber-500" : "bg-slate-300"
                }`}
              />
              <span
                className={`text-[0.68rem] font-semibold ${
                  changed ? "text-amber-700" : "text-slate-400"
                }`}
              >
                {changed ? "Draft has unapplied changes" : "No changes yet"}
              </span>
            </div>
          </footer>
        </div>
      </Modal>
    </>
  );
};

export default ConditionBuilderBtn;
