import { getDomain, isDev } from "@/utils";
import { getClientAuth } from "@/utils/client";
import { Button, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { ArrowUpRight, LockKeyhole, LogOut } from "lucide-react";
import { toast } from "sonner";

export default () => {
  const [opened, { open, close }] = useDisclosure(false);

  const mutationLogout = useMutation({
    mutationFn: async () => {
      await getClientAuth().logout({});
    },
    onSuccess: () => {
      window.location.reload();
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Could not log out");
    },
  });

  const authenticatorsHref = isDev()
    ? `${window.location.origin}/authenticators`
    : `https://${getDomain()}/authenticators`;

  return (
    <div className="flex flex-col w-full">
      <div className="flex flex-col gap-1">
        <a
          href={authenticatorsHref}
          className="flex min-h-9 items-center gap-2.5 rounded-lg px-2.5 text-[0.8125rem] font-medium text-slate-600 outline-none transition-colors duration-150 hover:bg-slate-200/50 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:min-h-11"
        >
          <LockKeyhole
            size={16}
            strokeWidth={1.8}
            className="shrink-0"
            aria-hidden
          />
          <span className="flex-1">Authenticators</span>
          <ArrowUpRight size={13} className="text-slate-500" aria-hidden />
        </a>

        <button
          type="button"
          onClick={open}
          className="flex min-h-9 items-center gap-2.5 rounded-lg px-2.5 text-[0.8125rem] font-medium text-slate-600 outline-none transition-colors duration-150 hover:bg-red-50 hover:text-red-700 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:min-h-11"
        >
          <LogOut size={16} strokeWidth={1.8} className="shrink-0" aria-hidden />
          Sign out
        </button>
      </div>

      <Modal
        opened={opened}
        onClose={close}
        centered
        title="Sign out"
        padding="lg"
        size="sm"
        zIndex={240}
        closeButtonProps={{ "aria-label": "Close sign out dialog" }}
        styles={{
          header: { backgroundColor: "var(--color-white)" },
          title: {
            fontSize: "0.9375rem",
            fontWeight: 600,
            color: "var(--color-slate-900)",
          },
          content: {
            backgroundColor: "var(--color-white)",
            borderRadius: "12px",
            border: "1px solid var(--color-slate-200)",
            overflow: "hidden",
          },
        }}
      >
        <div className="flex flex-col gap-5">
          <p className="text-[0.8125rem] leading-relaxed text-slate-600">
            Are you sure you want to sign out of this session?
          </p>
          <div className="flex items-center justify-end gap-2">
            <Button variant="default" size="sm" onClick={close}>
              Cancel
            </Button>
            <Button
              variant="filled"
              color="red"
              size="sm"
              leftSection={<LogOut size={14} strokeWidth={2} />}
              loading={mutationLogout.isPending}
              onClick={() => mutationLogout.mutate()}
            >
              {mutationLogout.isPending ? "Signing out…" : "Sign out"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
