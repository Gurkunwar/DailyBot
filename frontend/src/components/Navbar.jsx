import React from "react";
import { useDispatch, useSelector } from "react-redux";
import { useNavigate } from "react-router-dom";
import { logout } from "../store/authSlice"; // <-- Add this import

export default function Navbar() {
  const navigate = useNavigate();
  const dispatch = useDispatch();

  // This will now successfully pull from state.auth
  const { isAuthenticated } = useSelector((state) => state.auth);

  const handleAuthAction = () => {
    if (isAuthenticated) {
      dispatch(logout());
      navigate("/");
    } else {
      navigate("/login");
    }
  };

  return (
    <nav className="flex items-center justify-between px-8 py-5 max-w-7xl mx-auto border-b border-[#313338]">
      <div className="flex items-center gap-3">
        <span className="text-2xl">🤖</span>
        <span className="text-xl font-bold tracking-tight">AsyncFlow</span>
      </div>

      <div className="flex items-center gap-6">
        {isAuthenticated && (
          <button 
            onClick={() => navigate('/dashboard')}
            className="text-[#99AAB5] hover:text-white transition-colors text-sm font-semibold"
          >
            Dashboard
          </button>
        )}
        
        <button
          onClick={handleAuthAction}
          className="bg-[#5865F2] hover:bg-[#4752C4] text-white text-sm font-semibold py-2 px-5 
          rounded-full transition-colors duration-200 cursor-pointer shadow-md"
        >
          {isAuthenticated ? "Logout" : "Login"}
        </button>
      </div>
    </nav>
  );
}