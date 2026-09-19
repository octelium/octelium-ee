import * as AccessC from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import Label from "@/components/Label";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { ResourceListLabel } from "@/components/ResourceList";
import { onError } from "@/utils";
import { getClientAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import {
  Alert,
  Button,
  Modal,
  SegmentedControl,
  TextInput,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { CircleCheck, CircleX, Link2, X } from "lucide-react";
import * as React from "react";
import { getSourceLabel } from "../IntegrationIdentity/utils";
import { hasCapability } from "./utils";

type Direction = "externalID" | "userRef";

const ResolveIdentity = (props: { item: AccessC.Integration }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const [direction, setDirection] = React.useState<Direction>("externalID");
  const [externalID, setExternalID] = React.useState("");
  const [userName, setUserName] = React.useState<string>();
  const [result, setResult] =
    React.useState<AccessC.ResolveIntegrationIdentityResponse>();

  const canResolve = hasCapability(
    props.item,
    AccessC.Integration_Status_Capability.IDENTITY_RESOLUTION,
  );

  const reset = () => {
    setResult(undefined);
    setExternalID("");
    setUserName(undefined);
  };

  const handleClose = () => {
    reset();
    close();
  };

  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientAccess().resolveIntegrationIdentity({
          integrationRef: getResourceRef(props.item),
          type:
            direction === "externalID"
              ? { oneofKind: "externalID", externalID: externalID.trim() }
              : {
                  oneofKind: "userRef",
                  userRef: ObjectReference.create({
                    apiVersion: "core/v1",
                    kind: "User",
                    name: userName,
                  }),
                },
        })
      ).response,
    onSuccess: (response) => setResult(response),
    onError,
  });

  const isReady =
    direction === "externalID" ? externalID.trim().length > 0 : !!userName;

  return (
    <>
      <Button
        type="button"
        variant="default"
        size="sm"
        leftSection={<Link2 size={13} />}
        onClick={open}
      >
        Resolve identity
      </Button>
      <Modal
        opened={opened}
        onClose={handleClose}
        centered
        size="lg"
        withCloseButton={false}
        padding={0}
        styles={{ content: { borderRadius: 14, overflow: "hidden" } }}
      >
        <div className="bg-white">
          <header className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <div className="flex items-center gap-2.5">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Link2 size={15} />
              </span>
              <div>
                <h2 className="text-sm font-bold text-slate-900">
                  Resolve an identity
                </h2>
                <p className="text-xs font-normal text-slate-500">
                  {props.item.metadata?.displayName ||
                    props.item.metadata?.name}
                </p>
              </div>
            </div>
            <Button
              type="button"
              variant="subtle"
              color="gray"
              size="compact-xs"
              onClick={handleClose}
            >
              <X size={14} />
            </Button>
          </header>

          <div className="space-y-4 px-5 py-5">
            {!canResolve && (
              <Alert color="amber" title="No identity resolution">
                This Integration does not advertise the identity resolution
                capability, so only the already established
                IntegrationIdentities can be looked up.
              </Alert>
            )}

            <SegmentedControl
              fullWidth
              value={direction}
              onChange={(value) => {
                setDirection(value as Direction);
                reset();
              }}
              data={[
                { label: "External actor → User", value: "externalID" },
                { label: "User → External actor", value: "userRef" },
              ]}
            />

            {direction === "externalID" ? (
              <TextInput
                label="External actor ID"
                description="Stable identifier of the external actor within the provider (e.g. a Slack user ID or an Atlassian account ID)."
                placeholder="U01ABC234DE"
                value={externalID}
                onChange={(event) => {
                  setExternalID(event.target.value);
                  setResult(undefined);
                }}
              />
            ) : (
              <SelectResource
                api="core"
                kind="User"
                label="Cluster User"
                description="User whose external actor is looked up."
                defaultValue={userName}
                onChange={(user) => {
                  setUserName(user?.metadata?.name);
                  setResult(undefined);
                }}
              />
            )}

            {result && (
              <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/60 p-3.5">
                <div className="flex items-center gap-2">
                  {result.isResolved ? (
                    <CircleCheck size={15} className="text-emerald-600" />
                  ) : (
                    <CircleX size={15} className="text-red-600" />
                  )}
                  <span className="text-sm font-semibold text-slate-800">
                    {result.isResolved ? "Resolved" : "Not resolved"}
                  </span>
                </div>

                {result.detail && (
                  <p className="text-xs font-normal text-slate-600">
                    {result.detail}
                  </p>
                )}

                <div className="flex flex-wrap items-center gap-1.5">
                  {(result.userRef?.name || result.userRef?.uid) && (
                    <ResourceListLabel
                      itemRef={ObjectReference.create({
                        ...result.userRef,
                        apiVersion: result.userRef.apiVersion || "core/v1",
                        kind: result.userRef.kind || "User",
                      })}
                    />
                  )}
                  {result.externalID && (
                    <ResourceListLabel label="External actor">
                      {result.externalID}
                    </ResourceListLabel>
                  )}
                  {(result.integrationIdentityRef?.name ||
                    result.integrationIdentityRef?.uid) && (
                    <ResourceListLabel
                      itemRef={ObjectReference.create({
                        ...result.integrationIdentityRef,
                        apiVersion:
                          result.integrationIdentityRef.apiVersion ||
                          "access/v1",
                        kind:
                          result.integrationIdentityRef.kind ||
                          "IntegrationIdentity",
                      })}
                    />
                  )}
                  {result.isResolved && (
                    <Label size="sm" tone="info">
                      {getSourceLabel(result.source)}
                    </Label>
                  )}
                </div>
              </div>
            )}
          </div>

          <footer className="flex justify-end gap-2 border-t border-slate-200 bg-slate-50/60 px-5 py-3.5">
            <Button
              type="button"
              variant="default"
              size="sm"
              disabled={mutation.isPending}
              onClick={handleClose}
            >
              Close
            </Button>
            <Button
              type="button"
              color="ink"
              size="sm"
              disabled={!isReady}
              loading={mutation.isPending}
              leftSection={<Link2 size={13} />}
              onClick={() => mutation.mutate()}
            >
              Resolve
            </Button>
          </footer>
        </div>
      </Modal>
    </>
  );
};

export default ResolveIdentity;
