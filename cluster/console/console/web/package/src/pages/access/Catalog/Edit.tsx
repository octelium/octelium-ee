import * as AccessP from "@/apis/accessv1/accessv1";
import * as React from "react";

import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { CloseButton } from "@mantine/core";

const Edit = (props: {
  item: AccessP.Catalog;
  onUpdate: (item: AccessP.Catalog) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Catalog.clone(item));
  const updateReq = () => {
    setReq(AccessP.Catalog.clone(req));
    onUpdate(req);
  };

  return (
    <div className="w-full">
      <EditItem
        title="Resource Collection"
        description="Set the Services and Namespaces included in this Catalog"
        onUnset={() => {
          req.spec!.resourceCollection = undefined;
          updateReq();
        }}
        obj={req.spec!.resourceCollection}
        onSet={() => {
          req.spec!.resourceCollection =
            AccessP.Catalog_Spec_ResourceCollection.create({
              service: AccessP.Catalog_Spec_ResourceCollection_Service.create(),
            });
          updateReq();
        }}
      >
        {req.spec!.resourceCollection && (
          <EditItem
            title="Service"
            description="Set the Service-based resource collection"
            onUnset={() => {
              req.spec!.resourceCollection!.service = undefined;
              updateReq();
            }}
            obj={req.spec!.resourceCollection.service}
            onSet={() => {
              req.spec!.resourceCollection!.service =
                AccessP.Catalog_Spec_ResourceCollection_Service.create();
              updateReq();
            }}
          >
            {req.spec!.resourceCollection.service && (
              <div>
                <ItemMessage
                  title="Services"
                  obj={
                    req.spec!.resourceCollection.service.services.length > 0
                      ? req.spec!.resourceCollection.service.services
                      : undefined
                  }
                  isList
                  onSet={() => {
                    req.spec!.resourceCollection!.service!.services = [""];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    req.spec!.resourceCollection!.service!.services.push("");
                    updateReq();
                  }}
                >
                  {req.spec!.resourceCollection.service.services.map(
                    (s, idx) => (
                      <div className="w-full flex items-end mb-3" key={idx}>
                        <CloseButton
                          size="sm"
                          variant="subtle"
                          className="mr-2 mb-1"
                          onClick={() => {
                            req.spec!.resourceCollection!.service!.services.splice(
                              idx,
                              1,
                            );
                            updateReq();
                          }}
                        />
                        <div className="flex-1">
                          <SelectResource
                            api="core"
                            kind="Service"
                            label="Service"
                            description="Select the Service to include"
                            defaultValue={
                              req.spec!.resourceCollection!.service!.services[
                                idx
                              ]
                            }
                            onChange={(v) => {
                              req.spec!.resourceCollection!.service!.services[
                                idx
                              ] = v?.metadata?.name ?? "";
                              updateReq();
                            }}
                          />
                        </div>
                      </div>
                    ),
                  )}
                </ItemMessage>

                <ItemMessage
                  title="Namespaces"
                  obj={
                    req.spec!.resourceCollection.service.namespaces.length > 0
                      ? req.spec!.resourceCollection.service.namespaces
                      : undefined
                  }
                  isList
                  onSet={() => {
                    req.spec!.resourceCollection!.service!.namespaces = [""];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    req.spec!.resourceCollection!.service!.namespaces.push("");
                    updateReq();
                  }}
                >
                  {req.spec!.resourceCollection.service.namespaces.map(
                    (n, idx) => (
                      <div className="w-full flex items-end mb-3" key={idx}>
                        <CloseButton
                          size="sm"
                          variant="subtle"
                          className="mr-2 mb-1"
                          onClick={() => {
                            req.spec!.resourceCollection!.service!.namespaces.splice(
                              idx,
                              1,
                            );
                            updateReq();
                          }}
                        />
                        <div className="flex-1">
                          <SelectResource
                            api="core"
                            kind="Namespace"
                            label="Namespace"
                            description="Select the Namespace whose Services are included"
                            defaultValue={
                              req.spec!.resourceCollection!.service!.namespaces[
                                idx
                              ]
                            }
                            onChange={(v) => {
                              req.spec!.resourceCollection!.service!.namespaces[
                                idx
                              ] = v?.metadata?.name ?? "";
                              updateReq();
                            }}
                          />
                        </div>
                      </div>
                    ),
                  )}
                </ItemMessage>
              </div>
            )}
          </EditItem>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
