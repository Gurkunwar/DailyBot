import React, { useState } from "react"; // Notice: useEffect is gone!
import { useParams, useNavigate } from "react-router-dom";
import Sidebar from "../components/Sidebar";
import MembersTab from "./tabs/MembersTab";
import SettingsTab from "./tabs/SettingsTab";
import HistoryTab from "./tabs/HistoryTab";

import {
  useGetStandupByIdQuery,
  useGetGuildMembersQuery,
  useGetGuildChannelsQuery,
  useToggleMemberMutation,
  useUpdateStandupMutation,
} from "../store/apiSlice";

export default function ManageStandup() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState("members");
  const { data: standup, isLoading: isStandupLoading } =
    useGetStandupByIdQuery(id);
  const skipSecondaryFetches = !standup?.guild_id;

  const { data: guildMembers = [], isLoading: isMembersLoading } =
    useGetGuildMembersQuery(standup?.guild_id, { skip: skipSecondaryFetches });

  const { data: channels = [] } = useGetGuildChannelsQuery(standup?.guild_id, {
    skip: skipSecondaryFetches,
  });

  const isLoading = isStandupLoading || isMembersLoading;

  const [toggleMemberMutation] = useToggleMemberMutation();
  const [updateStandupMutation, { isLoading: isSaving }] =
    useUpdateStandupMutation();

  const toggleMember = async (userId, isCurrentlyMember) => {
    try {
      await toggleMemberMutation({
        standupId: id,
        userId,
        isCurrentlyMember,
      }).unwrap();
    } catch (err) {
      console.error("Failed to toggle member", err);
    }
  };

  const updateStandup = async (updatedData) => {
    try {
      await updateStandupMutation({
        id: parseInt(id),
        ...updatedData,
      }).unwrap();

      alert("Settings saved successfully!");
    } catch (err) {
      console.error("Failed to update standup", err);
      alert("Failed to save settings.");
    }
  };

  const tabs = [
    { id: "members", label: "👥 Members" },
    { id: "settings", label: "⚙️ Settings" },
    { id: "history", label: "📜 History" },
  ];

  return (
    <div className="flex h-screen bg-[#313338] text-white font-sans overflow-hidden">
      <Sidebar />

      <main className="flex-1 flex flex-col min-w-0">
        <div className="bg-[#313338] border-b border-[#1e1f22] pt-6 px-8 shrink-0">
          <div className="flex items-center gap-4 mb-6">
            <button
              onClick={() => navigate("/standups")}
              className="text-[#99AAB5] hover:text-white transition-colors flex items-center gap-1 text-sm 
              font-semibold bg-[#2b2d31] px-3 py-1.5 rounded-md border border-[#1e1f22]"
            >
              ← Back
            </button>
            <h1 className="text-2xl font-extrabold truncate">
              {standup?.name || "Loading..."}
            </h1>
          </div>

          <div className="flex gap-6 mt-2">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`pb-3 text-sm font-semibold transition-colors relative ${
                  activeTab === tab.id
                    ? "text-white"
                    : "text-[#99AAB5] hover:text-[#dcddde]"
                }`}
              >
                {tab.label}
                {activeTab === tab.id && (
                  <div className="absolute bottom-0 left-0 right-0 h-1 bg-[#5865F2] rounded-t-md" />
                )}
              </button>
            ))}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-8 custom-scrollbar">
          <div className="max-w-3xl mx-auto">
            {/* RENDER THE ACTIVE COMPONENT HERE */}
            {activeTab === "members" && (
              <MembersTab
                standup={standup}
                guildMembers={guildMembers}
                isLoading={isLoading}
                onToggleMember={toggleMember}
              />
            )}

            {activeTab === "settings" && (
              <SettingsTab
                standup={standup}
                channels={channels}
                onSave={updateStandup}
                isSaving={isSaving}
              />
            )}

            {activeTab === "history" && (
              <HistoryTab standup={standup} guildMembers={guildMembers} />
            )}
          </div>
        </div>
      </main>

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
