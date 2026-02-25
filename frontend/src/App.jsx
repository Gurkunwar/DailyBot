import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";

import Login from "./pages/Login";
import AuthCallback from "./pages/AuthCallback";
import Dashboard from "./pages/Dashboard";
import Home from "./pages/Home";
import ProtectedRoute from "./components/ProtectedRoute";
import MyStandups from "./pages/MyStandups";

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/auth/callback" element={<AuthCallback />} />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />
        <Route
          path="/standups"
          element={
            <ProtectedRoute>
              <MyStandups />
            </ProtectedRoute>
          }
        />
      </Routes>
    </Router>
  );
}

export default App;
