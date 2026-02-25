import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Sidebar from "../components/Sidebar";

export default function ManageStandup() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("members");

  const [standup, setStandup] = useState(null);
  const [guildMembers, setGuildMembers] = useState([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  const token = localStorage.getItem("token");

  // 1. Fetch the specific Standup details (including enrolled participants)
  const fetchStandupData = async () => {
    try {
      const res = await fetch(
        `http://localhost:8080/api/standups/get?id=${id}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        const data = await res.json();
        console.log("Here are the participants from Go:", data.participants);
        setStandup(data);
        return data.guild_id; // Return guild_id to fetch members next
      }
    } catch (err) {
      console.error("Failed to load standup", err);
    }
    return null;
  };

  // 2. Fetch the Discord Server Members
  const fetchGuildMembers = async (guildId) => {
    try {
      const res = await fetch(
        `http://localhost:8080/api/guild-members?guild_id=${guildId}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );
      if (res.ok) {
        const data = await res.json();
        setGuildMembers(data || []);
      }
    } catch (err) {
      console.error("Failed to load members", err);
    }
  };

  // Initialize Data
  useEffect(() => {
    setIsLoading(true);
    fetchStandupData().then((guildId) => {
      if (guildId) fetchGuildMembers(guildId);
      setIsLoading(false);
    });
  }, [id, token]);

  // 3. Handle Add/Remove Logic
  const toggleMember = async (userId, isCurrentlyMember) => {
    const endpoint = isCurrentlyMember ? "remove-member" : "add-member";
    try {
      const res = await fetch(
        `http://localhost:8080/api/standups/${endpoint}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({ standup_id: parseInt(id), user_id: userId }),
        },
      );

      if (res.ok) {
        // Optimistically update the UI so it feels instant
        fetchStandupData();
      }
    } catch (err) {
      console.error("Failed to toggle member", err);
    }
  };

  const tabs = [
    { id: "members", label: "👥 Members" },
    { id: "settings", label: "⚙️ Settings" },
    { id: "history", label: "📜 History" },
  ];

  // Filtering members based on the search bar
  const filteredMembers = guildMembers.filter((m) =>
    m.username.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  return (
    <div className="flex h-screen bg-[#313338] text-white font-sans overflow-hidden">
      <Sidebar />

      <main className="flex-1 flex flex-col min-w-0">
        <header className="h-14 border-b border-[#1e1f22] flex items-center px-6 shadow-sm gap-4">
          <button
            onClick={() => navigate("/standups")}
            className="text-[#99AAB5] hover:text-white transition-colors flex items-center gap-1 
            text-sm font-semibold"
          >
            ← Back
          </button>
          <div className="h-4 w-px bg-[#3f4147]"></div>
          <h1 className="text-lg font-bold truncate">
            Manage: {standup?.name || "Loading..."}
          </h1>
        </header>

        <div className="flex flex-1 overflow-hidden">
          {/* Sub-Navigation Sidebar */}
          <div className="w-56 bg-[#2b2d31] border-r border-[#1e1f22] flex flex-col p-4 gap-1">
            <div className="text-xs font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 px-2">
              Configuration
            </div>
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`text-left px-3 py-2 rounded-md text-sm font-semibold transition-colors ${
                  activeTab === tab.id
                    ? "bg-[#404249] text-white"
                    : "text-[#99AAB5] hover:bg-[#35373c] hover:text-white"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Main Content Area */}
          <div className="flex-1 overflow-y-auto p-8 bg-[#313338] custom-scrollbar">
            <div className="max-w-3xl mx-auto">
              {activeTab === "members" && (
                <div className="animate-fade-in">
                  <div className="flex items-center justify-between mb-6">
                    <div>
                      <h2 className="text-2xl font-bold text-white">
                        Team Members
                      </h2>
                      <p className="text-[#99AAB5] text-sm mt-1">
                        Manage who receives daily standup prompts.
                      </p>
                    </div>
                    <div className="bg-[#2b2d31] px-3 py-1 rounded-md text-sm font-mono border border-[#1e1f22]">
                      {standup?.participants?.length || 0} Enrolled
                    </div>
                  </div>

                  {/* Search Bar */}
                  <div className="relative mb-4">
                    <svg
                      className="w-5 h-5 absolute left-3 top-3 text-[#99AAB5]"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth="2"
                        d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                      ></path>
                    </svg>
                    <input
                      type="text"
                      placeholder="Search server members..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="w-full bg-[#1e1f22] pl-10 pr-4 py-3 rounded-lg border border-transparent 
                      focus:border-[#5865F2] outline-none text-sm transition-colors"
                    />
                  </div>

                  {/* Member List */}
                  <div className="bg-[#2b2d31] rounded-lg border border-[#1e1f22] shadow-sm overflow-hidden">
                    {isLoading ? (
                      <div className="p-8 text-center text-[#99AAB5]">
                        Loading members...
                      </div>
                    ) : filteredMembers.length === 0 ? (
                      <div className="p-8 text-center text-[#99AAB5]">
                        No members found matching "{searchQuery}"
                      </div>
                    ) : (
                      <div className="max-h-125 overflow-y-auto custom-scrollbar divide-y divide-[#1e1f22]">
                        {filteredMembers.map((member) => {
                          // Check if the current member is already in the database Participants array
                          const isEnrolled = standup?.participants?.some(
                            (p) => p.user_id === member.id,
                          );

                          return (
                            <div
                              key={member.id}
                              className="flex items-center justify-between p-4 hover:bg-[#313338] transition-colors"
                            >
                              <div className="flex items-center gap-4">
                                {member.avatar ? (
                                  <img
                                    src={member.avatar}
                                    alt="avatar"
                                    className="w-10 h-10 rounded-full"
                                  />
                                ) : (
                                  <div className="w-10 h-10 rounded-full bg-[#5865F2] flex items-center 
                                  justify-center font-bold">
                                    {member.username.charAt(0).toUpperCase()}
                                  </div>
                                )}
                                <span className="font-semibold text-[15px]">
                                  {member.username}
                                </span>
                              </div>

                              <button
                                onClick={() =>
                                  toggleMember(member.id, isEnrolled)
                                }
                                className={`px-4 py-1.5 rounded-md text-sm font-bold transition-all transform active:scale-95 border ${
                                  isEnrolled
                                    ? "bg-transparent border-[#da373c] text-[#da373c] hover:bg-[#da373c] hover:text-white"
                                    : "bg-[#23a559] border-[#23a559] text-white hover:bg-[#1d8a4a]"
                                }`}
                              >
                                {isEnrolled ? "Remove" : "Add"}
                              </button>
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {activeTab === "settings" && (
                <div className="animate-fade-in">
                  <h2 className="text-2xl font-bold mb-6">Standup Settings</h2>
                  <div className="bg-[#2b2d31] p-8 rounded-xl border border-[#1e1f22] text-center text-[#99AAB5]">
                    Settings form coming next!
                  </div>
                </div>
              )}

              {activeTab === "history" && (
                <div className="animate-fade-in">
                  <h2 className="text-2xl font-bold mb-6">Report History</h2>
                  <div className="bg-[#2b2d31] p-8 rounded-xl border border-[#1e1f22] text-center text-[#99AAB5]">
                    Past daily standup submissions will go here!
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </main>

      {/* Global CSS for scrollbars and animations */}
      <style
        dangerouslySetInnerHTML={{
          __html: `
        .custom-scrollbar::-webkit-scrollbar { width: 6px; }
        .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
        .custom-scrollbar::-webkit-scrollbar-thumb { background-color: #1e1f22; border-radius: 10px; }
        .custom-scrollbar:hover::-webkit-scrollbar-thumb { background-color: #404249; }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(5px); } to { opacity: 1; transform: translateY(0); } }
        .animate-fade-in { animation: fadeIn 0.2s ease-out forwards; }
      `,
        }}
      />
    </div>
  );
}
