import React, { useEffect, useState } from "react";
import Sidebar from "../components/Sidebar"; // IMPORT THE COMPONENT

// Helper component for the Dashboard Cards
function StatCard({ title, value, subtitle, textClass = "text-white" }) {
  return (
    <div className="bg-[#2b2d31] p-5 rounded-lg border border-[#1e1f22] shadow-sm">
      <h3 className="text-[#99AAB5] text-xs font-bold uppercase tracking-wider mb-2">
        {title}
      </h3>
      <div className={`text-3xl font-extrabold mb-1 ${textClass}`}>{value}</div>
      <p className="text-[#99AAB5] text-xs">{subtitle}</p>
    </div>
  );
}

export default function Dashboard() {
  const [standups, setStandups] = useState([]);
  const token = localStorage.getItem("token");

  useEffect(() => {
    const fetchTeams = async () => {
      try {
        const response = await fetch(
          "http://localhost:8080/api/managed-standups",
          {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          },
        );
        const data = await response.json();
        setStandups(data);
      } catch (error) {
        console.error("Failed to load teams:", error);
      }
    };

    fetchTeams();
  }, [token]);

  return (
    <div className="flex h-screen bg-[#313338] font-sans text-white overflow-hidden">
      <Sidebar /> {/* USE IT HERE */}
      {/* MAIN CONTENT AREA */}
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header className="h-14 bg-[#313338] border-b border-[#1e1f22] flex items-center px-8 shadow-sm">
          <h1 className="text-lg font-bold text-white flex items-center gap-2">
            <span className="text-[#99AAB5]">#</span> Overview
          </h1>
        </header>

        <div className="flex-1 overflow-y-auto p-8 bg-[#313338]">
          <h2 className="text-2xl font-bold mb-6">Welcome back!</h2>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <StatCard
              title="Active Standups"
              value={standups.length.toString()}
              subtitle="Managing 12 members"
            />
            <StatCard
              title="Reports Today"
              value="8"
              subtitle="4 pending submissions"
            />
            <StatCard
              title="Completion Rate"
              value="85%"
              subtitle="+5% from last week"
              textClass="text-green-400"
            />
          </div>

          <div className="bg-[#2b2d31] rounded-lg border border-[#1e1f22] p-6">
            <h3 className="text-lg font-bold mb-4 text-white">
              Recent Activity
            </h3>
            <div
              className="text-[#99AAB5] text-sm flex items-center justify-center h-32 border-2 border-dashed 
            border-[#404249] rounded-lg"
            >
              No recent standup reports to display.
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
