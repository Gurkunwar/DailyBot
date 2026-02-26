import { configureStore } from "@reduxjs/toolkit";
import { dailyBotApi } from "./apiSlice";

export const store = configureStore({
  reducer: {
    [dailyBotApi.reducerPath]: dailyBotApi.reducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(dailyBotApi.middleware),
});
