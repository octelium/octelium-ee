import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";
import { cloneResource } from "@/utils/pb";

import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import TimestampPicker from "@/components/TimestampPicker";
import { randomStringLowerCase } from "@/utils";
import { Group, Select, Switch } from "@mantine/core";
import { match } from "ts-pattern";

const Edit = (props: {
  item: CoreP.Credential;
  onUpdate: (item: CoreP.Credential) => void;
}) => {
  let [req, setReq] = React.useState<CoreP.Credential>(props.item);
  const data = props.item;

  React.useEffect(() => {
    if (data) {
      setReq(CoreP.Credential.clone(data));
    }
  }, [data]);

  const updateReq = () => {
    const clone = cloneResource(req) as CoreP.Credential;
    setReq(clone);

    props.onUpdate(clone);
  };
  const setName = () => {
    const isCreate = props.item.metadata!.uid.length === 0;
    if (!isCreate || req.spec!.user.length === 0) {
      return;
    }

    if (!req.metadata!.name.startsWith(req.spec!.user)) {
      return;
    }

    req.metadata!.name = `${req.spec!.user}-${match(req.spec!.type)
      .with(CoreP.Credential_Spec_Type.AUTH_TOKEN, () => "auth-token")
      .with(CoreP.Credential_Spec_Type.ACCESS_TOKEN, () => "access-token")
      .with(
        CoreP.Credential_Spec_Type.OAUTH2,
        () => "oauth2",
      )}-${randomStringLowerCase(4)}`;
  };

  if (!req) {
    return <></>;
  }

  return (
    <div>
      <Group grow>
        <SelectResource
          api="core"
          kind="User"
          label="Select the User"
          description="Select the User that will use this Credential to authenticate to the Cluster"
          required
          defaultValue={req.spec!.user}
          onChange={(v) => {
            req.spec!.user = v?.metadata?.name ?? "";
            setName();
            updateReq();
          }}
        />

        <Select
          label="Type"
          description="The Credential must be either an Authentication Token or a OAuth2 Client Credential"
          required
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
                CoreP.Credential_Spec_Type[CoreP.Credential_Spec_Type.OAUTH2],
            },
            {
              label: "Access Token",
              value:
                CoreP.Credential_Spec_Type[
                  CoreP.Credential_Spec_Type.ACCESS_TOKEN
                ],
            },
          ]}
          defaultValue={CoreP.Credential_Spec_Type[req.spec!.type]}
          onChange={(v) => {
            req.spec!.type = CoreP.Credential_Spec_Type[v as "OAUTH2"];
            setName();
            updateReq();
          }}
        />
      </Group>

      <Group grow>
        <TimestampPicker
          label="Expires at"
          description="Set the expiration time for the Credential"
          value={req.spec!.expiresAt}
          isFuture
          onChange={(v) => {
            req.spec!.expiresAt = v;
            updateReq();
          }}
        />

        <Switch
          label="Disabled"
          description="Disable/deactivate the Credential"
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>

      <EditItem
        title="Authorization"
        description="Set the Credential Policies"
        onUnset={() => {
          req.spec!.authorization = undefined;
          updateReq();
        }}
        obj={req.spec!.authorization}
        onSet={() => {
          if (!req.spec!.authorization) {
            req.spec!.authorization =
              CoreP.Credential_Spec_Authorization.create({
                policies: [],
              });
            updateReq();
          }
        }}
      >
        {req.spec!.authorization && (
          <>
            <SelectPolicies
              policies={req.spec!.authorization.policies}
              onUpdate={(v) => {
                if (!v) {
                  req.spec!.authorization!.policies = [];
                } else {
                  req.spec!.authorization!.policies = v;
                }

                updateReq();
              }}
            />

            <SelectInlinePolicies
              inlinePolicies={req.spec!.authorization!.inlinePolicies}
              onUpdate={(v) => {
                req.spec!.authorization!.inlinePolicies = v;
                updateReq();
              }}
            />
          </>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
