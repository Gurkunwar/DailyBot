import React from "react";

export default function Login() {
  const handleLogin = () => {
    const clientId = import.meta.env.VITE_DISCORD_CLIENT_ID;
    const redirectUri = import.meta.env.VITE_DISCORD_REDIRECT_URI;
    const scope = "identify guilds";

    const discordAuthUrl = `https://discord.com/api/oauth2/authorize?client_id=${clientId}&redirect_uri=${encodeURIComponent(redirectUri)}&response_type=code&scope=${encodeURIComponent(scope)}&prompt=consent`;
    window.location.href = discordAuthUrl;
  };

  return (
    <div
      className="min-h-screen bg-linear-to-br from-[#23272A] to-[#2C2F33] flex 
        items-center justify-center p-4 font-sans text-white"
    >
      <div
        className="bg-[#383e43] w-full max-w-md rounded-2xl p-10 shadow-[0_8px_30px_rgba(0,0,0,0.4)] 
        border border-gray-800 text-center"
      >
        <div
          className="bg-linear-to-tr from-[#5865F2] to-[#7983f5] w-16 h-16 rounded-2xl mx-auto mb-8 
            flex items-center justify-center shadow-lg transform -rotate-6"
        >
          <span className="text-3xl filter drop-shadow-md origin-center rotate-6">
            🤖
          </span>
        </div>

        <h1 className="text-3xl font-bold mb-3 tracking-tight">DailyBot</h1>
        <p className="text-[#99AAB5] mb-10 text-sm leading-relaxed px-4">
          Automate your team's standups. Log in with Discord to access your
          dashboard.
        </p>

        <button
          onClick={handleLogin}
          className="cursor-pointer group w-full bg-[#5865F2] text-white font-semibold py-3.5 px-4 
          rounded-lg flex items-center justify-center gap-3 shadow-md transition-all duration-300 ease-in-out 
          hover:bg-[#4752C4] hover:-translate-y-1 hover:shadow-[0_0_20px_rgba(88,101,242,0.6)]"
        >
          <svg
            className="w-6 h-6 transform transition-transform duration-300 group-hover:scale-110"
            viewBox="0 0 127.14 96.36"
            fill="currentColor"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M107.7,8.07A105.15,105.15,0,0,0,81.47,0a72.06,72.06,0,0,0-3.36,6.83A97.68,97.68,0,0,
            0,49,6.83,72.37,72.37,0,0,0,45.64,0,105.89,105.89,0,0,0,19.39,8.09C2.79,32.65-1.71,56.6.54,
            80.21h0A105.73,105.73,0,0,0,32.71,96.36,77.7,77.7,0,0,0,39.6,85.25a68.42,68.42,0,0,1-10.85-5.18c.91-.66,
            1.8-1.34,2.66-2a75.57,75.57,0,0,0,64.32,0c.87.71,1.76,1.39,2.66,2a68.68,68.68,0,0,1-10.87,5.19,77.7,
            77.7,0,0,0,6.89,11.1,105.25,105.25,0,0,0,32.19-16.14c0,0,.04-.06.09-.09C129.67,52.82,122.93,28.21,107.7,
            8.07ZM42.45,65.69C36.18,65.69,31,60,31,53s5-12.74,11.43-12.74S54,46,53.89,53,48.84,65.69,42.45,65.69Zm42.24,
            0C78.41,65.69,73.31,60,73.31,53s5-12.74,11.43-12.74S96.3,46,96.19,53,91.08,65.69,84.69,65.69Z"
            />
          </svg>
          Login with Discord
        </button>
      </div>
    </div>
  );
}
