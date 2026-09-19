import { Request_Spec_Urgency } from "@/apis/accessv1/accessv1";
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
import { getUrgencyColor } from "@/pages/access/Request/utils";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  CalendarClock,
  CircleCheck,
  CircleX,
  Inbox,
  Layers,
  RotateCcw,
  Server,
  ShieldCheck,
  UserRound,
  UsersRound,
} from "lucide-react";
import { useAccessCreated, useAccessTotals } from "./queries";

const Requests = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.critical);

  const data = totals.data?.access?.request;
  const fresh = created.data?.access?.request;
  const total = n(data?.totalNumber);

  return (
    <Panel
      icon={Inbox}
      title="Requests"
      description={`How just-in-time access is being asked for and decided · deltas cover the last ${rangeLabel}`}
      to="/access/requests"
      toLabel="All Requests"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Requesters"
            value={n(data?.totalUser)}
            icon={UserRound}
            to="/core/users"
          />
          <MiniStat
            label="Subjects"
            value={n(data?.totalSubjectUser)}
            icon={UsersRound}
            to="/core/users"
          />
          <MiniStat
            label="Services"
            value={n(data?.totalService)}
            icon={Server}
            to="/core/services"
          />
          <MiniStat
            label="Catalogs"
            value={n(data?.totalCatalog)}
            icon={Layers}
            to="/access/catalogs"
          />
          <MiniStat
            label="Policies matched"
            value={n(data?.totalPolicy)}
            icon={ShieldCheck}
            to="/access/policies"
          />
          <MiniStat
            label="With deadline"
            value={n(data?.totalWithDeadline)}
            icon={CalendarClock}
            to="/access/requests?hasDeadline=true"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown title="Outcomes" note={`${compact(total)} all time`}>
            <CompositionBar
              segments={[
                {
                  label: "Approved",
                  value: n(data?.totalApproved),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Pending",
                  value: n(data?.totalPending),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Rejected",
                  value: n(data?.totalRejected),
                  color: STATUS_COLORS.critical,
                },
                {
                  label: "Revoked",
                  value: n(data?.totalRevoked),
                  color: STATUS_COLORS.serious,
                },
                { label: "Expired", value: n(data?.totalExpired) },
                { label: "Cancelled", value: n(data?.totalCancelled) },
              ]}
              total={total}
            />
          </Breakdown>

          <Breakdown
            title="Urgency mix"
            note={`${compact(n(data?.totalActive))} grants live`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Highest",
                  value: n(data?.totalUrgencyHighest),
                  color: getUrgencyColor(Request_Spec_Urgency.HIGHEST),
                },
                {
                  label: "Very high",
                  value: n(data?.totalUrgencyVeryHigh),
                  color: getUrgencyColor(Request_Spec_Urgency.VERY_HIGH),
                },
                {
                  label: "High",
                  value: n(data?.totalUrgencyHigh),
                  color: getUrgencyColor(Request_Spec_Urgency.HIGH),
                },
                {
                  label: "Normal",
                  value: n(data?.totalUrgencyNormal),
                  color: getUrgencyColor(Request_Spec_Urgency.NORMAL),
                },
                {
                  label: "Low",
                  value: n(data?.totalUrgencyLow),
                  color: getUrgencyColor(Request_Spec_Urgency.LOW),
                },
                {
                  label: "Very low",
                  value: n(data?.totalUrgencyVeryLow),
                  color: getUrgencyColor(Request_Spec_Urgency.VERY_LOW),
                },
              ]}
              total={total}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Raised · ${rangeLabel}`}
            value={n(fresh?.totalNumber)}
            prev={n(fresh?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={Inbox}
            to="/access/requests"
          />
          <DeltaStat
            label="Approved"
            value={n(fresh?.totalApproved)}
            prev={n(fresh?.previous?.totalApproved)}
            rangeLabel={rangeLabel}
            icon={CircleCheck}
            to="/access/requests?state=APPROVED"
          />
          <DeltaStat
            label="Rejected"
            value={n(fresh?.totalRejected)}
            prev={n(fresh?.previous?.totalRejected)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={CircleX}
            to="/access/requests?state=REJECTED"
          />
          <DeltaStat
            label="Revoked"
            value={n(fresh?.totalRevoked)}
            prev={n(fresh?.previous?.totalRevoked)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={RotateCcw}
            to="/access/requests?state=REVOKED"
          />
          <DeltaStat
            label="High urgency"
            value={
              n(fresh?.totalUrgencyHigh) +
              n(fresh?.totalUrgencyVeryHigh) +
              n(fresh?.totalUrgencyHighest)
            }
            prev={
              n(fresh?.previous?.totalUrgencyHigh) +
              n(fresh?.previous?.totalUrgencyVeryHigh) +
              n(fresh?.previous?.totalUrgencyHighest)
            }
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={CalendarClock}
            to="/access/requests?urgency=HIGHEST"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Requests;
