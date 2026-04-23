import * as CoreC from "@/apis/corev1/corev1";
import AccessLogViewer from "@/components/AccessLogViewer";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { randomStringLowerCase, slugify } from "@/utils";
import { getClientCore } from "@/utils/client";
import { getResourcePath, getResourceRef } from "@/utils/pb";
import { Button, Modal, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import Edit from "../Credential/Edit";

export const AccessLog = (props: { item: CoreC.User }) => {
  return <AccessLogViewer userRef={getResourceRef(props.item)} />;
};

export const ResourceItemInfo = (props: { item: CoreC.User }) => {
  let { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="Type">
        <Label>
          {match(item.spec!.type)
            .with(CoreC.User_Spec_Type.HUMAN, () => "Human")
            .with(CoreC.User_Spec_Type.WORKLOAD, () => "Workload")
            .otherwise(() => "")}
        </Label>
      </InfoItem>

      {item.spec?.email && <InfoItem title="Email">{item.spec.email}</InfoItem>}

      {item.spec!.groups.length > 0 && (
        <InfoItem title="Groups">
          <div className="flex items-center">
            {item.spec!.groups.map((x) => (
              <Label key={x}>{x}</Label>
            ))}
          </div>
        </InfoItem>
      )}

      {item.spec!.authorization &&
        item.spec!.authorization.policies.length > 0 && (
          <InfoItem title="Policies">
            <div className="flex items-center">
              {item.spec!.authorization.policies.map((x) => (
                <Label key={x}>{x}</Label>
              ))}
            </div>
          </InfoItem>
        )}

      <InfoItem title="Active">
        <div className="w-full flex items-center">
          <span
            className={twMerge(
              item.spec!.isDisabled ? `text-red-500` : undefined,
            )}
          >
            {item.spec!.isDisabled ? `No` : `Yes`}
          </span>
          <Switch
            className="ml-2"
            checked={item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = v.currentTarget.checked;
              mutationUpdate.mutate(item);
            }}
          />
        </div>
      </InfoItem>
    </>
  );
};

export default (props: { item: CoreC.User }) => {
  let { item } = props;

  return (
    <div>
      <div className="w-full mb-8">
        <ResourceItemInfo item={item} />
      </div>
    </div>
  );
};

export const MainInfo = (props: { item: CoreC.User }): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const [opened, { open, close }] = useDisclosure(false);
  const navigate = useNavigate();
  const [credential, setCredential] = useState(
    CoreC.Credential.create({
      kind: "Credential",
      apiVersion: "core/v1",
      metadata: {},
      spec: {
        user: item.metadata?.name,
      },
    }),
  );

  const mutationCredential = useMutation({
    mutationFn: async () => {
      return await getClientCore().createCredential(credential).response;
    },
    onSuccess: (response) => {
      navigate(getResourcePath(response));
    },
    onError: (err: unknown) => {
      if (err instanceof Error) {
        toast.error(err.message);
      }
    },
  });

  return {
    items: [
      {
        label: "Type",
        value: (
          <Label>
            {match(item.spec!.type)
              .with(CoreC.User_Spec_Type.HUMAN, () => "Human")
              .with(CoreC.User_Spec_Type.WORKLOAD, () => "Workload")
              .otherwise(() => "")}
          </Label>
        ),
      },

      ...(item.spec?.email
        ? [
            {
              label: "Email",
              value: (
                <span className="font-mono text-[0.75rem]">
                  {item.spec.email}
                </span>
              ),
            },
          ]
        : []),

      ...(item.spec!.groups.length > 0
        ? [
            {
              label: "Groups",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec!.groups.map((x) => (
                    <Label key={x}>{x}</Label>
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.spec!.authorization &&
      item.spec!.authorization?.policies.length > 0
        ? [
            {
              label: "Policies",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec!.authorization!.policies.map((x) => (
                    <Label key={x}>{x}</Label>
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      {
        label: "Active",
        value: (
          <EditItemWrap
            label="active"
            showComponent={
              <span
                className={twMerge(
                  "text-[0.75rem] font-semibold",
                  item.spec!.isDisabled ? "text-red-500" : "text-emerald-600",
                )}
              >
                {item.spec!.isDisabled ? "Disabled" : "Active"}
              </span>
            }
            editComponent={
              <Switch
                size="sm"
                checked={!item.spec!.isDisabled}
                onChange={(v) => {
                  item.spec!.isDisabled = !v.currentTarget.checked;
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },
      {
        label: "Credential",
        value: (
          <div>
            <Button onClick={open}>Create Credential</Button>
            <Modal opened={opened} onClose={close} size={"xl"} centered>
              <div>
                <Edit
                  onUpdate={(v) => {
                    v.metadata!.name = `${v.spec!.user}-${slugify(CoreC.Credential_Spec_Type[v.spec!.type]).toLowerCase()}-${randomStringLowerCase(4)}`;
                    setCredential(CoreC.Credential.clone(v));
                  }}
                  item={credential}
                />

                <div className="mt-4 flex items-end justify-end">
                  <Button
                    onClick={() => {
                      mutationCredential.mutate();
                    }}
                  >
                    Create
                  </Button>
                </div>
              </div>
            </Modal>
          </div>
        ),
      },
    ],
  };
};
