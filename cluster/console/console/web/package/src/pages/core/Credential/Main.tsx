import * as CoreC from "@/apis/corev1/corev1";
import { onError } from "@/utils";
import { getClientCore } from "@/utils/client";
import {
  getResourceRef,
  invalidateResource,
  invalidateResourceList,
} from "@/utils/pb";
import { Button, Modal, Select, Switch } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";

import * as CoreP from "@/apis/corev1/corev1";
import CopyText from "@/components/CopyText";
import InfoItem from "@/components/InfoItem";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo from "@/components/TimeAgo";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceMainInfo } from "@/pages/utils/types";
import { useDisclosure } from "@mantine/hooks";
import { RefreshCcw, RefreshCw } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

const GenerateC = (props: { item: CoreC.Credential }) => {
  const { item } = props;
  let [tkn, setTkn] = React.useState<CoreC.CredentialToken | undefined>(
    undefined,
  );

  let [enabled, setEnabled] = React.useState(false);

  const mutationGenerate = useMutation({
    mutationFn: async () => {
      const { response } = await getClientCore().generateCredentialToken(
        CoreC.GenerateCredentialTokenRequest.create({
          credentialRef: getResourceRef(item),
        }),
      );
      return response;
    },

    onSuccess: (response) => {
      setTkn(response);
      invalidateResource(item);
      invalidateResourceList(item);
    },
    onError,
  });

  return (
    <div className="w-full">
      <div className="flex flex-col items-center justify-center my-8 w-full">
        <div className="w-full flex items-center justify-center">
          <Button
            onClick={() => {
              mutationGenerate.mutate();
            }}
            loading={mutationGenerate.isPending}
            leftSection={<RefreshCcw />}
            disabled={!enabled}
          >
            Generate Token
          </Button>
        </div>

        <div className="w-full flex items-center my-4 justify-center">
          <Switch
            checked={enabled}
            label={
              <div className="font-bold text-sm text-center text-slate-600">
                Generating a new token immediately invalidates the previous one
              </div>
            }
            onChange={(event) => setEnabled(event.currentTarget.checked)}
          />
        </div>
      </div>
      <div className="w-full min-h-[100px]">
        {tkn && (
          <div className="w-full flex items-center justify-center">
            {tkn.type.oneofKind === `authenticationToken` && (
              <div className="w-full flex items-center justify-center">
                <InfoItem title="Authentication Token">
                  <CopyText
                    value={tkn.type.authenticationToken.authenticationToken}
                    truncate={20}
                  />
                </InfoItem>
              </div>
            )}
            {tkn.type.oneofKind === `accessToken` && (
              <div className="w-full flex items-center justify-center">
                <InfoItem title="Access Token">
                  <CopyText
                    value={tkn.type.accessToken.accessToken}
                    truncate={20}
                  />
                </InfoItem>
              </div>
            )}
            {tkn.type.oneofKind === `oauth2Credentials` && (
              <div className="w-full">
                <InfoItem title="Client ID">
                  <CopyText value={tkn.type.oauth2Credentials.clientID} />
                </InfoItem>

                <InfoItem title="Client Secret">
                  <CopyText
                    value={tkn.type.oauth2Credentials.clientSecret}
                    truncate={20}
                  />
                </InfoItem>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export const ItemInfo = (props: { item: CoreC.Credential }) => {
  let { item } = props;
  const [opened, { open, close }] = useDisclosure(false);
  const mutationUpdate = useUpdateResource();

  return (
    <>
      <InfoItem title="Type">
        <EditItemWrap
          showComponent={
            <span>
              {match(item.spec!.type)
                .with(
                  CoreP.Credential_Spec_Type.AUTH_TOKEN,
                  () => "Authentication Token",
                )
                .with(
                  CoreP.Credential_Spec_Type.OAUTH2,
                  () => "OAuth2 Client Credential",
                )
                .otherwise(() => "")}
            </span>
          }
          editComponent={
            <Select
              data={[
                {
                  label: "Authentication Token",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.AUTH_TOKEN
                    ],
                },
                {
                  label: "OAuth2 Client Credential",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.OAUTH2
                    ],
                },
                {
                  label: "Access Token",
                  value:
                    CoreP.Credential_Spec_Type[
                      CoreP.Credential_Spec_Type.ACCESS_TOKEN
                    ],
                },
              ]}
              defaultValue={CoreP.Credential_Spec_Type[item.spec!.type]}
              onChange={(v) => {
                item.spec!.type = CoreP.Credential_Spec_Type[v as "OAUTH2"];
                mutationUpdate.mutate(item);
              }}
            />
          }
        />
      </InfoItem>
      {item.spec!.maxAuthentications > 0 && (
        <InfoItem title="Max Authentications">
          {item.spec!.maxAuthentications}
        </InfoItem>
      )}
      {item.status!.lastRotationAt && (
        <InfoItem title="Last Rotation">
          <TimeAgo rfc3339={item.status!.lastRotationAt} />
        </InfoItem>
      )}

      {item.spec!.expiresAt && (
        <InfoItem title="Expires at">
          <TimeAgo rfc3339={item.spec!.expiresAt} />
        </InfoItem>
      )}

      {item.status!.totalRotations > 0 && (
        <InfoItem title="Total Rotations">
          <span>{item.status!.totalRotations}</span>
          {item.spec!.maxAuthentications > 0 && (
            <span className="ml-1">
              (Max Authentications {item.spec!.maxAuthentications})
            </span>
          )}
        </InfoItem>
      )}

      {item.status!.totalRotations > 0 && (
        <InfoItem title="Total Authentications">
          {item.status!.totalAuthentications}
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

      <InfoItem title="Generate">
        <Button size={`xs`} onClick={open}>
          Generate/Rotate Token
        </Button>
      </InfoItem>
      <Modal opened={opened} onClose={close} size={"xl"} centered>
        <GenerateC item={item} />
      </Modal>
    </>
  );
};

export default (props: { item: CoreC.Credential }) => {
  const { item } = props;

  return (
    <div className="w-full">
      <ItemInfo item={item} />
    </div>
  );
};

export const MainInfo = (props: {
  item: CoreC.Credential;
}): ResourceMainInfo => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const [opened, { open, close }] = useDisclosure(false);

  return {
    items: [
      {
        label: "User",
        value: (
          <ResourceListLabel itemRef={item.status!.userRef}></ResourceListLabel>
        ),
      },
      {
        label: "Type",
        value: (
          <EditItemWrap
            label="type"
            showComponent={
              <span className="text-[0.75rem] font-semibold text-slate-700">
                {match(item.spec!.type)
                  .with(
                    CoreP.Credential_Spec_Type.AUTH_TOKEN,
                    () => "Authentication Token",
                  )
                  .with(
                    CoreP.Credential_Spec_Type.OAUTH2,
                    () => "OAuth2 Client Credential",
                  )
                  .with(
                    CoreP.Credential_Spec_Type.ACCESS_TOKEN,
                    () => "Access Token",
                  )
                  .otherwise(() => "")}
              </span>
            }
            editComponent={
              <Select
                size="xs"
                data={[
                  {
                    label: "Authentication Token",
                    value:
                      CoreP.Credential_Spec_Type[
                        CoreP.Credential_Spec_Type.AUTH_TOKEN
                      ],
                  },
                  {
                    label: "OAuth2 Client Credential",
                    value:
                      CoreP.Credential_Spec_Type[
                        CoreP.Credential_Spec_Type.OAUTH2
                      ],
                  },
                  {
                    label: "Access Token",
                    value:
                      CoreP.Credential_Spec_Type[
                        CoreP.Credential_Spec_Type.ACCESS_TOKEN
                      ],
                  },
                ]}
                defaultValue={CoreP.Credential_Spec_Type[item.spec!.type]}
                onChange={(v) => {
                  if (!v) return;
                  item.spec!.type = CoreP.Credential_Spec_Type[v as "OAUTH2"];
                  mutationUpdate.mutate(item);
                }}
              />
            }
          />
        ),
      },

      ...(item.spec!.expiresAt
        ? [
            {
              label: "Expires",
              value: (
                <span className="text-[0.75rem] font-semibold text-slate-600">
                  <TimeAgo rfc3339={item.spec!.expiresAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.lastRotationAt
        ? [
            {
              label: "Last rotation",
              value: (
                <span className="text-[0.75rem] font-semibold text-slate-600">
                  <TimeAgo rfc3339={item.status!.lastRotationAt} />
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.totalRotations > 0
        ? [
            {
              label: "Total rotations",
              value: (
                <span className="text-[0.75rem] font-semibold text-slate-700 tabular-nums">
                  {item.status!.totalRotations}
                </span>
              ),
            },
          ]
        : []),

      ...(item.status!.totalAuthentications > 0
        ? [
            {
              label: "Total authentications",
              value: (
                <span className="text-[0.75rem] font-semibold text-slate-700 tabular-nums">
                  {item.status!.totalAuthentications}
                  {item.spec!.maxAuthentications > 0 && (
                    <span className="text-slate-400 font-medium ml-1">
                      / {item.spec!.maxAuthentications} max
                    </span>
                  )}
                </span>
              ),
            },
          ]
        : []),

      ...(item.spec!.maxAuthentications > 0
        ? [
            {
              label: "Max authentications",
              value: (
                <span className="text-[0.75rem] font-semibold text-slate-700 tabular-nums">
                  {item.spec!.maxAuthentications}
                </span>
              ),
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
        label: "Token",
        value: (
          <>
            <button
              onClick={open}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[0.72rem] font-bold text-slate-600 border border-slate-200 bg-white hover:text-slate-900 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)]"
            >
              <RefreshCw size={11} strokeWidth={2.5} />
              Generate / Rotate
            </button>
            <Modal opened={opened} onClose={close} size="xl" centered>
              <GenerateC item={item} />
            </Modal>
          </>
        ),
        span: "full" as const,
      },
    ],
  };
};
