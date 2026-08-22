import * as AccessP from "@/apis/accessv1/accessv1";
import EditItem from "@/components/EditItem";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import { Library, PanelTop } from "lucide-react";
import * as React from "react";

const Edit = (props: {
  item: AccessP.Catalog;
  onUpdate: (item: AccessP.Catalog) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Catalog.clone(item));
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    setReq(AccessP.Catalog.clone(item));
  }, [itemKey]);

  const updateReq = () => {
    const next = AccessP.Catalog.clone(req);
    setReq(next);
    onUpdate(AccessP.Catalog.clone(next));
  };

  const ensureCollection = () => {
    const spec = req.spec!;
    if (!spec.resourceCollection) {
      spec.resourceCollection =
        AccessP.Catalog_Spec_ResourceCollection.create();
    }
    if (!spec.resourceCollection.service) {
      spec.resourceCollection.service =
        AccessP.Catalog_Spec_ResourceCollection_Service.create();
    }
    return spec.resourceCollection.service;
  };

  const collection = req.spec?.resourceCollection?.service;

  return (
    <div className="w-full">
      <EditItem
        title="Catalog resources"
        description="Include specific Services and every Service belonging to selected Namespaces. At least one must be configured."
        obj={req.spec?.resourceCollection}
        onSet={() => {
          ensureCollection();
          updateReq();
        }}
        onUnset={() => {
          req.spec!.resourceCollection = undefined;
          updateReq();
        }}
      >
        {req.spec?.resourceCollection && (
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <section className="rounded-xl border border-slate-200 bg-slate-50/50 p-3.5">
              <div className="mb-3 flex items-start gap-2.5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm">
                  <PanelTop size={14} strokeWidth={2.2} />
                </span>
                <div>
                  <p className="text-[0.75rem] font-bold text-slate-700">
                    Individual Services
                  </p>
                  <p className="mt-0.5 text-[0.65rem] font-semibold leading-4 text-slate-400">
                    Include only the selected Services.
                  </p>
                </div>
              </div>
              <SelectResourceMultiple
                api="core"
                kind="Service"
                label="Services"
                description="Services included directly in this Catalog."
                clearable
                defaultValue={collection?.services ?? []}
                onChange={(resources) => {
                  ensureCollection().services =
                    resources?.map((resource) => resource.metadata!.name) ?? [];
                  updateReq();
                }}
              />
            </section>

            <section className="rounded-xl border border-slate-200 bg-slate-50/50 p-3.5">
              <div className="mb-3 flex items-start gap-2.5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm">
                  <Library size={14} strokeWidth={2.2} />
                </span>
                <div>
                  <p className="text-[0.75rem] font-bold text-slate-700">
                    Namespace Services
                  </p>
                  <p className="mt-0.5 text-[0.65rem] font-semibold leading-4 text-slate-400">
                    Include Services belonging to the selected Namespaces.
                  </p>
                </div>
              </div>
              <SelectResourceMultiple
                api="core"
                kind="Namespace"
                label="Namespaces"
                description="Include every Service belonging to these Namespaces."
                clearable
                defaultValue={collection?.namespaces ?? []}
                onChange={(resources) => {
                  ensureCollection().namespaces =
                    resources?.map((resource) => resource.metadata!.name) ?? [];
                  updateReq();
                }}
              />
            </section>
          </div>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
