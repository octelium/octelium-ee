import { getDomain } from "@/utils";
import { getClientAuth } from "@/utils/client";
import { Button, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { Loader2, LockKeyhole, LogOut, X } from "lucide-react";

export default () => {
  const [opened, { open, close }] = useDisclosure(false);

  const mutationLogout = useMutation({
    mutationFn: async () => {
      await getClientAuth().logout({});
    },
    onSuccess: () => {
      window.location.reload();
    },
  });

  return (
    <div className="flex flex-col h-full w-full">
      <div className="flex flex-col gap-2 mt-12">
        <Button
          variant="outline"
          component="a"
          href={`https://${getDomain()}/authenticators`}
          leftSection={
            <LockKeyhole size={14} strokeWidth={2.5} className="shrink-0" />
          }
        >
          Authenticators
        </Button>

        <Button
          onClick={open}
          variant={`outline`}
          leftSection={
            <LogOut size={14} strokeWidth={2.5} className="shrink-0" />
          }
        >
          Log out
        </Button>
      </div>

      <Modal
        opened={opened}
        onClose={close}
        centered
        withCloseButton={false}
        padding={0}
        size="sm"
        styles={{
          content: {
            borderRadius: "12px",
            border: "1px solid #e2e8f0",
            overflow: "hidden",
          },
        }}
      >
        <div className="flex flex-col">
          <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 bg-slate-50/60">
            <div className="flex items-center gap-2">
              <LogOut
                size={14}
                className="text-red-500 shrink-0"
                strokeWidth={2.5}
              />
              <span className="text-[0.82rem] font-bold text-slate-800">
                Log out
              </span>
            </div>
            <button
              onClick={close}
              className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-200 transition-colors duration-150 cursor-pointer"
            >
              <X size={13} strokeWidth={2.5} />
            </button>
          </div>

          <div className="px-5 py-4">
            <p className="text-[0.82rem] font-semibold text-slate-600">
              Are you sure you want to log out of this session?
            </p>
          </div>

          <div className="flex items-center justify-end gap-2 px-5 py-3.5 border-t border-slate-200 bg-slate-50/60">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              onClick={close}
            >
              Cancel
            </Button>
            <Button
              variant="filled"
              color="red"
              size="sm"
              leftSection={
                mutationLogout.isPending ? (
                  <Loader2
                    size={13}
                    className="animate-spin"
                    strokeWidth={2.5}
                  />
                ) : (
                  <LogOut size={13} strokeWidth={2.5} />
                )
              }
              loading={mutationLogout.isPending}
              onClick={() => mutationLogout.mutate()}
            >
              {mutationLogout.isPending ? "Logging out…" : "Log out"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
