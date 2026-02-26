import React, { useState } from "react";

export default function MembersTab({ standup, guildMembers, isLoading, onToggleMember }) {
  const [searchQuery, setSearchQuery] = useState("");

  const filteredMembers = guildMembers.filter((m) =>
    m.username.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="animate-fade-in">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-bold text-white">Team Roster</h2>
          <p className="text-[#99AAB5] text-sm mt-1">
            Manage who receives daily standup prompts via Discord.
          </p>
        </div>
        <div className="bg-[#2b2d31] px-3 py-1.5 rounded-md text-sm font-bold border border-[#1e1f22] text-[#5865F2]">
          {standup?.participants?.length || 0} Enrolled
        </div>
      </div>

      <div className="relative mb-4">
        <svg className="w-5 h-5 absolute left-3 top-3 text-[#99AAB5]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
        </svg>
        <input
          type="text"
          placeholder="Search server members..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full bg-[#1e1f22] pl-10 pr-4 py-3 rounded-lg border border-transparent focus:border-[#5865F2] outline-none text-sm transition-colors shadow-inner"
        />
      </div>

      <div className="bg-[#2b2d31] rounded-lg border border-[#1e1f22] shadow-sm overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-[#99AAB5]">Loading members...</div>
        ) : filteredMembers.length === 0 ? (
          <div className="p-8 text-center text-[#99AAB5]">No members found matching "{searchQuery}"</div>
        ) : (
          <div className="max-h-125 overflow-y-auto custom-scrollbar divide-y divide-[#1e1f22]">
            {filteredMembers.map((member) => {
              const isEnrolled = standup?.participants?.some(
                (p) => p.user_id === member.id || p.UserID === member.id
              );

              return (
                <div key={member.id} className="flex items-center justify-between p-4 hover:bg-[#313338] transition-colors group">
                  <div className="flex items-center gap-4">
                    {member.avatar ? (
                      <img src={member.avatar} alt="avatar" className="w-10 h-10 rounded-full shadow-md" />
                    ) : (
                      <div className="w-10 h-10 rounded-full bg-[#5865F2] flex items-center justify-center font-bold shadow-md">
                        {member.username.charAt(0).toUpperCase()}
                      </div>
                    )}
                    <span className="font-semibold text-[15px]">{member.username}</span>
                  </div>

                  <button
                    onClick={() => onToggleMember(member.id, isEnrolled)}
                    className={`px-5 py-1.5 rounded-md text-sm font-bold transition-all transform active:scale-95 border ${
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
  );
}