import * as AccessP from "@/apis/accessv1/accessv1";
import * as React from "react";

import { Select, Textarea } from "@mantine/core";

const Edit = (props: {
  item: AccessP.Review;
  onUpdate: (item: AccessP.Review) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Review.clone(item));
  const updateReq = () => {
    setReq(AccessP.Review.clone(req));
    onUpdate(req);
  };

  return (
    <div className="w-full">
      <Select
        label="Decision"
        required
        description="Set the review decision"
        data={[
          {
            label: "Approve",
            value:
              AccessP.Review_Spec_Decision[
                AccessP.Review_Spec_Decision.APPROVE
              ],
          },
          {
            label: "Reject",
            value:
              AccessP.Review_Spec_Decision[AccessP.Review_Spec_Decision.REJECT],
          },
        ]}
        value={AccessP.Review_Spec_Decision[req.spec!.decision]}
        onChange={(v) => {
          if (!v) return;
          req.spec!.decision = AccessP.Review_Spec_Decision[v as "APPROVE"];
          updateReq();
        }}
      />

      <Textarea
        label="Justification"
        description="Set the justification for the decision"
        placeholder="Approving because ..."
        autosize
        minRows={2}
        maxRows={6}
        value={req.spec!.justification}
        onChange={(e) => {
          req.spec!.justification = e.currentTarget.value;
          updateReq();
        }}
      />
    </div>
  );
};

export default Edit;
