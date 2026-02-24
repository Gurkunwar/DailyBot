import React from "react";
import { useNavigate } from "react-router-dom";
import Navbar from "../components/Navbar";

export default function Home() {
  const navigate = useNavigate();

  return (
    <div className="min-h-screen bg-[#1e1f22] font-sans text-white">
      <Navbar />

      {/* Hero Section */}
      <main className="max-w-7xl mx-auto px-8 py-24 flex flex-col items-center text-center">
        <div className="inline-block bg-[#2b2d31] text-[#5865F2] font-semibold px-4 py-1.5 rounded-full 
        text-sm mb-6 border border-[#1e1f22] shadow-sm">
          🚀 The ultimate Discord standup bot
        </div>

        <h1 className="text-5xl md:text-7xl font-extrabold tracking-tight mb-8 text-transparent bg-clip-text 
        bg-linear-to-r from-white to-[#99AAB5]">
          Automate your team's <br className="hidden md:block" /> daily
          standups.
        </h1>

        <p className="text-lg md:text-xl text-[#99AAB5] max-w-2xl mb-12 leading-relaxed">
          DailyBot lives directly in your Discord server. Collect asynchronous
          updates, track team progress, and view histories all from one
          beautiful dashboard.
        </p>

        <button
          onClick={() => navigate("/login")}
          className="group bg-[#5865F2] hover:bg-[#4752C4] text-white font-semibold py-4 px-8 rounded-lg 
          text-lg flex items-center gap-3 transition-all duration-300 shadow-[0_0_20px_rgba(88,101,242,0.3)] 
          hover:shadow-[0_0_30px_rgba(88,101,242,0.6)] hover:-translate-y-1 cursor-pointer"
        >
          <svg
            className="w-6 h-6 transform transition-transform duration-300 group-hover:scale-110"
            viewBox="0 0 127.14 96.36"
            fill="currentColor"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M107.7,8.07A105.15,105.15,0,0,0,81.47,0a72.06,72.06,0,0,0-3.36,6.83A97.68,97.68,0,0,0,49,
            6.83,72.37,72.37,0,0,0,45.64,0,105.89,105.89,0,0,0,19.39,8.09C2.79,32.65-1.71,56.6.54,80.21h0A105.73,
            105.73,0,0,0,32.71,96.36,77.7,77.7,0,0,0,39.6,85.25a68.42,68.42,0,0,1-10.85-5.18c.91-.66,1.8-1.34,
            2.66-2a75.57,75.57,0,0,0,64.32,0c.87.71,1.76,1.39,2.66,2a68.68,68.68,0,0,1-10.87,5.19,77.7,77.7,0,0,
            0,6.89,11.1,105.25,105.25,0,0,0,32.19-16.14c0,0,.04-.06.09-.09C129.67,52.82,122.93,28.21,107.7,
            8.07ZM42.45,65.69C36.18,65.69,31,60,31,53s5-12.74,11.43-12.74S54,46,53.89,53,48.84,65.69,42.45,
            65.69Zm42.24,0C78.41,65.69,73.31,60,73.31,53s5-12.74,11.43-12.74S96.3,46,96.19,53,91.08,65.69,
            84.69,65.69Z"
            />
          </svg>
          Get Started with Discord
        </button>
      </main>
    </div>
  );
}
