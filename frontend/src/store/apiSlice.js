import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

export const dailyBotApi = createApi({
  reducerPath: "dailyBotApi",
  baseQuery: fetchBaseQuery({
    baseUrl: `${import.meta.env.VITE_API_BASE_URL}/api/`,
    prepareHeaders: (headers) => {
      const token = localStorage.getItem("token");
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }
      return headers;
    },
  }),
  tagTypes: ["Standup", "Members", "History", "Managed Standups"],
  endpoints: (builder) => ({
    getStandupById: builder.query({
      query: (id) => `standups/get?id=${id}`,
      providesTags: (result, error, id) => [{ type: "Standup", id }],
    }),
    getGuildMembers: builder.query({
      query: (guildId) => `guild-members?guild_id=${guildId}`,
      providesTags: ["Members"],
    }),
    getGuildChannels: builder.query({
      query: (guildId) => `guild-channels?guild_id=${guildId}`,
    }),
    getHistory: builder.query({
      query: (standupId) => `standups/history?standup_id=${standupId}`,
      providesTags: ["History"],
    }),
    getManagedStandups: builder.query({
      query: () => "managed-standups",
      providesTags: ["ManagedStandups"],
    }),
    getUserGuilds: builder.query({
      query: () => "user-guilds",
    }),
    getDashboardStats: builder.query({
      query: () => 'dashboard/stats',
    }),
    toggleMember: builder.mutation({
      query: ({ standupId, userId, isCurrentlyMember }) => ({
        url: `standups/${isCurrentlyMember ? "remove-member" : "add-member"}`,
        method: "POST",
        body: { standup_id: parseInt(standupId), user_id: userId },
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "Standup", id: arg.standupId },
      ],
    }),
    createStandup: builder.mutation({
      query: (payload) => ({
        url: `standups/create`,
        method: "POST",
        body: payload,
      }),
      invalidatesTags: ["Managed Standups"],
    }),
    updateStandup: builder.mutation({
      query: (payload) => ({
        url: `standups/update`,
        method: "POST",
        body: payload,
      }),
      invalidatesTags: (result, error, arg) => [
        { type: "Standup", id: arg.id },
      ],
    }),
  }),
});

export const {
  useGetStandupByIdQuery,
  useGetUserGuildsQuery,
  useGetGuildMembersQuery,
  useGetGuildChannelsQuery,
  useGetHistoryQuery,
  useGetDashboardStats,
  useCreateStandupMutation,
  useToggleMemberMutation,
  useUpdateStandupMutation,
} = dailyBotApi;
