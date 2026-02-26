import React, { useState, useEffect } from "react";

export default function HistoryTab({ standup, guildMembers }) {
  const [histories, setHistories] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [selectedUser, setSelectedUser] = useState("ALL");
  const [selectedDate, setSelectedDate] = useState("");

  useEffect(() => {
    const sID = standup?.id || standup?.ID;

    const fetchHistory = async () => {
      setIsLoading(true);
      const token = localStorage.getItem("token");
      try {
        const res = await fetch(`http://localhost:8080/api/standups/history?standup_id=${sID}`, {
          headers: { Authorization: `Bearer ${token}` }
        });
        if (res.ok) {
          const data = await res.json();
          const validHistory = (data || []).filter(h => h.answers && h.answers.length > 0);
          setHistories(validHistory);
        }
      } catch (err) {
        console.error("Failed to load history", err);
      }
      setIsLoading(false);
    };

    if (sID) fetchHistory();
  }, [standup]);

  const getUserInfo = (userId) => {
    const member = guildMembers.find(m => m.id === userId);
    if (member) return member;
    return { username: "Unknown User", avatar: null };
  };

  const filteredHistories = histories.filter(h => {
    const matchUser = selectedUser === "ALL" || h.user_id === selectedUser;
    const matchDate = selectedDate === "" || h.date === selectedDate;
    return matchUser && matchDate;
  });

  return (
    <div className="animate-fade-in flex flex-col h-full">
      <div className="mb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-white">Report History</h2>
          <p className="text-[#99AAB5] text-sm mt-1">
            Review past daily updates from your team.
          </p>
        </div>

        {/* FILTERS */}
        <div className="flex items-center gap-3">
          <input 
            type="date" 
            value={selectedDate}
            onChange={(e) => setSelectedDate(e.target.value)}
            className="bg-[#1e1f22] p-2 rounded-md border border-transparent focus:border-[#5865F2] 
            outline-none text-white text-sm cursor-pointer shadow-inner"
          />
          <select 
            value={selectedUser}
            onChange={(e) => setSelectedUser(e.target.value)}
            className="bg-[#1e1f22] p-2 rounded-md border border-transparent focus:border-[#5865F2] 
            outline-none text-white text-sm cursor-pointer shadow-inner min-w-37.5"
          >
            <option value="ALL">All Members</option>
            {standup?.participants?.map(p => {
              const uID = p.user_id || p.UserID;
              const user = getUserInfo(uID);
              return <option key={uID} value={uID}>{user.username}</option>
            })}
          </select>
        </div>
      </div>

      <div className="space-y-4">
        {isLoading ? (
          <div className="bg-[#2b2d31] p-8 rounded-xl border border-[#1e1f22] text-center text-[#99AAB5]">
            <span className="animate-pulse">Loading reports...</span>
          </div>
        ) : filteredHistories.length === 0 ? (
          <div className="bg-[#2b2d31] p-10 rounded-xl border border-[#1e1f22] text-center shadow-sm">
            <div className="text-4xl mb-3">📭</div>
            <h3 className="text-white font-bold text-lg mb-1">No reports found</h3>
            <p className="text-[#99AAB5] text-sm">No standup submissions match your current filters.</p>
          </div>
        ) : (
          filteredHistories.map((log, index) => {
            const user = getUserInfo(log.user_id);
            return (
              <div key={log.ID || index} className="bg-[#2b2d31] rounded-xl border border-[#1e1f22] 
              overflow-hidden shadow-sm hover:border-[#3f4147] transition-colors">
                
                {/* Header: User Info & Date */}
                <div className="bg-[#232428] px-5 py-3 border-b border-[#1e1f22] flex justify-between 
                items-center">
                  <div className="flex items-center gap-3">
                    {user.avatar ? (
                      <img src={user.avatar} alt="avatar" className="w-8 h-8 rounded-full" />
                    ) : (
                      <div className="w-8 h-8 rounded-full bg-[#5865F2] flex items-center justify-center 
                      font-bold text-xs text-white">
                        {user.username.charAt(0).toUpperCase()}
                      </div>
                    )}
                    <span className="font-bold text-white text-sm">{user.username}</span>
                  </div>
                  <div className="text-xs font-bold text-[#99AAB5] bg-[#1e1f22] px-2.5 py-1 rounded">
                    {log.date}
                  </div>
                </div>

                {/* Body: Questions and Answers */}
                <div className="p-5 space-y-4">
                  {log.answers.map((answer, i) => {
                    const question = standup?.questions?.[i] || standup?.Questions?.[i] || `Question ${i + 1}`;
                    return (
                      <div key={i}>
                        <h4 className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-1.5">
                          {question}
                        </h4>
                        <p className="text-white text-sm leading-relaxed bg-[#1e1f22] p-3 rounded-md border 
                        border-[#3f4147]/30 wrap-break-word whitespace-pre-wrap">
                          {answer}
                        </p>
                      </div>
                    );
                  })}
                </div>

              </div>
            );
          })
        )}
      </div>
    </div>
  );
}