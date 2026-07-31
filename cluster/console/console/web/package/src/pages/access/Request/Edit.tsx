import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import * as React from "react";

import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { getResourceRef } from "@/utils/pb";
import { Group, Select, Textarea } from "@mantine/core";
import { match } from "ts-pattern";

const Edit = (props: {
  item: AccessP.Request;
  onUpdate: (item: AccessP.Request) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Request.clone(item));
  const updateReq = () => {
    setReq(AccessP.Request.clone(req));
    onUpdate(req);
  };

  return (
    <div className="w-full">
      <Group grow>
        <Select
          label="Urgency"
          description="Set the request urgency"
          data={[
            {
              label: "Very Low",
              value:
                AccessP.Request_Spec_Urgency[
                  AccessP.Request_Spec_Urgency.VERY_LOW
                ],
            },
            {
              label: "Low",
              value:
                AccessP.Request_Spec_Urgency[AccessP.Request_Spec_Urgency.LOW],
            },
            {
              label: "Normal",
              value:
                AccessP.Request_Spec_Urgency[
                  AccessP.Request_Spec_Urgency.NORMAL
                ],
            },
            {
              label: "High",
              value:
                AccessP.Request_Spec_Urgency[AccessP.Request_Spec_Urgency.HIGH],
            },
            {
              label: "Very High",
              value:
                AccessP.Request_Spec_Urgency[
                  AccessP.Request_Spec_Urgency.VERY_HIGH
                ],
            },
            {
              label: "Highest",
              value:
                AccessP.Request_Spec_Urgency[
                  AccessP.Request_Spec_Urgency.HIGHEST
                ],
            },
          ]}
          value={AccessP.Request_Spec_Urgency[req.spec!.urgency]}
          onChange={(v) => {
            if (!v) return;
            req.spec!.urgency = AccessP.Request_Spec_Urgency[v as "NORMAL"];
            updateReq();
          }}
        />

        <DurationPicker
          value={req.spec!.duration}
          title="Duration"
          onChange={(v) => {
            req.spec!.duration = v;
            updateReq();
          }}
        />
      </Group>

      <Textarea
        label="Justification"
        description="Set the justification for the access request"
        placeholder="I need access to ..."
        autosize
        minRows={2}
        maxRows={6}
        value={req.spec!.justification}
        onChange={(e) => {
          req.spec!.justification = e.currentTarget.value;
          updateReq();
        }}
      />

      <EditItem
        title="Resource"
        description="Set the requested Resource"
        onUnset={() => {
          req.spec!.resource = undefined;
          updateReq();
        }}
        obj={req.spec!.resource}
        onSet={() => {
          req.spec!.resource = AccessP.Request_Spec_Resource.create({
            type: {
              oneofKind: "serviceRef",
              serviceRef: MetaP.ObjectReference.create(),
            },
          });
          updateReq();
        }}
      >
        {req.spec!.resource && (
          <div>
            <Select
              label="Resource type"
              required
              description="Request a Service or a Catalog"
              data={[
                { label: "Service", value: "serviceRef" },
                { label: "Catalog", value: "catalog" },
              ]}
              value={req.spec!.resource.type.oneofKind}
              onChange={(v) => {
                req.spec!.resource!.type = match(v)
                  .with("catalog", () => ({
                    oneofKind: "catalog" as const,
                    catalog: AccessP.Request_Spec_Resource_Catalog.create({
                      catalogRef: MetaP.ObjectReference.create(),
                    }),
                  }))
                  .otherwise(() => ({
                    oneofKind: "serviceRef" as const,
                    serviceRef: MetaP.ObjectReference.create(),
                  }));
                updateReq();
              }}
            />

            {req.spec!.resource.type.oneofKind === "serviceRef" && (
              <SelectResource
                api="core"
                kind="Service"
                label="Service"
                description="Select the requested Service"
                defaultValue={req.spec!.resource.type.serviceRef.name}
                onChange={(v) => {
                  if (req.spec!.resource!.type.oneofKind === "serviceRef") {
                    req.spec!.resource!.type.serviceRef = v
                      ? getResourceRef(v)
                      : MetaP.ObjectReference.create();
                    updateReq();
                  }
                }}
              />
            )}

            {req.spec!.resource.type.oneofKind === "catalog" && (
              <SelectResource
                api="access"
                kind="Catalog"
                label="Catalog"
                description="Select the requested Catalog"
                defaultValue={req.spec!.resource.type.catalog.catalogRef?.name}
                onChange={(v) => {
                  if (req.spec!.resource!.type.oneofKind === "catalog") {
                    req.spec!.resource!.type.catalog.catalogRef = v
                      ? getResourceRef(v)
                      : MetaP.ObjectReference.create();
                    updateReq();
                  }
                }}
              />
            )}
          </div>
        )}
      </EditItem>

      <EditItem
        title="Subject"
        description="Set the User the access is requested for"
        onUnset={() => {
          req.spec!.subject = undefined;
          updateReq();
        }}
        obj={req.spec!.subject}
        onSet={() => {
          req.spec!.subject = AccessP.Request_Spec_Subject.create({
            type: {
              oneofKind: "userRef",
              userRef: MetaP.ObjectReference.create(),
            },
          });
          updateReq();
        }}
      >
        {req.spec!.subject &&
          req.spec!.subject.type.oneofKind === "userRef" && (
            <SelectResource
              api="core"
              kind="User"
              label="User"
              description="Select the subject User"
              defaultValue={req.spec!.subject.type.userRef.name}
              onChange={(v) => {
                if (req.spec!.subject!.type.oneofKind === "userRef") {
                  req.spec!.subject!.type.userRef = v
                    ? getResourceRef(v)
                    : MetaP.ObjectReference.create();
                  updateReq();
                }
              }}
            />
          )}
      </EditItem>
    </div>
  );
};

export default Edit;
