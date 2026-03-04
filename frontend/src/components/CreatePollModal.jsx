import React, { useState, useEffect, useRef } from "react";
import {
  useGetUserGuildsQuery,
  useGetGuildChannelsQuery,
  useCreatePollMutation,
} from "../store/apiSlice";

export default function CreatePollModal({ isOpen, onClose }) {
  const dropdownRef = useRef(null);

  const [formData, setFormData] = useState({
    guild_id: "",
    report_channel_id: "",
    question: "",
    duration: 24, // Default 24 hours
    options: ["", ""], // Start with 2 empty options
  });

  const [isChannelDropdownOpen, setIsChannelDropdownOpen] = useState(false);

  const { data: guilds = [], isLoading: isLoadingGuilds } =
    useGetUserGuildsQuery(undefined, { skip: !isOpen });
  const { data: channels = [], isFetching: isFetchingChannels } =
    useGetGuildChannelsQuery(formData.guild_id, { skip: !formData.guild_id });
  const [createPoll, { isLoading: isCreating }] = useCreatePollMutation();

  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsChannelDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  useEffect(() => {
    if (!isOpen) {
      setFormData({
        guild_id: "",
        report_channel_id: "",
        question: "",
        duration: 24,
        options: ["", ""],
      });
    }
  }, [isOpen]);

  const handleOptionChange = (index, value) => {
    const newOptions = [...formData.options];
    newOptions[index] = value;
    setFormData({ ...formData, options: newOptions });
  };

  const addOption = () => {
    if (formData.options.length < 10) {
      // Discord allows max 10 options
      setFormData({ ...formData, options: [...formData.options, ""] });
    }
  };

  const removeOption = (index) => {
    if (formData.options.length <= 2) return; // Must have at least 2 options
    const newOptions = formData.options.filter((_, i) => i !== index);
    setFormData({ ...formData, options: newOptions });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    const cleanedOptions = formData.options
      .map((o) => o.trim())
      .filter((o) => o.length > 0);

    if (
      !formData.question.trim() ||
      cleanedOptions.length < 2 ||
      !formData.report_channel_id
    ) {
      alert(
        "Please enter a question, select a channel, and provide at least 2 valid options.",
      );
      return;
    }

    try {
      await createPoll({
        guild_id: formData.guild_id,
        channel_id: formData.report_channel_id,
        question: formData.question.trim(),
        duration: formData.duration,
        options: cleanedOptions,
      }).unwrap();

      onClose();
    } catch (err) {
      console.error("Failed to create poll", err);
      alert("Failed to create poll.");
    }
  };

  if (!isOpen) return null;

  const selectedChannelName =
    channels.find((c) => c.id === formData.report_channel_id)?.name ||
    "Select a channel...";

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
      <div className="bg-[#313338] w-full max-w-2xl rounded-xl p-6 md:p-8 border border-[#1e1f22] shadow-2xl max-h-[90vh] overflow-y-auto custom-scrollbar">
        <h2 className="text-2xl font-extrabold text-white mb-6 flex items-center gap-2">
          <span className="text-[#5865F2]">🗳️</span> Create Native Poll
        </h2>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* SERVER SELECTION */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Target Server
            </label>
            <div className="bg-[#1e1f22] rounded-md h-32 overflow-y-auto p-2 space-y-1 custom-scrollbar shadow-inner">
              {isLoadingGuilds ? (
                <div className="p-4 text-center text-[#99AAB5]">Loading...</div>
              ) : (
                guilds.map((g) => (
                  <div
                    key={g.id}
                    onClick={() =>
                      g.bot_present &&
                      setFormData({
                        ...formData,
                        guild_id: g.id,
                        report_channel_id: "",
                      })
                    }
                    className={`flex items-center justify-between p-2 rounded-md transition-all ${
                      formData.guild_id === g.id
                        ? "bg-[#5865F2]/20 border border-[#5865F2] cursor-default"
                        : g.bot_present
                          ? "hover:bg-[#2b2d31] cursor-pointer"
                          : "opacity-50"
                    }`}
                  >
                    <span className="text-sm font-semibold truncate text-white">
                      {g.name}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* CHANNEL SELECTION */}
          <div className="relative" ref={dropdownRef}>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Post In Channel
            </label>
            <div
              onClick={() =>
                formData.guild_id &&
                channels.length > 0 &&
                setIsChannelDropdownOpen(!isChannelDropdownOpen)
              }
              className={`w-full bg-[#1e1f22] p-3 rounded-md flex items-center justify-between text-sm shadow-inner ${!formData.guild_id ? "opacity-50 cursor-not-allowed" : "cursor-pointer text-white"}`}
            >
              <span>
                {isFetchingChannels ? "Loading..." : selectedChannelName}
              </span>
            </div>
            {isChannelDropdownOpen && (
              <div className="absolute z-10 w-full mt-1 bg-[#2b2d31] border border-[#1e1f22] rounded-md shadow-xl max-h-40 overflow-y-auto py-1">
                {channels.map((c) => (
                  <div
                    key={c.id}
                    onClick={() => {
                      setFormData({ ...formData, report_channel_id: c.id });
                      setIsChannelDropdownOpen(false);
                    }}
                    className="px-3 py-2 text-sm text-[#99AAB5] hover:bg-[#35373c] hover:text-white cursor-pointer"
                  >
                    # {c.name}
                  </div>
                ))}
              </div>
            )}
          </div>

          <hr className="border-[#3f4147]" />

          {/* QUESTION */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Question
            </label>
            <input
              type="text"
              className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm shadow-inner"
              placeholder="What are we doing for lunch?"
              value={formData.question}
              onChange={(e) =>
                setFormData({ ...formData, question: e.target.value })
              }
              required
            />
          </div>

          {/* OPTIONS */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 flex justify-between items-center">
              <span>Answers</span>
              <span className="text-[10px] bg-[#1e1f22] px-2 py-0.5 rounded">
                {formData.options.length}/10
              </span>
            </label>
            <div className="space-y-2">
              {formData.options.map((opt, index) => (
                <div key={index} className="flex items-center gap-2">
                  <input
                    type="text"
                    className="flex-1 bg-[#1e1f22] p-2.5 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm"
                    placeholder={`Option ${index + 1}`}
                    value={opt}
                    onChange={(e) => handleOptionChange(index, e.target.value)}
                    required
                  />
                  {formData.options.length > 2 && (
                    <button
                      type="button"
                      onClick={() => removeOption(index)}
                      className="text-[#da373c] hover:bg-[#da373c]/10 p-2 rounded-md transition-colors"
                    >
                      ✕
                    </button>
                  )}
                </div>
              ))}
            </div>
            {formData.options.length < 10 && (
              <button
                type="button"
                onClick={addOption}
                className="text-sm text-[#5865F2] hover:text-[#4752C4] font-bold mt-3"
              >
                + Add Option
              </button>
            )}
          </div>

          {/* DURATION */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Duration
            </label>
            <select
              className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm cursor-pointer shadow-inner appearance-none"
              value={formData.duration}
              onChange={(e) =>
                setFormData({ ...formData, duration: parseInt(e.target.value) })
              }
            >
              <option value={1}>1 Hour</option>
              <option value={4}>4 Hours</option>
              <option value={8}>8 Hours</option>
              <option value={24}>24 Hours</option>
              <option value={72}>3 Days</option>
              <option value={168}>1 Week</option>
            </select>
          </div>

          {/* FOOTER */}
          <div className="flex gap-3 pt-4 border-t border-[#3f4147]">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2.5 rounded-md font-semibold text-sm text-[#99AAB5] hover:text-white hover:bg-[#2b2d31]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isCreating}
              className={`flex-1 bg-[#5865F2] hover:bg-[#4752C4] py-2.5 rounded-md font-bold text-sm text-white transition-all shadow-lg ${isCreating ? "opacity-50" : ""}`}
            >
              {isCreating ? "Publishing..." : "Publish to Discord"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
