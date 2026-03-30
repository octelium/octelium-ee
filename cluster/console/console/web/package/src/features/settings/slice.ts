import Settings from "@/utils/types/settings";
import { PayloadAction, createSlice } from "@reduxjs/toolkit";

import * as UserPB from "@/apis/userv1/userv1";
import {
  CommonListOptions_OrderBy,
  CommonListOptions_OrderBy_Mode,
  CommonListOptions_OrderBy_Type,
} from "@/apis/metav1/metav1";

export const slice = createSlice({
  name: "settings",
  initialState: {
    wideTerminal: true,
    itemsPerPage: 10,
    orderBy: {
      type: CommonListOptions_OrderBy_Type.NAME,
      mode: CommonListOptions_OrderBy_Mode.ASC,
    },
    // itemsPerPageNavigator: 5,
  } as Settings,
  reducers: {
    setItemsPerPage: (
      state,
      action: PayloadAction<{ itemsPerPage: number }>
    ) => {
      state.itemsPerPage = action.payload.itemsPerPage;
    },

    setStatus: (
      state,
      action: PayloadAction<{ status: UserPB.GetStatusResponse }>
    ) => {
      state.status = action.payload.status;
    },

    setPersonalSpaceUID: (state, action: PayloadAction<{ uid: string }>) => {
      state.personalSpaceUID = action.payload.uid;
    },

    setOrderByType: (
      state,
      action: PayloadAction<{ type: CommonListOptions_OrderBy_Type }>
    ) => {
      state.orderBy.type = action.payload.type;
    },

    setOrderByMode: (
      state,
      action: PayloadAction<{ mode: CommonListOptions_OrderBy_Mode }>
    ) => {
      state.orderBy.mode = action.payload.mode;
    },

    setListOptFilter: (
      state,
      action: PayloadAction<{ listOptFilter?: any }>
    ) => {
      state.listOptFilter = action.payload.listOptFilter;
    },

    setUseListSearch: (
      state,
      action: PayloadAction<{ useListSearch?: boolean }>
    ) => {
      state.useListSearch = action.payload.useListSearch;
    },
  },
});

export const {
  setItemsPerPage,
  setStatus,
  setPersonalSpaceUID,
  setOrderByType,
  setOrderByMode,
  setListOptFilter,
  setUseListSearch,
} = slice.actions;

export default slice.reducer;
