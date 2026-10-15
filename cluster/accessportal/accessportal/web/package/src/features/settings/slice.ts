import Settings from "@/utils/types/settings";
import { PayloadAction, createSlice } from "@reduxjs/toolkit";

import * as UserPB from "@/apis/userv1/userv1";

export const slice = createSlice({
  name: "settings",
  initialState: {} as Settings,
  reducers: {
    setStatus: (
      state,
      action: PayloadAction<{ status: UserPB.GetStatusResponse }>
    ) => {
      state.status = action.payload.status;
    },
  },
});

export const { setStatus } = slice.actions;

export default slice.reducer;
