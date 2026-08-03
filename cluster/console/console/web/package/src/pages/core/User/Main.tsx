import * as CoreC from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import AccessLogViewer from "@/components/AccessLogViewer";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { randomStringLowerCase, slugify } from "@/utils";
import { getClientCore } from "@/utils/client";
import { getResourcePath, getResourceRef } from "@/utils/pb";
import { Button, Modal, Switch } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import {
  KeyRound,
  Plus,
  Shield,
  X,
} from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import Edit from "../Credential/Edit";
import { useUserRelatedResources } from "./related";

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

      {item.spec?.email && (
        <InfoItem title="Email">
          <CopyText value={item.spec.email} />
        </InfoItem>
      )}

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
            checked={!item.spec!.isDisabled}
            onChange={(v) => {
              item.spec!.isDisabled = !v.currentTarget.checked;
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

const UserActiveControl = (props: { item: CoreC.User }) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();

  return (
    <EditItemWrap
      mutation={mutationUpdate}
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
  );
};

const CreateUserCredential = (props: { item: CoreC.User }) => {
  const { item } = props;
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

  return (
    <div className="flex items-center">
      <Button
        variant="filled"
        color="dark"
        size="compact-sm"
        leftSection={<Plus size={12} strokeWidth={2.5} />}
        onClick={open}
      >
        Create credential
      </Button>
      <Modal
        opened={opened}
        onClose={close}
        size="xl"
        centered
        closeOnClickOutside={!mutationCredential.isPending}
        closeOnEscape={!mutationCredential.isPending}
        title={
          <div className="flex items-center gap-3">
            <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-600">
              <KeyRound size={16} strokeWidth={2.25} />
            </span>
            <div className="flex flex-col">
              <span className="text-sm font-bold text-slate-800">
                Create credential
              </span>
              <span className="text-[0.7rem] font-semibold text-slate-500">
                Issue a new credential for {item.metadata?.name}
              </span>
            </div>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "64px" },
          body: { padding: 0 },
          content: { border: "1px solid #e2e8f0" },
        }}
      >
        <div className="flex flex-col bg-slate-50/70">
          <div className="p-4 sm:p-5">
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <Edit
            onUpdate={(v) => {
              v.metadata!.name = `${v.spec!.user}-${slugify(CoreC.Credential_Spec_Type[v.spec!.type]).toLowerCase()}-${randomStringLowerCase(6)}`;
              setCredential(CoreC.Credential.clone(v));
            }}
            item={credential}
          />
            </div>
          </div>

          <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-white px-4 py-3 sm:px-5">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              disabled={mutationCredential.isPending}
              onClick={close}
            >
              Cancel
            </Button>
            <Button
              color="dark"
              size="sm"
              leftSection={<KeyRound size={13} strokeWidth={2.5} />}
              loading={mutationCredential.isPending}
              disabled={
                mutationCredential.isPending ||
                credential.spec?.type ===
                  CoreC.Credential_Spec_Type.TYPE_UNKNOWN
              }
              onClick={() => mutationCredential.mutate()}
            >
              {mutationCredential.isPending
                ? "Creating…"
                : "Create credential"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export const MainInfo = (props: { item: CoreC.User }): ResourceMainInfo => {
  const { item } = props;
  const related = useUserRelatedResources(item);

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
                <span className="text-[0.75rem]">
                  <CopyText value={item.spec.email} />
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
                    <ResourceListLabel
                      key={x}
                      itemRef={ObjectReference.create({
                        apiVersion: "core/v1",
                        kind: "Group",
                        name: x,
                      })}
                    />
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
                    <ResourceListLabel
                      key={x}
                      itemRef={ObjectReference.create({
                        apiVersion: "core/v1",
                        kind: "Policy",
                        name: x,
                      })}
                    />
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      ...(item.spec?.authorization?.inlinePolicies.length
        ? [
            {
              label: "Inline policies",
              value: (
                <div className="flex flex-wrap gap-1">
                  {item.spec.authorization.inlinePolicies.map((policy) => (
                    <ResourceListLabel key={policy.name} label="Inline policy">
                      <Shield size={12} strokeWidth={2.5} />
                      {policy.name}
                    </ResourceListLabel>
                  ))}
                </div>
              ),
              span: "full" as const,
            },
          ]
        : []),

      {
        label: "Related resources",
        value: (
          <div className="flex flex-wrap gap-1">
            {related.map(({ label, count, path, icon: Icon }) => (
              <ResourceListLabel key={label} label={label} to={path}>
                <Icon size={12} strokeWidth={2.5} />
                {count === undefined ? "…" : count.toLocaleString()}
              </ResourceListLabel>
            ))}
          </div>
        ),
        span: "full" as const,
      },

      {
        label: "Active",
        value: <UserActiveControl item={item} />,
      },
      {
        label: "Credential",
        value: <CreateUserCredential item={item} />,
      },
    ],
  };
};
