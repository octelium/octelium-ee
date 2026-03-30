import DeleteResource from "@/components/DeleteResource";
import PageWrap from "@/components/PageWrap";
import { Resource } from "@/utils/pb";
import { useContextResource } from "./utils";

const ResourceActions = (props: { resource: Resource }) => {
  return (
    <div>
      <div className="flex items-center justify-end">
        <div className="flex flex-1 items-center justify-end">
          <div>
            <DeleteResource item={props.resource} onSuccess={() => {}} />
          </div>
        </div>
      </div>
    </div>
  );
};

const ResourceItemActionsPage = () => {
  const ctx = useContextResource();

  if (!ctx) {
    return <></>;
  }

  return (
    <PageWrap qry={ctx}>
      {ctx.data && <ResourceActions resource={ctx.data} />}
    </PageWrap>
  );
};

export default ResourceItemActionsPage;
