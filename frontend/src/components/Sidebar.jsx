import React from "react";
import { NavLink, useNavigate } from "react-router-dom";

function NavItem({ to, icon, label }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-3 px-3 py-2 rounded-md transition-colors duration-200 ${
          isActive
            ? "bg-[#404249] text-white"
            : "text-[#99AAB5] hover:bg-[#35373c] hover:text-gray-200"
        }`
      }
    >
      <span className="text-lg">{icon}</span>
      <span className="font-medium text-sm">{label}</span>
    </NavLink>
  );
}

export default function Sidebar() {
  const navigate = useNavigate();
  const userData = JSON.parse(localStorage.getItem("user") || "{}");

  const handleLogout = () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    navigate("/");
  };

  const avatarUrl = userData.avatar
    ? `https://cdn.discordapp.com/avatars/${userData.id}/${userData.avatar}.png`
    : `https://cdn.discordapp.com/embed/avatars/0.png`;

  return (
    <aside className="w-64 bg-[#2b2d31] flex flex-col justify-between md:flex border-r border-[#1e1f22]">
      {/* Top: Branding & Navigation */}
      <div>
        <div
          className="h-14 flex items-center px-6 border-b border-[#1e1f22] shadow-sm font-bold text-lg 
        tracking-wide"
        >
          <span className="mr-3 text-xl">🤖</span> DailyBot
        </div>

        <nav className="p-3 space-y-1 mt-2">
          <NavItem to="/dashboard" icon="📊" label="Overview" />
          <NavItem to="/standups" icon="👥" label="My Standups" />
          <NavItem to="/history" icon="📜" label="History" />
          <NavItem to="/settings" icon="⚙️" label="Settings" />
        </nav>
      </div>

      {/* Bottom: User Profile & Logout */}
      <div className="bg-[#232428] p-3 flex items-center justify-between mt-auto">
        <div className="flex items-center gap-3 overflow-hidden">
          <img
            src={avatarUrl}
            alt="Avatar"
            className="w-9 h-9 rounded-full shrink-0 border border-gray-700"
          />
          <div className="flex flex-col truncate">
            <span className="text-sm font-bold truncate text-white">
              {userData.username || "Guest"}
            </span>
            <span className="text-xs text-[#99AAB5] truncate">Manager</span>
          </div>
        </div>

        <button
          onClick={handleLogout}
          className="text-[#99AAB5] hover:text-[#da373c] p-2 rounded-md hover:bg-[#313338] 
          transition-colors cursor-pointer"
          title="Logout"
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
              strokeWidth={2}
              d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
            />
          </svg>
        </button>
      </div>
    </aside>
  );
}
