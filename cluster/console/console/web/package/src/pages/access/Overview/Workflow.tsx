import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
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
  CircleCheck,
  CircleX,
  ClipboardCheck,
  Clock3,
  Layers,
  ListChecks,
  PowerOff,
  Shield,
  Timer,
  UserCheck,
} from "lucide-react";
import { useAccessCreated, useAccessTotals } from "./queries";

const Workflow = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.high);

  const reviews = totals.data?.access?.review;
  const policies = totals.data?.access?.policy;
  const catalogs = totals.data?.access?.catalog;
  const freshReviews = created.data?.access?.review;

  return (
    <Panel
      icon={ClipboardCheck}
      title="Review workflow & Policies"
      description="The rules that decide the Requests and the reviewers who act on them"
      to="/access/policies"
      toLabel="Access Policies"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Open reviews"
            value={n(reviews?.totalPending)}
            tone={n(reviews?.totalPending) > 0 ? "warning" : "default"}
            icon={Clock3}
            to="/access/reviews?isDecided=false"
          />
          <MiniStat
            label="Reviewers"
            value={n(policies?.totalReviewer)}
            icon={UserCheck}
          />
          <MiniStat
            label="Review steps"
            value={n(policies?.totalReviewStep)}
            icon={ListChecks}
          />
          <MiniStat
            label="Policies"
            value={n(policies?.totalNumber)}
            icon={Shield}
            to="/access/policies"
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
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <Breakdown
            title="Rule effects"
            note={`${compact(n(policies?.totalRule))} rules`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Auto-approve",
                  value: n(policies?.totalRuleAutoApprove),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Review",
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
                With authorization
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(policies?.totalRuleAuthorization))}
                </span>
              </span>
              <span>
                Capped duration
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(policies?.totalRuleMaxAccessDuration))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title="Review decisions"
            note={`${compact(n(reviews?.totalNumber))} all time`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Approved",
                  value: n(reviews?.totalApproved),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Pending",
                  value: n(reviews?.totalPending),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Rejected",
                  value: n(reviews?.totalRejected),
                  color: STATUS_COLORS.critical,
                },
                { label: "Revised", value: n(reviews?.totalRevised) },
              ]}
              total={n(reviews?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Reviewers active
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(reviews?.totalUser))}
                </span>
              </span>
              <span>
                Requests covered
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(reviews?.totalRequest))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title="Catalog coverage"
            note={`${compact(n(catalogs?.totalNumber))} Catalogs`}
          >
            <CompositionBar
              segments={[
                { label: "Services", value: n(catalogs?.totalService) },
                { label: "Namespaces", value: n(catalogs?.totalNamespace) },
              ]}
              total={n(catalogs?.totalService) + n(catalogs?.totalNamespace)}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Reviews · ${rangeLabel}`}
            value={n(freshReviews?.totalNumber)}
            prev={n(freshReviews?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={ClipboardCheck}
            to="/access/reviews"
          />
          <DeltaStat
            label="Approving"
            value={n(freshReviews?.totalApproved)}
            prev={n(freshReviews?.previous?.totalApproved)}
            rangeLabel={rangeLabel}
            icon={CircleCheck}
            to="/access/reviews?decision=DECISION_APPROVE"
          />
          <DeltaStat
            label="Rejecting"
            value={n(freshReviews?.totalRejected)}
            prev={n(freshReviews?.previous?.totalRejected)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={CircleX}
            to="/access/reviews?decision=DECISION_REJECT"
          />
          <DeltaStat
            label="Still open"
            value={n(freshReviews?.totalPending)}
            prev={n(freshReviews?.previous?.totalPending)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={Timer}
            to="/access/reviews?isDecided=false"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Workflow;
