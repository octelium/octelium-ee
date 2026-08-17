import * as AccessPB from "@/apis/accessv1/accessv1";
import { onError } from "@/utils";
import { getClientAccess } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Button, Modal, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { Ban, Loader2, ShieldAlert, X } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

const REVOCABLE_STATES = new Set([
  AccessPB.Request_Status_State_Status.STATUS_UNKNOWN,
  AccessPB.Request_Status_State_Status.PENDING,
  AccessPB.Request_Status_State_Status.APPROVED,
]);

const RevokeRequest = (props: { item: AccessPB.Request }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [isConfirmed, setIsConfirmed] = React.useState(false);
  const state = props.item.status?.state?.status;

  const mutation = useMutation({
    mutationFn: () =>
      getClientAccess().revokeRequest(
        AccessPB.RevokeRequestRequest.create({
          requestRef: getResourceRef(props.item),
        }),
      ).response,
    onSuccess: () => {
      setIsConfirmed(false);
      close();
      invalidateResource(props.item);
      invalidateResourceList(props.item);
      toast.success(`Request ${props.item.metadata?.name} revoked`);
    },
    onError,
  });

  React.useEffect(() => {
    if (!opened) {
      setIsConfirmed(false);
      mutation.reset();
    }
  }, [opened]);

  if (state !== undefined && !REVOCABLE_STATES.has(state)) return null;

  const handleClose = () => {
    if (mutation.isPending) return;
    close();
  };

  return (
    <>
      <Button
        type="button"
        size="compact-xs"
        variant="outline"
        color="red.7"
        leftSection={<Ban size={12} strokeWidth={2.25} />}
        onClick={open}
      >
        Revoke
      </Button>

      <Modal
        opened={opened}
        onClose={handleClose}
        centered
        size="md"
        withCloseButton={false}
        padding={0}
        closeOnClickOutside={!mutation.isPending}
        closeOnEscape={!mutation.isPending}
        overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
        transitionProps={{ transition: "pop", duration: 250 }}
        styles={{
          content: {
            border: "1px solid #e2e8f0",
            borderRadius: "14px",
            boxShadow: "0 24px 64px rgba(15,23,42,0.18)",
            overflow: "hidden",
          },
        }}
      >
        <header className="flex items-center justify-between gap-3 border-b border-slate-200 bg-slate-50/70 px-4 py-3.5 sm:px-5">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-red-600 text-white shadow-sm">
              <Ban size={16} strokeWidth={2.25} />
            </span>
            <div className="min-w-0">
              <h2 className="truncate text-sm font-bold text-slate-900">
                Revoke request
              </h2>
              <p className="mt-0.5 truncate text-[0.65rem] font-semibold text-slate-400">
                {props.item.metadata?.name}
              </p>
            </div>
          </div>
          <Button
            type="button"
            variant="subtle"
            color="gray"
            size="compact-xs"
            disabled={mutation.isPending}
            leftSection={<X size={12} strokeWidth={2.5} />}
            onClick={handleClose}
          >
            Close
          </Button>
        </header>

        <div className="space-y-4 px-4 py-4 sm:px-5">
          <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50/70 px-3.5 py-3">
            <ShieldAlert
              size={16}
              className="mt-0.5 shrink-0 text-red-600"
              strokeWidth={2.25}
            />
            <div>
              <p className="text-[0.76rem] font-bold text-red-800">
                Access granted by this request will end
              </p>
              <p className="mt-1 text-[0.69rem] font-semibold leading-relaxed text-red-700/80">
                A pending request will be withdrawn. An approved request will
                lose its active approval.
              </p>
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <Switch
              autoFocus
              checked={isConfirmed}
              disabled={mutation.isPending}
              color="red.8"
              size="sm"
              label="Confirm request revocation"
              onChange={(event) => {
                setIsConfirmed(event.currentTarget.checked);
                if (mutation.isError) mutation.reset();
              }}
              styles={{
                label: {
                  color: "#334155",
                  fontSize: "0.75rem",
                  fontWeight: 700,
                },
              }}
            />
          </div>

          {mutation.isError && (
            <div className="rounded-xl border border-red-200 bg-red-50 px-3.5 py-3 text-[0.7rem] font-semibold text-red-700">
              {mutation.error instanceof Error
                ? mutation.error.message
                : "The request could not be revoked."}
            </div>
          )}

          <div className="flex justify-end gap-2 border-t border-slate-100 pt-4">
            <Button
              type="button"
              variant="default"
              size="xs"
              disabled={mutation.isPending}
              onClick={handleClose}
            >
              Cancel
            </Button>
            <Button
              type="button"
              color="red.8"
              size="xs"
              disabled={!isConfirmed || mutation.isPending}
              leftSection={
                mutation.isPending ? (
                  <Loader2 className="animate-spin" size={13} />
                ) : (
                  <Ban size={13} strokeWidth={2.25} />
                )
              }
              onClick={() => mutation.mutate()}
            >
              {mutation.isPending ? "Revoking…" : "Revoke request"}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};

export default RevokeRequest;
