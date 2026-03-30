import * as React from "react";

import { MdDelete } from "react-icons/md";

import {
  getResourceClient,
  getResourceListPathFromResource,
  invalidateResourceList,
  Resource,
} from "@/utils/pb";
import { Button, DefaultMantineColor, MantineSize } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";

import { onError } from "@/utils";
import { Modal, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { twMerge } from "tailwind-merge";
import InfoItem from "../InfoItem";
import ResourceYAML from "../ResourceYAML";

const DeleteResource = (props: {
  btnSize?:
    | MantineSize
    | "compact-lg"
    | "compact-xs"
    | "compact-sm"
    | "compact-md"
    | "compact-xl";
  btnVariant?: `filled` | `default` | `outline`;
  btnColor?: DefaultMantineColor;
  item: Resource;
  onSuccess?: () => void;
  doNotNavigateAfter?: boolean;
}) => {
  const { item } = props;
  const [openDelete, setOpenDelete] = React.useState(false);
  const [isDeletable, setIsDeletable] = React.useState(false);

  const [deleteField, setDeleteField] = React.useState("");
  const navigate = useNavigate();
  const [opened, { open, close }] = useDisclosure(false);

  const mutationDelete = useMutation({
    mutationFn: async () => {
      // @ts-ignore
      await getResourceClient(item)[`delete${item.kind as ResourceCoreName}`]({
        uid: item?.metadata?.uid,
      });
    },
    onSuccess: () => {
      setIsDeletable(false);
      close();
      invalidateResourceList(item);
      if (!props.doNotNavigateAfter) {
        navigate(getResourceListPathFromResource(item));
      }

      toast.success(`${item.kind}: ${item.metadata!.name} deleted`);
    },
    onError: onError,
  });

  if (item.metadata?.isSystem) {
    return <></>;
  }

  return (
    <>
      <Button
        color={props.btnColor ? props.btnColor : "red.8"}
        onClick={open}
        size={props.btnSize}
        variant={props.btnVariant}
      >
        <MdDelete />
      </Button>

      <Modal
        opened={opened}
        onClose={() => {
          setIsDeletable(false);
          close();
        }}
        centered
      >
        <div className="font-bold text-xl mb-4">
          {`Are you sure that you want to delete this ${props.item.kind}?`}
        </div>

        <div className="w-full my-4">
          <InfoItem title="Name">{props.item.metadata!.name}</InfoItem>
          <InfoItem title="UID">{props.item.metadata!.uid}</InfoItem>
          <InfoItem title="Detailed Info">
            <ResourceYAML item={item} size="xs" />
          </InfoItem>
        </div>
        <div>
          <Switch
            checked={isDeletable}
            onChange={(event) => setIsDeletable(event.currentTarget.checked)}
            color="red"
            size="lg"
            label={`Yes, Delete this ${props.item.kind}`}
          />
        </div>
        <div className="mt-4 flex justify-end items-center">
          <Button
            variant="outline"
            onClick={() => {
              setIsDeletable(false);
              close();
            }}
          >
            Cancel
          </Button>
          <Button
            className={twMerge(
              "ml-4  transition-all duration-500",
              isDeletable ? `shadow-lg` : undefined
            )}
            disabled={!isDeletable}
            loading={mutationDelete.isPending}
            color="red"
            onClick={() => {
              mutationDelete.mutate();
            }}
            autoFocus
          >
            Delete this {props.item.kind}
          </Button>
        </div>
      </Modal>
    </>
  );
};

export default DeleteResource;
