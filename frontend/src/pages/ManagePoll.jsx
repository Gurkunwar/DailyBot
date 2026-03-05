import React, { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Sidebar from "../components/Sidebar";
import {
  useGetPollByIdQuery,
  useGetGuildMembersQuery,
  useEndPollMutation,
  useDeletePollMutation,
} from "../store/apiSlice";

const CHART_COLORS = [
  "#2dd4bf",
  "#38bdf8",
  "#818cf8",
  "#c084fc",
  "#e879f9",
  "#f472b6",
  "#fb923c",
];

export default function ManagePoll() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [userModal, setUserModal] = useState(null);

  const { data: poll, isLoading: isPollLoading } = useGetPollByIdQuery(id, {
    pollingInterval: 3000,
  });

  const { data: guildMembers = [] } = useGetGuildMembersQuery(
    poll?.GuildID || poll?.guild_id,
    {
      skip: !poll,
    },
  );

  const [endPollMutation, { isLoading: isEnding }] = useEndPollMutation();
  const [deletePoll, { isLoading: isDeleting }] = useDeletePollMutation();

  const handleEndPoll = async () => {
    if (
      window.confirm(
        "🚨 Are you sure you want to end this poll early? Discord will close voting immediately.",
      )
    ) {
      try {
        await endPollMutation(id).unwrap();
        alert("Poll successfully ended!");
      } catch (err) {
        console.error("Failed to end poll", err);
      }
    }
  };

  const handleDelete = async () => {
    if (
      window.confirm(
        "🚨 Are you sure you want to permanently delete this poll? This cannot be undone.",
      )
    ) {
      try {
        await deletePoll(id).unwrap();
        navigate("/polls"); // Redirect back to the dashboard after deletion
      } catch (err) {
        console.error("Failed to delete poll", err);
        alert("Failed to delete the poll.");
      }
    }
  };

  const handleExportCSV = async () => {
    try {
      const token = localStorage.getItem("token");
      const API_BASE = import.meta.env.VITE_API_BASE_URL;

      const response = await fetch(`${API_BASE}/polls/export?id=${id}`, {
        method: "GET",
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!response.ok) throw new Error("Failed to generate CSV");

      const blob = await response.blob();
      const downloadUrl = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.style.display = "none";
      a.href = downloadUrl;
      a.download = `poll_${id}_results.csv`;

      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(downloadUrl);
      document.body.removeChild(a);
    } catch (err) {
      console.error("Export failed:", err);
      alert(
        "Failed to export CSV. The poll may have been deleted from Discord.",
      );
    }
  };

  if (isPollLoading) {
    return (
      <div className="flex h-screen bg-[#313338] text-white">
        <Sidebar />
        <div className="flex-1 flex justify-center items-center">
          Loading poll analytics...
        </div>
      </div>
    );
  }

  if (!poll) return null;

  const totalVotes = poll.Votes?.length || 0;
  const uniqueVoterIds = [...new Set(poll.Votes?.map((v) => v.UserID) || [])];

  const votedUsers = uniqueVoterIds.map((id) => {
    const member = guildMembers.find((m) => m.id === id);
    const voteRecord = poll.Votes?.find((v) => v.UserID === id);
    const option = poll.Options?.find((o) => o.ID === voteRecord?.OptionID);

    return {
      ...(member || {
        id,
        username: `User ${id.substring(0, 4)}...`,
        avatar: "https://cdn.discordapp.com/embed/avatars/0.png",
      }),
      votedFor: option?.Label || "Unknown",
    };
  });

  const unvotedUsers = guildMembers.filter(
    (m) => !uniqueVoterIds.includes(m.id),
  );

  const totalEligible =
    Math.max(guildMembers.length, uniqueVoterIds.length) || 1;
  const participationRate = Math.round(
    (uniqueVoterIds.length / totalEligible) * 100,
  );

  let cumulativePercent = 0;

  const AvatarGroup = ({ users, label, onViewAll }) => {
    const displayUsers = users.slice(0, 5);
    const overflow = users.length - 5;

    return (
      <div className="flex flex-col">
        <div className="flex -space-x-2">
          {displayUsers.map((u) => (
            <img
              key={u.id}
              src={u.avatar || "https://cdn.discordapp.com/embed/avatars/0.png"}
              onError={(e) => {
                e.target.onerror = null;
                e.target.src = "https://cdn.discordapp.com/embed/avatars/0.png";
              }}
              alt={u.username}
              title={u.username}
              className="w-8 h-8 rounded-full border-2 border-[#2b2d31] bg-[#1e1f22]"
            />
          ))}
          {overflow > 0 && (
            <div
              className="w-8 h-8 rounded-full border-2 border-[#2b2d31] bg-[#1e1f22] flex items-center 
            justify-center text-[10px] font-bold text-[#99AAB5] z-10"
            >
              +{overflow}
            </div>
          )}
          {users.length === 0 && (
            <div className="h-8 flex items-center text-xs text-[#99AAB5]">
              None
            </div>
          )}
        </div>
        <div className="text-[11px] font-medium text-[#99AAB5] mt-2">
          {label}: {users.length}
          <span
            onClick={onViewAll}
            className="text-[#5865F2] cursor-pointer hover:text-[#4752c4] 
          hover:underline ml-1 transition-colors"
          >
            View all
          </span>
        </div>
      </div>
    );
  };

  return (
    <div className="flex h-screen bg-[#313338] text-white font-sans overflow-hidden">
      <Sidebar />

      <main className="flex-1 flex flex-col min-w-0 overflow-y-auto custom-scrollbar">
        {/* HEADER CONTROLS */}
        <div className="flex items-center justify-between p-8 pb-4 shrink-0">
          <div className="flex items-center gap-4">
            <button
              onClick={() => navigate("/polls")}
              className="text-[#99AAB5] hover:text-white transition-colors flex items-center gap-2 text-sm 
              font-semibold bg-[#2b2d31] px-4 py-2 rounded-md border border-[#1e1f22]"
            >
              ← <span className="hidden sm:inline">Back to Polls</span>
            </button>
            <span
              className={`px-3 py-1 rounded-md text-[10px] font-bold uppercase tracking-widest 
              ${poll.IsActive ? "bg-[#2dd4bf]/20 text-[#2dd4bf]" : "bg-[#404249] text-[#99AAB5]"}`}
            >
              {poll.IsActive ? "Live" : "Closed"}
            </span>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={handleExportCSV}
              className="px-4 py-2 rounded-md text-sm font-semibold border 
            border-[#404249] text-[#99AAB5] hover:text-white hover:bg-[#404249] transition-all"
            >
              Export data as CSV
            </button>
            {poll.IsActive && (
              <button
                onClick={handleEndPoll}
                disabled={isEnding}
                className="px-4 py-2 rounded-md text-sm font-semibold border border-[#da373c]/50 text-[#da373c] 
                hover:bg-[#da373c] hover:text-white transition-all shadow-sm"
              >
                {isEnding ? "Ending..." : "End Poll Early"}
              </button>
            )}
            <button
              onClick={handleDelete}
              disabled={isDeleting}
              className="px-4 py-2 rounded-md text-sm font-semibold border border-[#da373c]/30 text-[#da373c] hover:bg-[#da373c] hover:text-white transition-all shadow-sm"
            >
              {isDeleting ? "Deleting..." : "Delete Poll"}
            </button>
          </div>
        </div>

        <div className="px-8 pb-8 space-y-6 max-w-5xl mx-auto w-full">
          {/* CARD 1: PARTICIPATION RATE */}
          <div className="bg-[#2b2d31] p-8 rounded-2xl border border-[#1e1f22] shadow-sm">
            <h2 className="text-sm font-bold text-[#99AAB5] mb-2">
              Participation rate
            </h2>

            <div className="flex items-end gap-3 mb-4">
              <span className="text-5xl font-extrabold">
                {participationRate}%
              </span>
              <span className="text-sm text-[#99AAB5] font-medium mb-1.5">
                Voted: {uniqueVoterIds.length}/{totalEligible}
              </span>
            </div>

            <div className="h-3 w-full bg-[#1e1f22] rounded-full overflow-hidden mb-6">
              <div
                className="h-full bg-[#2dd4bf] rounded-full transition-all duration-1000 ease-out relative"
                style={{ width: `${participationRate}%` }}
              >
                <div className="absolute top-0 left-0 right-0 h-px bg-white/30" />
              </div>
            </div>

            <div className="flex gap-12">
              <AvatarGroup
                users={votedUsers}
                label="Voted"
                onViewAll={() =>
                  setUserModal({ title: "Users Who Voted", users: votedUsers })
                }
              />
              <AvatarGroup
                users={unvotedUsers}
                label="Not voted"
                onViewAll={() =>
                  setUserModal({
                    title: "Users Who Haven't Voted",
                    users: unvotedUsers,
                  })
                }
              />
            </div>
          </div>

          {/* CARD 2: RESULTS & DONUT CHART */}
          <div className="bg-[#2b2d31] p-8 rounded-2xl border border-[#1e1f22] shadow-sm">
            <h2 className="text-xl font-bold mb-8 flex items-center gap-3">
              <span className="w-2 h-2 rounded-full bg-[#2dd4bf]"></span>
              {poll.Question}
            </h2>

            <div className="flex flex-col md:flex-row gap-12 items-center md:items-start">
              <div className="flex-1 space-y-6 w-full">
                {poll.Options?.map((option, index) => {
                  const votesForOption =
                    poll.Votes?.filter((v) => v.OptionID === option.ID)
                      .length || 0;
                  const percentage =
                    totalVotes === 0 ? 0 : (votesForOption / totalVotes) * 100;
                  const color = CHART_COLORS[index % CHART_COLORS.length];

                  return (
                    <div key={option.ID} className="relative group">
                      <div className="flex justify-between items-end mb-2">
                        <span className="text-sm font-semibold text-gray-200">
                          {option.Label}
                        </span>
                        <span className="text-sm font-bold">
                          {Math.round(percentage)}%
                        </span>
                      </div>

                      <div className="h-3 w-full bg-[#1e1f22] rounded-full overflow-hidden relative">
                        <div
                          className="h-full rounded-full transition-all duration-1000 ease-out"
                          style={{
                            width: `${percentage}%`,
                            backgroundColor: color,
                          }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="w-48 h-48 shrink-0 relative flex items-center justify-center">
                <svg
                  viewBox="0 0 42 42"
                  className="w-full h-full -rotate-90 drop-shadow-xl"
                >
                  <circle
                    cx="21"
                    cy="21"
                    r="15.91549431"
                    fill="transparent"
                    stroke="#1e1f22"
                    strokeWidth="6"
                  />

                  {poll.Options?.map((option, index) => {
                    const votesForOption =
                      poll.Votes?.filter((v) => v.OptionID === option.ID)
                        .length || 0;
                    if (votesForOption === 0) return null;

                    const percentage = (votesForOption / totalVotes) * 100;
                    const dashArray = `${percentage} ${100 - percentage}`;
                    const dashOffset = 100 - cumulativePercent;
                    const color = CHART_COLORS[index % CHART_COLORS.length];

                    cumulativePercent += percentage;

                    return (
                      <circle
                        key={option.ID}
                        cx="21"
                        cy="21"
                        r="15.91549431"
                        fill="transparent"
                        stroke={color}
                        strokeWidth="6"
                        strokeDasharray={dashArray}
                        strokeDashoffset={dashOffset}
                        className="transition-all duration-1000 ease-out hover:opacity-80 hover:stroke-[7px] cursor-pointer"
                      >
                        <title>
                          {option.Label}: {votesForOption} votes (
                          {Math.round(percentage)}%)
                        </title>
                      </circle>
                    );
                  })}
                </svg>

                <div className="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
                  <span className="text-xl font-bold">{totalVotes}</span>
                  <span className="text-[10px] text-[#99AAB5] uppercase tracking-widest mt-0.5">
                    Votes
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* --- USER LIST MODAL --- */}
      {userModal && (
        <div
          className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 
        backdrop-blur-sm animate-fade-in"
        >
          <div
            className="bg-[#313338] w-full max-w-md rounded-xl border border-[#1e1f22] shadow-2xl 
          flex flex-col max-h-[80vh]"
          >
            {/* Modal Header */}
            <div className="flex justify-between items-center p-6 border-b border-[#1e1f22]">
              <h2 className="text-lg font-bold text-white">
                {userModal.title}
              </h2>
              <button
                onClick={() => setUserModal(null)}
                className="text-[#99AAB5] hover:text-[#da373c] transition-colors p-1"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M6 18L18 6M6 6l12 12"
                  ></path>
                </svg>
              </button>
            </div>

            {/* Modal List */}
            <div className="p-6 overflow-y-auto custom-scrollbar flex-1 space-y-3">
              {userModal.users.map((u) => (
                <div
                  key={u.id}
                  className="flex items-center gap-3 bg-[#2b2d31] p-3 rounded-lg 
                border border-[#1e1f22] shadow-sm"
                >
                  <img
                    src={
                      u.avatar ||
                      "https://cdn.discordapp.com/embed/avatars/0.png"
                    }
                    onError={(e) => {
                      e.target.onerror = null;
                      e.target.src =
                        "https://cdn.discordapp.com/embed/avatars/0.png";
                    }}
                    alt="avatar"
                    className="w-10 h-10 rounded-full bg-[#1e1f22]"
                  />
                  <div className="flex flex-col flex-1 min-w-0">
                    <span className="font-bold text-sm text-white truncate">
                      {u.username}
                    </span>
                    <span className="text-[10px] text-[#99AAB5] font-mono truncate">
                      ID: {u.id}
                    </span>
                  </div>

                  {/* Show their vote badge if we know what they voted for! */}
                  {u.votedFor && (
                    <span
                      className="text-[10px] bg-[#5865F2]/20 text-[#5865F2] px-2 py-1 rounded font-bold 
                      shrink-0 max-w-30 truncate"
                      title={u.votedFor}
                    >
                      {u.votedFor}
                    </span>
                  )}
                </div>
              ))}

              {userModal.users.length === 0 && (
                <div className="text-center text-[#99AAB5] py-8">
                  No users found in this category.
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Global Style for the modal fade-in */}
      <style
        dangerouslySetInnerHTML={{
          __html: `
        @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to 
        { opacity: 1; transform: translateY(0); } }
        .animate-fade-in { animation: fadeIn 0.2s ease-out forwards; }
      `,
        }}
      />
    </div>
  );
}
