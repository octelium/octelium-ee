import { StatTile } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { useEnterpriseCreated, useEnterpriseTotals } from "./queries";

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  useChartColorScheme();

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useEnterpriseCreated(periodMinutes, QUERY_PRIORITY.high);

  const certificates = totals.data?.enterprise?.certificate;
  const issuers = totals.data?.enterprise?.certificateIssuer;
  const directories = totals.data?.enterprise?.directoryProvider;
  const directoryUsers = totals.data?.enterprise?.directoryProviderUser;
  const directoryGroups = totals.data?.enterprise?.directoryProviderGroup;
  const stores = totals.data?.enterprise?.secretStore;
  const managers = totals.data?.enterprise?.deviceManager;

  const newDirectoryUsers = created.data?.enterprise?.directoryProviderUser;

  const managerIssues =
    n(managers?.totalError) +
    n(managers?.totalDegraded) +
    n(managers?.totalFailedUpdates);

  return (
    <section
      aria-label="Enterprise signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Expiring soon"
        value={n(certificates?.totalExpiringSoon)}
        badge={
          n(certificates?.totalExpired) > 0
            ? `${compact(n(certificates?.totalExpired))} expired`
            : undefined
        }
        footer={`${compact(n(certificates?.totalNumber))} Certificates · ${compact(n(certificates?.totalManaged))} managed`}
        bar={{
          value: n(certificates?.totalExpiringSoon),
          total: n(certificates?.totalNumber),
          label: "expire within 30 days",
        }}
        color={
          n(certificates?.totalExpiringSoon) > 0
            ? STATUS_COLORS.warning
            : STATUS_COLORS.good
        }
        to="/enterprise/certificates?isExpiringSoon=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Issuance failures"
        value={n(certificates?.totalIssuanceFailed)}
        footer={`${compact(n(issuers?.totalReady))} of ${compact(n(issuers?.totalNumber))} issuers ready`}
        bar={{
          value: n(certificates?.totalIssuanceFailed),
          total: n(certificates?.totalManaged),
          label: "of the managed Certificates",
        }}
        color={STATUS_COLORS.critical}
        to="/enterprise/certificates?issuanceState=FAILED"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Directory sync"
        value={n(directories?.totalSynchronizationFailed)}
        badge={
          n(directories?.totalSynchronizing) > 0
            ? `${compact(n(directories?.totalSynchronizing))} running`
            : undefined
        }
        footer={`${compact(n(directories?.totalSynchronizationSuccess))} of ${compact(n(directories?.totalNumber))} providers in sync`}
        bar={{
          value: n(directories?.totalSynchronizationFailed),
          total: n(directories?.totalNumber),
          label: "failing to synchronize",
        }}
        color={STATUS_COLORS.critical}
        to="/enterprise/directoryproviders?synchronizationState=FAILED"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Secret stores"
        value={n(stores?.totalSynchronizationFailed)}
        footer={`${compact(n(stores?.totalOK))} of ${compact(n(stores?.totalNumber))} stores healthy`}
        bar={{
          value: n(stores?.totalSynchronizationFailed),
          total: n(stores?.totalNumber),
          label: "failing to synchronize",
        }}
        color={STATUS_COLORS.critical}
        to="/enterprise/secretstores?synchronizationState=FAILED"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Device managers"
        value={managerIssues}
        badge={
          n(managers?.totalWaitingApproval) > 0
            ? `${compact(n(managers?.totalWaitingApproval))} waiting`
            : undefined
        }
        footer={`${compact(n(managers?.totalLinkedDevices))} of ${compact(n(managers?.totalManagedDevices))} devices linked`}
        bar={{
          value: managerIssues,
          total: n(managers?.totalNumber),
          label: "reporting a problem",
        }}
        color={managerIssues > 0 ? STATUS_COLORS.serious : STATUS_COLORS.good}
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Directory identities"
        value={n(directoryUsers?.totalNumber) + n(directoryGroups?.totalNumber)}
        badge={
          n(newDirectoryUsers?.totalNumber) > 0
            ? `+${compact(n(newDirectoryUsers?.totalNumber))}`
            : undefined
        }
        footer={`${compact(n(directoryUsers?.totalUser))} Users · ${compact(n(directoryGroups?.totalGroup))} Groups linked`}
        bar={{
          value: n(directoryUsers?.totalUser),
          total: n(directoryUsers?.totalNumber),
          label: "of the directory users are linked",
        }}
        color={seriesColor(4)}
        to="/enterprise/directoryproviders"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />
    </section>
  );
};

export default Signals;
