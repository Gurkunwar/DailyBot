import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";

import Login from "./pages/Login";
import AuthCallback from "./pages/AuthCallback";
import Dashboard from "./pages/Dashboard";
import Home from "./pages/Home";
import ProtectedRoute from "./components/ProtectedRoute";
import MyStandups from "./pages/MyStandups";
import ManageStandup from "./pages/ManageStandup";
import MyPolls from "./pages/MyPolls";
import ManagePoll from "./pages/ManagePoll";

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
        <Route path="/standups/:id" element={<ManageStandup />} />
        <Route path="/polls" element={<MyPolls />} />
        <Route path="/polls/:id" element={<ManagePoll />} />
      </Routes>
    </Router>
  );
}

export default App;
