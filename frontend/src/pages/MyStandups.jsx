import React, { useEffect, useState, useCallback } from "react";
import Sidebar from "../components/Sidebar";
import CreateStandupModal from "../components/CreateStandupModel";
import { useNavigate } from "react-router-dom";

export default function MyStandups() {
  const [standups, setStandups] = useState([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedGuild, setSelectedGuild] = useState("All");
  const token = localStorage.getItem("token");
  const navigate = useNavigate();

  const fetchStandups = useCallback(async () => {
    try {
      const API_BASE = import.meta.env.VITE_API_BASE_URL;
      const response = await fetch(`${API_BASE}/managed-standups`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      const data = await response.json();
      setStandups(data || []);
    } catch (error) {
      console.error("Failed to load teams:", error);
    }
  }, [token]);

  useEffect(() => {
    fetchStandups();
  }, [fetchStandups]);

  const uniqueGuilds = ["All", ...new Set(standups.map((s) => s.guild_name))];

  const filteredStandups =
    selectedGuild === "All"
      ? standups
      : standups.filter((s) => s.guild_name === selectedGuild);

  return (
    <div className="flex h-screen bg-[#313338] text-white overflow-hidden font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header className="h-14 border-b border-[#1e1f22] flex items-center px-8 shadow-sm">
          <h1 className="text-lg font-bold">My Standups</h1>
        </header>

        <div className="flex-1 overflow-y-auto p-8 relative">
          <div className="flex justify-between items-center mb-8">
            <h2 className="text-2xl font-bold">Managed Standups</h2>
            
            {/* 4. Added the filter dropdown UI next to the button */}
            <div className="flex items-center gap-4">
              {standups.length > 0 && (
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
                className="bg-[#5865F2] hover:bg-[#4752C4] px-4 py-2 rounded font-semibold text-sm 
              transition-colors cursor-pointer"
              >
                + New Standup
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {/* 5. Map over filteredStandups instead of the raw standups array */}
            {filteredStandups.map((s) => (
              <div
                key={s.id}
                onClick={() => navigate(`/standups/${s.id}`)}
                className="bg-[#2b2d31] p-6 rounded-xl border border-[#1e1f22] hover:border-[#5865F2] 
                transition-all cursor-pointer group shadow-lg"
              >
                <div className="flex justify-between items-start mb-1">
                  <div>
                    <h3 className="text-xl font-bold group-hover:text-[#5865F2] transition-colors">
                      {s.name}
                    </h3>
                    <p className="text-[10px] text-[#5865F2] font-bold uppercase tracking-widest mb-4">
                      {s.guild_name}
                    </p>
                  </div>
                  <span className="text-[10px] bg-[#232428] px-2 py-1 rounded text-[#99AAB5] font-mono">
                    ID: {s.id}
                  </span>
                </div>

                <div className="space-y-3 text-sm pt-4 border-t border-[#3f4147]">
                  <div className="flex items-center justify-between">
                    <span className="text-[#99AAB5]">Schedule</span>
                    <span className="text-white bg-[#313338] px-2 py-1 rounded-md text-xs font-semibold">
                      🕒 {s.time}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-[#99AAB5]">Channel</span>
                    <span className="text-[#43b581] font-bold flex items-center gap-1">
                      <span className="text-lg">#</span> {s.channel_name}
                    </span>
                  </div>
                </div>
              </div>
            ))}
            
            {/* Display empty states conditionally */}
            {standups.length === 0 && (
              <div
                className="col-span-full py-20 text-center bg-[#2b2d31] rounded-xl border-2 border-dashed 
              border-[#404249]"
              >
                <p className="text-[#99AAB5]">
                  No standups found. Create your first one to get started!
                </p>
              </div>
            )}

            {standups.length > 0 && filteredStandups.length === 0 && (
               <div
               className="col-span-full py-20 text-center bg-[#2b2d31] rounded-xl border-2 border-dashed 
             border-[#404249]"
             >
               <p className="text-[#99AAB5]">
                 No standups found in this server.
               </p>
             </div>
            )}
          </div>
        </div>
      </main>

      <CreateStandupModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onRefresh={fetchStandups}
      />
    </div>
  );
}