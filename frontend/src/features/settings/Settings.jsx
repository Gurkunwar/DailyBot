import React, { useState, useEffect } from "react";
import Sidebar from "../../components/Sidebar";
import {
  useGetUserSettingsQuery,
  useUpdateUserSettingsMutation,
} from "../../store/apiSlice";

const COMMON_TIMEZONES = [
  { label: "Universal Time (UTC)", value: "UTC" },
  { label: "US Pacific (PST/PDT)", value: "America/Los_Angeles" },
  { label: "US Central (CST/CDT)", value: "America/Chicago" },
  { label: "US East (EST/EDT)", value: "America/New_York" },
  { label: "London (GMT/BST)", value: "Europe/London" },
  { label: "Europe Central (CET)", value: "Europe/Paris" },
  { label: "India (IST)", value: "Asia/Kolkata" },
  { label: "Singapore (SGT)", value: "Asia/Singapore" },
  { label: "Japan (JST)", value: "Asia/Tokyo" },
  { label: "Australia East (AEST)", value: "Australia/Sydney" },
];

export default function Settings() {
  const { data: settings, isLoading } = useGetUserSettingsQuery();
  const [updateSettings, { isLoading: isUpdating }] =
    useUpdateUserSettingsMutation();

  const [timezone, setTimezone] = useState("UTC");
  const [saveStatus, setSaveStatus] = useState({ message: "", type: "" });

  useEffect(() => {
    if (settings?.timezone) {
      setTimezone(settings.timezone);
    }
  }, [settings]);

  const handleSave = async () => {
    try {
      await updateSettings({ timezone }).unwrap();
      setSaveStatus({
        message: "Settings saved successfully!",
        type: "success",
      });
      setTimeout(() => setSaveStatus({ message: "", type: "" }), 3000);
    } catch (err) {
      setSaveStatus({ message: "Failed to save settings.", type: "error" });
    }
  };

  return (
    <div className="flex h-screen bg-[#313338] text-white overflow-hidden font-sans">
      <Sidebar />
      <main className="flex-1 flex flex-col min-w-0 overflow-y-auto custom-scrollbar">
        <header className="h-14 border-b border-[#1e1f22] flex items-center px-8 shadow-sm shrink-0">
          <h1 className="text-lg font-bold">App Settings</h1>
        </header>

        <div className="p-8 max-w-4xl mx-auto w-full">
          <div className="mb-8">
            <h2 className="text-2xl font-bold mb-2">User Preferences</h2>
            <p className="text-[#99AAB5] text-sm">
              Manage your personal bot settings and timezone configurations.
            </p>
          </div>

          <div className="bg-[#2b2d31] border border-[#1e1f22] rounded-xl shadow-sm p-6 mb-6">
            <h3 className="text-[11px] font-bold uppercase tracking-widest text-[#99AAB5] mb-4">
              Date & Time
            </h3>

            <div className="flex flex-col md:flex-row md:items-start justify-between gap-6">
              <div className="flex-1">
                <label className="block text-sm font-semibold mb-2 text-gray-200">
                  Your Timezone
                </label>
                <p className="text-[#99AAB5] text-xs leading-relaxed mb-3 max-w-md">
                  This timezone determines when you receive your daily Standup
                  reminders from the bot. By default, this is set to UTC.
                </p>

                {isLoading ? (
                  <div
                    className="h-10 w-full md:w-72 bg-[#1e1f22] animate-pulse rounded-md 
                  border border-[#3f4147]"
                  ></div>
                ) : (
                  <div className="relative w-full md:w-72">
                    <select
                      value={timezone}
                      onChange={(e) => setTimezone(e.target.value)}
                      className="w-full bg-[#1e1f22] text-sm text-gray-200 px-4 py-2.5 rounded-md outline-none 
                      border border-[#3f4147] focus:border-[#5865F2] cursor-pointer shadow-inner 
                      appearance-none transition-colors"
                    >
                      {/* FIX: Now correctly maps over the curated objects */}
                      {COMMON_TIMEZONES.map((tz) => (
                        <option key={tz.value} value={tz.value}>
                          {tz.label}
                        </option>
                      ))}
                    </select>
                    <div className="absolute right-3 top-3 pointer-events-none text-[#99AAB5]">
                      <svg
                        className="w-4 h-4"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth="2"
                          d="M19 9l-7 7-7-7"
                        ></path>
                      </svg>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="mt-8 pt-6 border-t border-[#3f4147]/50 flex items-center justify-between">
              <div className="text-sm font-medium">
                {saveStatus.message && (
                  <span
                    className={
                      saveStatus.type === "success"
                        ? "text-[#43b581]"
                        : "text-[#da373c]"
                    }
                  >
                    {saveStatus.type === "success" ? "✓ " : "❌ "}
                    {saveStatus.message}
                  </span>
                )}
              </div>
              <button
                onClick={handleSave}
                disabled={
                  isLoading || isUpdating || timezone === settings?.timezone
                }
                className="bg-[#5865F2] hover:bg-[#4752C4] disabled:bg-[#5865F2]/50 
                disabled:cursor-not-allowed px-6 py-2 rounded font-semibold text-sm transition-colors 
                cursor-pointer shadow-md flex items-center gap-2"
              >
                {isUpdating ? "Saving..." : "Save Changes"}
              </button>
            </div>
          </div>

          <div
            className="bg-[#2b2d31] border border-[#1e1f22] rounded-xl shadow-sm p-6 opacity-60 
          grayscale cursor-not-allowed"
          >
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-[11px] font-bold uppercase tracking-widest text-[#99AAB5] mb-1">
                  Notifications (Coming Soon)
                </h3>
                <p className="text-gray-400 text-sm">
                  Configure DM notifications and email alerts.
                </p>
              </div>
              <div
                className="bg-[#1e1f22] text-[#99AAB5] text-xs px-3 py-1 rounded font-bold 
                tracking-wider"
              >
                WIP
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
