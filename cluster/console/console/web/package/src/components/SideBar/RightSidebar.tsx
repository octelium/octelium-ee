import { getDomain } from "@/utils";
import { getClientAuth } from "@/utils/client";
import { Button, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { LockKeyhole, LogOut } from "lucide-react";

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
      <div className="flex flex-col mt-32">
        <Button
          className="mb-3"
          fullWidth
          variant="outline"
          component="a"
          href={`https://${getDomain()}/authenticators`}
        >
          <LockKeyhole className="mr-1" />
          <span>Authenticators</span>
        </Button>

        <Button fullWidth variant="outline" onClick={open}>
          <LogOut className="mr-1" />
          <span>Logout</span>
        </Button>
      </div>

      <Modal opened={opened} onClose={close} centered>
        <div className="font-bold text-xl mb-4">
          {`Are you sure that you want to logout?`}
        </div>

        <div className="mt-4 flex justify-end items-center">
          <Button variant="outline" onClick={close}>
            Cancel
          </Button>
          <Button
            className="ml-4"
            loading={mutationLogout.isPending}
            onClick={() => {
              mutationLogout.mutate();
            }}
            autoFocus
          >
            Yes, Logout
          </Button>
        </div>
      </Modal>
    </div>
  );
};
