import React from "react";
import { useNavigate } from "react-router-dom";

export default function Navbar() {
  const navigate = useNavigate();

  return (
    <nav className="flex items-center justify-between px-8 py-5 max-w-7xl mx-auto border-b border-[#313338]">
      <div className="flex items-center gap-3">
        <span className="text-2xl">🤖</span>
        <span className="text-xl font-bold tracking-tight">DailyBot</span>
      </div>

      <div className="flex items-center gap-6">
        <a
          href="#features"
          className="text-[#99AAB5] hover:text-white transition-colors text-sm font-semibold hidden md:block"
        >
          Features
        </a>
        <button
          onClick={() => navigate('/login')}
          className="bg-[#5865F2] hover:bg-[#4752C4] text-white text-sm font-semibold py-2 px-5 
              rounded-full transition-colors duration-200 cursor-pointer shadow-md"
        >
          Login
        </button>
      </div>
    </nav>
  );
}