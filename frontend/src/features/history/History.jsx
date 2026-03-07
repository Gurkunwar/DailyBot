import React, { useState, useEffect } from "react";
import Sidebar from "../../components/Sidebar";
import {
  useGetManagedStandupsQuery,
  useGetHistoryQuery,
  useGetStandupByIdQuery,
  useGetManagedPollsQuery,
  useGetPollHistoryQuery,
} from "../../store/apiSlice";

export default function History() {
  const [viewMode, setViewMode] = useState(
    () => localStorage.getItem("historyViewMode") || "standups",
  );
  const [selectedStandupId, setSelectedStandupId] = useState(
    () => localStorage.getItem("historyStandupId") || "",
  );
  const [selectedPollId, setSelectedPollId] = useState(
    () => localStorage.getItem("historyPollId") || "",
  );

  useEffect(() => {
    localStorage.setItem("historyViewMode", viewMode);
  }, [viewMode]);

  useEffect(() => {
    if (selectedStandupId)
      localStorage.setItem("historyStandupId", selectedStandupId);
  }, [selectedStandupId]);

  useEffect(() => {
    if (selectedPollId) localStorage.setItem("historyPollId", selectedPollId);
  }, [selectedPollId]);

  const { data: standupsData, isLoading: isLoadingStandups } =
    useGetManagedStandupsQuery(
      { filter: "all", page: 1, limit: 50 },
      { skip: viewMode !== "standups" },
    );
  const standups = standupsData?.data || [];

  const { data: standupConfig } = useGetStandupByIdQuery(selectedStandupId, {
    skip: viewMode !== "standups" || !selectedStandupId,
  });

  const { data: historyData, isLoading: isLoadingHistory } = useGetHistoryQuery(
    selectedStandupId,
    {
      skip: viewMode !== "standups" || !selectedStandupId,
    },
  );

  const { data: pollsData, isLoading: isLoadingPolls } =
    useGetManagedPollsQuery(
      { filter: "all", page: 1, limit: 50 },
      { skip: viewMode !== "polls" },
    );
  const polls = pollsData?.data || [];

  const { data: pollHistoryData, isLoading: isLoadingPollHistory } =
    useGetPollHistoryQuery(selectedPollId, {
      skip: viewMode !== "polls" || !selectedPollId,
    });

  useEffect(() => {
    if (viewMode === "standups" && standups.length > 0) {
      const isValid = standups.some(
        (st) => st.id.toString() === selectedStandupId,
      );
      if (!isValid) {
        setSelectedStandupId(standups[0].id.toString());
      }
    }
  }, [standups, selectedStandupId, viewMode]);

  useEffect(() => {
    if (viewMode === "polls" && polls.length > 0) {
      const isValid = polls.some((p) => p.id.toString() === selectedPollId);
      if (!isValid) {
        setSelectedPollId(polls[0].id.toString());
      }
    }
  }, [polls, selectedPollId, viewMode]);

  const getDiscordAvatarUrl = (avatarStr, userId) => {
    if (!avatarStr || avatarStr === "0" || avatarStr === "") {
      const id = userId ? BigInt(userId) : 0n;
      return `https://cdn.discordapp.com/embed/avatars/${Number((id >> 22n) % 6n)}.png`;
    }
    if (avatarStr.startsWith("http")) return avatarStr;
    const cleanHash = avatarStr.includes("/")
      ? avatarStr.split("/")[1]
      : avatarStr;
    return `https://cdn.discordapp.com/avatars/${userId}/${cleanHash}.png`;
  };

  const handleExportCSV = () => {
    let csvContent = "";
    let filename = "";

    if (viewMode === "standups") {
      if (!historyData || historyData.length === 0 || !standupConfig) return;
      const questions =
        standupConfig.questions || standupConfig.Questions || [];
      const headers = ["Date", "User", "Status", ...questions];

      const rows = historyData.map((row) => {
        const isSkipped =
          row.answers.length > 0 && row.answers[0] === "Skipped / OOO";
        const statusStr = isSkipped ? "Skipped" : "Submitted";
        const userStr = `"${row.user_name}"`;
        const dateStr = `"${row.date}"`;
        const answersStr = row.answers.map((a) => `"${a.replace(/"/g, '""')}"`);
        return [dateStr, userStr, statusStr, ...answersStr].join(",");
      });

      csvContent = [headers.join(","), ...rows].join("\n");
      filename = `standup_history_${selectedStandupId}.csv`;
    } else {
      if (!pollHistoryData || pollHistoryData.length === 0) return;
      const headers = ["Date", "User", "Voted For"];

      const rows = pollHistoryData.map((row) => {
        return `"${row.created_at}","${row.user_name}","${row.option.replace(/"/g, '""')}"`;
      });

      csvContent = [headers.join(","), ...rows].join("\n");
      filename = `poll_history_${selectedPollId}.csv`;
    }

    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", filename);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const displayQuestions = standupConfig?.questions || standupConfig?.Questions;
  const isExportDisabled =
    viewMode === "standups"
      ? !historyData || historyData.length === 0
      : !pollHistoryData || pollHistoryData.length === 0;

  return (
    <div className="flex h-screen bg-[#313338] text-white overflow-hidden font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header
          className="h-14 border-b border-[#1e1f22] flex items-center 
        justify-between px-8 shadow-sm shrink-0"
        >
          <h1 className="text-lg font-bold">History & Exports</h1>

          {/* View Mode Toggle */}
          <div className="flex bg-[#1e1f22] p-1 rounded-lg border border-[#3f4147] shadow-sm">
            <button
              onClick={() => setViewMode("standups")}
              className={`px-5 py-1 text-sm font-bold rounded-md transition-all 
                ${
                  viewMode === "standups"
                    ? "bg-[#5865F2] text-white shadow"
                    : "text-[#99AAB5] hover:text-white"
                }`}
            >
              Standups
            </button>
            <button
              onClick={() => setViewMode("polls")}
              className={`px-5 py-1 text-sm font-bold rounded-md transition-all 
                ${
                  viewMode === "polls"
                    ? "bg-[#38bdf8] text-white shadow"
                    : "text-[#99AAB5] hover:text-white"
                }`}
            >
              Polls
            </button>
          </div>
        </header>

        <div className="flex-1 overflow-y-auto p-8 relative custom-scrollbar">
          <div className="flex justify-between items-center mb-8 max-w-7xl mx-auto">
            <h2 className="text-2xl font-bold">
              {viewMode === "standups" ? "Standup Logs" : "Poll Vote Logs"}
            </h2>
            <button
              onClick={handleExportCSV}
              disabled={isExportDisabled}
              className="bg-[#43b581] hover:bg-[#3ca374] 
              disabled:bg-[#43b581]/50 disabled:cursor-not-allowed px-4 py-2 
              rounded font-semibold text-sm transition-colors cursor-pointer shadow-md flex items-center gap-2"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                ></path>
              </svg>
              Export CSV
            </button>
          </div>

          <div className="max-w-7xl mx-auto">
            {/* Target Selector */}
            <div
              className="mb-6 bg-[#2b2d31] p-4 rounded-xl border 
            border-[#1e1f22] shadow-sm flex items-center gap-4"
            >
              <span className="text-sm font-bold text-[#99AAB5]">
                {viewMode === "standups" ? "Target Team:" : "Target Poll:"}
              </span>

              {viewMode === "standups" ? (
                isLoadingStandups ? (
                  <span className="text-sm text-[#99AAB5] animate-pulse">
                    Loading teams...
                  </span>
                ) : standups.length === 0 ? (
                  <span className="text-sm text-[#da373c]">
                    You are not managing any standups.
                  </span>
                ) : (
                  <select
                    value={selectedStandupId}
                    onChange={(e) => setSelectedStandupId(e.target.value)}
                    className="bg-[#1e1f22] text-sm font-medium text-white 
                    px-4 py-2 rounded-md outline-none border border-[#3f4147] 
                    focus:border-[#5865F2] cursor-pointer min-w-64 shadow-inner"
                  >
                    {standups.map((st) => (
                      <option key={st.id} value={st.id}>
                        {st.name} — ({st.guild_name})
                      </option>
                    ))}
                  </select>
                )
              ) : isLoadingPolls ? (
                <span className="text-sm text-[#99AAB5] animate-pulse">
                  Loading polls...
                </span>
              ) : polls.length === 0 ? (
                <span className="text-sm text-[#da373c]">
                  You have not created any polls.
                </span>
              ) : (
                <select
                  value={selectedPollId}
                  onChange={(e) => setSelectedPollId(e.target.value)}
                  className="bg-[#1e1f22] text-sm font-medium text-white px-4 py-2 
                  rounded-md outline-none border border-[#3f4147] focus:border-[#38bdf8] 
                  cursor-pointer min-w-64 shadow-inner"
                >
                  {polls.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.question.length > 50
                        ? p.question.substring(0, 47) + "..."
                        : p.question}{" "}
                      — ({p.guild_name})
                    </option>
                  ))}
                </select>
              )}
            </div>

            {/* Data Table */}
            <div
              className="bg-[#2b2d31] border border-[#1e1f22] rounded-xl 
            shadow-sm flex flex-col overflow-hidden"
            >
              <div className="overflow-x-auto custom-scrollbar">
                <table className="w-full text-left text-sm whitespace-nowrap">
                  <thead
                    className="bg-[#1e1f22] text-[#99AAB5] font-bold text-[11px] 
                  uppercase tracking-widest border-b border-[#3f4147]"
                  >
                    <tr>
                      <th className="px-5 py-4 w-32">Date</th>
                      <th className="px-5 py-4 w-48">User</th>
                      {viewMode === "standups" ? (
                        <>
                          <th className="px-5 py-4 w-32">Status</th>
                          {displayQuestions?.map((q, i) => (
                            <th
                              key={i}
                              className="px-5 py-4 max-w-62.5 truncate"
                              title={q}
                            >
                              <span className="text-[#5865F2] mr-1">
                                Q{i + 1}:
                              </span>
                              {q}
                            </th>
                          ))}
                        </>
                      ) : (
                        <th className="px-5 py-4">Voted For</th>
                      )}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#3f4147]/50">
                    {(
                      viewMode === "standups"
                        ? isLoadingHistory
                        : isLoadingPollHistory
                    ) ? (
                      <tr>
                        <td
                          colSpan={10}
                          className="px-5 py-12 text-center text-[#99AAB5] font-medium"
                        >
                          Fetching logs...
                        </td>
                      </tr>
                    ) : (viewMode === "standups" &&
                        (!historyData || historyData.length === 0)) ||
                      (viewMode === "polls" &&
                        (!pollHistoryData || pollHistoryData.length === 0)) ? (
                      <tr>
                        <td colSpan={10} className="px-5 py-16 text-center">
                          <span className="text-4xl mb-3 block opacity-50">
                            📭
                          </span>
                          <span className="text-[#99AAB5] font-medium">
                            No logs found yet.
                          </span>
                        </td>
                      </tr>
                    ) : viewMode === "standups" ? (
                      historyData.map((log, index) => {
                        const isSkipped =
                          log.answers.length > 0 &&
                          log.answers[0] === "Skipped / OOO";
                        const totalQuestionCols = displayQuestions
                          ? displayQuestions.length
                          : 1;
                        return (
                          <tr
                            key={log.id}
                            className={`hover:bg-[#35373c]/80 transition-colors 
                              ${index % 2 === 0 ? "bg-[#2b2d31]" : "bg-[#2f3136]"}`}
                          >
                            <td className="px-5 py-3 font-mono text-xs text-[#99AAB5] whitespace-nowrap">
                              {log.date}
                            </td>
                            <td className="px-5 py-3">
                              <div className="flex items-center gap-3">
                                <img
                                  src={getDiscordAvatarUrl(
                                    log.avatar,
                                    log.user_id,
                                  )}
                                  alt="Avatar"
                                  className="w-7 h-7 rounded-full border border-[#1e1f22] shrink-0 object-cover"
                                />
                                <span className="font-semibold text-gray-200 truncate max-w-30">
                                  {log.user_name}
                                </span>
                              </div>
                            </td>
                            <td className="px-5 py-3">
                              {isSkipped ? (
                                <span
                                  className="bg-[#da373c]/10 text-[#da373c] border 
                                border-[#da373c]/20 px-2 py-0.5 rounded text-[10px] font-bold 
                                uppercase tracking-widest inline-flex items-center gap-1.5"
                                >
                                  <svg
                                    className="w-3 h-3"
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      strokeWidth="2"
                                      d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
                                    ></path>
                                  </svg>
                                  Skipped
                                </span>
                              ) : (
                                <span
                                  className="bg-[#43b581]/10 text-[#43b581] 
                                border border-[#43b581]/20 px-2 py-0.5 rounded text-[10px] 
                                font-bold uppercase tracking-widest inline-flex items-center gap-1.5"
                                >
                                  <svg
                                    className="w-3 h-3"
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                  >
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      strokeWidth="2"
                                      d="M5 13l4 4L19 7"
                                    ></path>
                                  </svg>
                                  Submitted
                                </span>
                              )}
                            </td>
                            {isSkipped ? (
                              <td
                                colSpan={totalQuestionCols}
                                className="px-5 py-3 text-center"
                              >
                                <span className="text-[#99AAB5] italic text-sm opacity-60">
                                  No data reported
                                </span>
                              </td>
                            ) : (
                              <>
                                {log.answers.map((ans, i) => (
                                  <td key={i} className="px-5 py-3">
                                    <div
                                      className="max-w-md truncate whitespace-normal 
                                      line-clamp-2 text-sm text-gray-300 leading-relaxed"
                                      title={ans}
                                    >
                                      {ans}
                                    </div>
                                  </td>
                                ))}
                                {displayQuestions &&
                                  log.answers.length <
                                    displayQuestions.length &&
                                  Array.from({
                                    length:
                                      displayQuestions.length -
                                      log.answers.length,
                                  }).map((_, i) => (
                                    <td key={`pad-${i}`} className="px-5 py-3">
                                      <span className="bg-[#1e1f22] text-[#99AAB5] px-2 py-0.5 rounded text-xs">
                                        N/A
                                      </span>
                                    </td>
                                  ))}
                              </>
                            )}
                          </tr>
                        );
                      })
                    ) : (
                      // --- POLLS ROW RENDERER ---
                      pollHistoryData.map((log, index) => (
                        <tr
                          key={log.id}
                          className={`hover:bg-[#35373c]/80 transition-colors 
                            ${index % 2 === 0 ? "bg-[#2b2d31]" : "bg-[#2f3136]"}`}
                        >
                          <td className="px-5 py-3 font-mono text-xs text-[#99AAB5] whitespace-nowrap">
                            {log.created_at}
                          </td>
                          <td className="px-5 py-3">
                            <div className="flex items-center gap-3">
                              <img
                                src={getDiscordAvatarUrl(
                                  log.avatar,
                                  log.user_id,
                                )}
                                alt="Avatar"
                                className="w-7 h-7 rounded-full border border-[#1e1f22] shrink-0 object-cover"
                              />
                              <span className="font-semibold text-gray-200 truncate max-w-30">
                                {log.user_name}
                              </span>
                            </div>
                          </td>
                          <td className="px-5 py-3">
                            <span
                              className="bg-[#38bdf8]/10 text-[#38bdf8] border 
                            border-[#38bdf8]/20 px-3 py-1 rounded text-xs font-bold shadow-sm"
                            >
                              {log.option}
                            </span>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
