import * as UserPB from "@/apis/userv1/userv1";
import * as MetaPB from "@/apis/metav1/metav1";

interface Settings {
  wideTerminal?: boolean;
  terminalFontSize?: number;
  itemsPerPage?: number;
  status?: UserPB.GetStatusResponse;
  personalSpaceUID?: string;
  autoCreateFirstTerminal?: boolean;
  orderBy: MetaPB.CommonListOptions_OrderBy;
  // itemsPerPageNavigator?: number;
  listOptFilter?: any;
  useListSearch?: boolean;
}

export default Settings;
