import React, { useState } from "react";
import Sidebar from "../components/Sidebar";
import { useNavigate } from "react-router-dom";
import { useGetManagedPollsQuery } from "../store/apiSlice";
import CreatePollModal from "../components/CreatePollModal";

export default function MyPolls() {
  const navigate = useNavigate();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedGuild, setSelectedGuild] = useState("All");

  // RTK Query handles all the loading, caching, and fetching automatically!
  const { data: polls = [], isLoading } = useGetManagedPollsQuery();

  // Extract unique guild names to populate the dropdown
  const uniqueGuilds = ["All", ...new Set(polls.map((p) => p.guild_name))];

  // Filter the polls based on the selected dropdown value
  const filteredPolls =
    selectedGuild === "All"
      ? polls
      : polls.filter((p) => p.guild_name === selectedGuild);

  return (
    <div className="flex h-screen bg-[#313338] text-white overflow-hidden font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header className="h-14 border-b border-[#1e1f22] flex items-center px-8 shadow-sm">
          <h1 className="text-lg font-bold">My Polls</h1>
        </header>

        <div className="flex-1 overflow-y-auto p-8 relative">
          <div className="flex justify-between items-center mb-8">
            <h2 className="text-2xl font-bold">Managed Polls</h2>
            
            <div className="flex items-center gap-4">
              {polls.length > 0 && (
                <div className="flex items-center gap-2">
                  <span className="text-sm font-bold text-[#99AAB5]">Filter:</span>
                  <select
                    value={selectedGuild}
                    onChange={(e) => setSelectedGuild(e.target.value)}
                    className="bg-[#1e1f22] text-sm text-white px-3 py-2 rounded-md outline-none border border-transparent focus:border-[#5865F2] cursor-pointer"
                  >
                    {uniqueGuilds.map((guild) => (
                      <option key={guild} value={guild}>
                        {guild === "All" ? "All Servers" : guild}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              <button
                onClick={() => setIsModalOpen(true)}
                className="bg-[#5865F2] hover:bg-[#4752C4] px-4 py-2 rounded font-semibold text-sm transition-colors cursor-pointer shadow-md"
              >
                + New Poll
              </button>
            </div>
          </div>

          {isLoading ? (
            <div className="flex justify-center items-center h-64 text-[#99AAB5]">
              Loading polls...
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {filteredPolls.map((p) => (
                <div
                  key={p.id}
                  onClick={() => navigate(`/polls/${p.id}`)} // You can build a ManagePoll.jsx page later!
                  className="bg-[#2b2d31] p-6 rounded-xl border border-[#1e1f22] hover:border-[#5865F2] transition-all cursor-pointer group shadow-lg"
                >
                  <div className="flex justify-between items-start mb-1">
                    <div>
                      <h3 className="text-xl font-bold group-hover:text-[#5865F2] transition-colors line-clamp-2">
                        {p.question}
                      </h3>
                      <p className="text-[10px] text-[#5865F2] font-bold uppercase tracking-widest mb-4 mt-1">
                        {p.guild_name}
                      </p>
                    </div>
                    <span className="text-[10px] bg-[#232428] px-2 py-1 rounded text-[#99AAB5] font-mono shrink-0 ml-2">
                      ID: {p.id}
                    </span>
                  </div>

                  <div className="space-y-3 text-sm pt-4 border-t border-[#3f4147]">
                    <div className="flex items-center justify-between">
                      <span className="text-[#99AAB5]">Status</span>
                      <span className={`px-2 py-1 rounded-md text-xs font-semibold ${p.is_active ? "bg-[#23a559]/10 text-[#23a559]" : "bg-[#404249] text-[#99AAB5]"}`}>
                        {p.is_active ? "🟢 Active" : "⚪ Closed"}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-[#99AAB5]">Channel</span>
                      <span className="text-[#43b581] font-bold flex items-center gap-1">
                        <span className="text-lg">#</span> {p.channel_name}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
              
              {polls.length === 0 && (
                <div className="col-span-full py-20 text-center bg-[#2b2d31] rounded-xl border-2 border-dashed border-[#404249]">
                  <p className="text-[#99AAB5] mb-2 text-4xl">🗳️</p>
                  <p className="text-[#99AAB5]">No polls found. Type `/poll` in Discord to create your first one!</p>
                </div>
              )}

              {polls.length > 0 && filteredPolls.length === 0 && (
                <div className="col-span-full py-20 text-center bg-[#2b2d31] rounded-xl border-2 border-dashed border-[#404249]">
                  <p className="text-[#99AAB5]">No polls found in this specific server.</p>
                </div>
              )}
            </div>
          )}
        </div>
      </main>

      <CreatePollModal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)} />
    </div>
  );
}