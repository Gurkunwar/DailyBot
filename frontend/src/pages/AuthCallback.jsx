import React, { useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";

export default function AuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const code = searchParams.get("code");

  useEffect(() => {
    if (code) {
      console.log("Success! Discord gave us Auth Code:", code);
      // TODO: Send this code to our Go Backend!

      setTimeout(() => {
        navigate("/dashboard");
      }, 2000);
    }
  }, [code, navigate]);

  return (
    <div
      className="min-h-screen bg-linear-to-br from-[#23272A] to-[#2C2F33] flex items-center justify-center 
    p-4 font-sans text-white"
    >
      <div
        className="bg-[#383e43] w-full max-w-sm rounded-2xl p-10 shadow-[0_8px_30px_rgba(0,0,0,0.4)] border 
      border-gray-800 text-center flex flex-col items-center"
      >
        {/* Spinning Loading Ring */}
        <div className="relative w-20 h-20 mb-8">
          <div className="absolute inset-0 rounded-full border-4 border-[#2b2d31]"></div>
          <div
            className="absolute inset-0 rounded-full border-4 border-[#5865F2] border-t-transparent 
          animate-spin"
          ></div>
          <div className="absolute inset-0 flex items-center justify-center text-2xl filter drop-shadow-md">
            🤖
          </div>
        </div>

        <h2 className="text-2xl font-bold mb-2 tracking-tight">
          Authenticating...
        </h2>
        <p className="text-[#99AAB5] text-sm leading-relaxed">
          Securing your connection to Discord.
          <br />
          Please hold on a moment.
        </p>
      </div>
    </div>
  );
}
