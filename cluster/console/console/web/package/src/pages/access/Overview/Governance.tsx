import { ListRequestTopRequest_Dimension } from "@/apis/visibilityv1/access/vaccessv1";
import {
  Breakdown,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  ClipboardCheck,
  Layers,
  ListChecks,
  MousePointerClick,
  PowerOff,
  Shield,
  UserCheck,
} from "lucide-react";
import {
  useAccessCreated,
  useAccessTotals,
  useRequestTop,
  useReviewerTop,
} from "./queries";
import { TopRefList } from "./shared";

const Governance = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.high);
  const reviewers = useReviewerTop(periodMinutes);
  const policiesTop = useRequestTop(
    ListRequestTopRequest_Dimension.POLICY,
    periodMinutes,
  );

  const policies = totals.data?.access?.policy;
  const catalogs = totals.data?.access?.catalog;
  const reviews = created.data?.access?.review;

  return (
    <Panel
      icon={ClipboardCheck}
      title="Reviews & policies"
      description={`Who reviews, and the rules that decide · activity of the last ${rangeLabel}`}
      to="/access/policies"
      toLabel="Policies"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Policies"
            value={n(policies?.totalNumber)}
            icon={Shield}
            to="/access/policies"
          />
          <MiniStat
            label="Rules"
            value={n(policies?.totalRule)}
            icon={ListChecks}
          />
          <MiniStat
            label="Disabled"
            value={n(policies?.totalDisabled)}
            tone={n(policies?.totalDisabled) > 0 ? "warning" : "default"}
            icon={PowerOff}
            to="/access/policies?isDisabled=true"
          />
          <MiniStat
            label="Catalogs"
            value={n(catalogs?.totalNumber)}
            icon={Layers}
            to="/access/catalogs"
          />
          <MiniStat
            label={`Reviews · ${rangeLabel}`}
            value={n(reviews?.totalNumber)}
            icon={UserCheck}
            to="/access/reviews"
          />
          <MiniStat
            label="Decided in a chat"
            value={n(reviews?.totalFromIntegration)}
            icon={MousePointerClick}
            to="/access/reviews?originType=INTEGRATION"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Rule effects"
            note={`${compact(n(policies?.totalRule))} rules · ${compact(n(policies?.totalReviewStep))} review steps`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Auto-approve",
                  value: n(policies?.totalRuleAutoApprove),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Require review",
                  value: n(policies?.totalRuleReview),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Deny",
                  value: n(policies?.totalRuleDeny),
                  color: STATUS_COLORS.critical,
                },
              ]}
              total={n(policies?.totalRule)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Review surfaces
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(policies?.totalReviewSurface))}
                </span>
              </span>
              <span>
                Interactive
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(policies?.totalInteractiveSurface))}
                </span>
              </span>
              <span>
                Notifying rules
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(policies?.totalRuleNotification))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title={`Review decisions · ${rangeLabel}`}
            note={`${compact(n(reviews?.totalUser))} reviewers`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Approved",
                  value: n(reviews?.totalApproved),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Rejected",
                  value: n(reviews?.totalRejected),
                  color: STATUS_COLORS.critical,
                },
                {
                  label: "Withdrawn",
                  value: n(reviews?.totalPending),
                  color: STATUS_COLORS.warning,
                },
              ]}
              total={n(reviews?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Changed at least once
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(reviews?.totalRevised))}
                </span>
              </span>
              <span>
                Requests reviewed
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(reviews?.totalRequest))}
                </span>
              </span>
            </div>
          </Breakdown>
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <TopRefList
            title="Most active reviewers"
            icon={UserCheck}
            avatars
            items={reviewers.data?.items}
            totalOther={n(reviewers.data?.totalOther)}
            isLoading={reviewers.isLoading}
            emptyLabel={`No review in the last ${rangeLabel}`}
          />
          <TopRefList
            title="Policies that matched"
            icon={Shield}
            items={policiesTop.data?.items}
            totalOther={n(policiesTop.data?.totalOther)}
            isLoading={policiesTop.isLoading}
            emptyLabel={`No request matched a Policy in the last ${rangeLabel}`}
          />
        </div>
      </div>
    </Panel>
  );
};

export default Governance;
