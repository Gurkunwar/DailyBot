import React, { useState, useEffect, useRef } from "react";
import { DragDropContext, Droppable, Draggable } from "@hello-pangea/dnd";
import TimePicker from "./TimePicker";
// 1. Import the hooks from your API slice
import {
  useGetUserGuildsQuery,
  useGetGuildChannelsQuery,
  useCreateStandupMutation,
} from "../store/apiSlice";

export default function CreateStandupModal({ isOpen, onClose }) {
  const dropdownRef = useRef(null);

  // Form State
  const [formData, setFormData] = useState({
    name: "",
    time: "09:00",
    guild_id: "",
    report_channel_id: "",
    questions: ["What did you do yesterday?", "What will you do today?"],
  });

  const [isChannelDropdownOpen, setIsChannelDropdownOpen] = useState(false);

  // 2. RTK Query Hooks
  // Fetch Guilds
  const { 
    data: guilds = [], 
    isLoading: isLoadingGuilds, 
    isError: isGuildError 
  } = useGetUserGuildsQuery(undefined, { skip: !isOpen });

  // Fetch Channels (Skipped until a guild is selected)
  const { 
    data: channels = [], 
    isFetching: isFetchingChannels 
  } = useGetGuildChannelsQuery(formData.guild_id, { 
    skip: !formData.guild_id 
  });

  // Create Standup Mutation
  const [createStandup, { isLoading: isCreating }] = useCreateStandupMutation();

  // Close custom dropdown if clicked outside
  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsChannelDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // --- Handlers ---
  const handleQuestionChange = (index, value) => {
    const newQuestions = [...formData.questions];
    newQuestions[index] = value;
    setFormData({ ...formData, questions: newQuestions });
  };

  const addQuestion = () => {
    if (formData.questions.length < 5) {
      setFormData({ ...formData, questions: [...formData.questions, ""] });
    }
  };

  const removeQuestion = (index) => {
    if (formData.questions.length <= 1) return;
    const newQuestions = formData.questions.filter((_, i) => i !== index);
    setFormData({ ...formData, questions: newQuestions });
  };

  const onDragEnd = (result) => {
    if (!result.destination) return;
    const items = Array.from(formData.questions);
    const [reorderedItem] = items.splice(result.source.index, 1);
    items.splice(result.destination.index, 0, reorderedItem);
    setFormData({ ...formData, questions: items });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    const cleanedQuestions = formData.questions
      .map((q) => q.trim())
      .filter((q) => q.length > 0);

    if (cleanedQuestions.length === 0 || !formData.report_channel_id) {
      alert("Please ensure all fields are filled.");
      return;
    }

    try {
      // 3. Use the mutation hook
      await createStandup({ ...formData, questions: cleanedQuestions }).unwrap();
      
      // Cleanup on success
      setFormData({
        name: "",
        time: "09:00",
        guild_id: "",
        report_channel_id: "",
        questions: ["What did you do yesterday?", "What will you do today?"],
      });
      onClose();
    } catch (err) {
      console.error("Creation failed", err);
    }
  };

  if (!isOpen) return null;

  const selectedChannelName =
    channels.find((c) => c.id === formData.report_channel_id)?.name ||
    "Select a channel...";

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
      <div className="bg-[#313338] w-full max-w-2xl rounded-xl p-6 md:p-8 border border-[#1e1f22] max-h-[90vh] overflow-y-auto shadow-2xl custom-scrollbar">
        <h2 className="text-2xl font-extrabold mb-6 flex items-center gap-2 text-white">
          <span className="text-[#5865F2]">✨</span> New Standup
        </h2>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* TEAM NAME */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Team Name
            </label>
            <input
              type="text"
              className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm transition-all"
              placeholder="e.g. Backend Engineering Sync"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
            />
          </div>

          {/* SERVER SELECTION */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 flex items-center justify-between">
              <span>Discord Server</span>
              <span className="normal-case font-medium text-[10px] text-[#5865F2]">Select to fetch channels</span>
            </label>
            <div className="bg-[#1e1f22] rounded-md border border-[#1e1f22] h-40 overflow-y-auto p-2 space-y-1 custom-scrollbar shadow-inner">
              {isGuildError ? (
                <div className="h-full flex items-center justify-center p-4 text-center text-[#da373c] text-sm font-semibold">
                  Failed to fetch servers. Please refresh.
                </div>
              ) : isLoadingGuilds ? (
                <div className="h-full flex items-center justify-center gap-2 text-[#99AAB5]">
                   <svg className="animate-spin h-5 w-5 text-[#5865F2]" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span className="text-sm">Fetching...</span>
                </div>
              ) : (
                guilds.map((g) => (
                  <div
                    key={g.id}
                    onClick={() => g.bot_present && setFormData({ ...formData, guild_id: g.id, report_channel_id: "" })}
                    className={`flex items-center justify-between p-3 rounded-md transition-all ${
                      formData.guild_id === g.id
                        ? "bg-[#5865F2]/20 border border-[#5865F2] cursor-default"
                        : g.bot_present ? "hover:bg-[#2b2d31] border border-transparent cursor-pointer" : "opacity-50 grayscale"
                    }`}
                  >
                    <div className="flex items-center gap-3 overflow-hidden">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold ${formData.guild_id === g.id ? "bg-[#5865F2] text-white" : "bg-[#313338] text-[#99AAB5]"}`}>
                        {g.name.charAt(0)}
                      </div>
                      <span className="text-sm font-semibold truncate">{g.name}</span>
                    </div>
                    {!g.bot_present && (
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          const inviteUrl = `https://discord.com/api/oauth2/authorize?client_id=${import.meta.env.VITE_DISCORD_CLIENT_ID}&permissions=8&scope=bot%20applications.commands&guild_id=${g.id}&disable_guild_select=true`;
                          window.open(inviteUrl, "_blank", "width=500,height=700");
                        }}
                        className="text-[10px] px-3 py-1.5 rounded font-bold uppercase bg-[#23a559] hover:bg-[#1d8a4a] text-white"
                      >
                        Invite Bot
                      </button>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>

          {/* CHANNEL SELECTION */}
          <div className="relative" ref={dropdownRef}>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Report Channel
            </label>
            <div
              onClick={() => formData.guild_id && channels.length > 0 && setIsChannelDropdownOpen(!isChannelDropdownOpen)}
              className={`w-full bg-[#1e1f22] p-3 rounded-md border flex items-center justify-between text-sm transition-all ${
                !formData.guild_id ? "opacity-50 cursor-not-allowed" : "cursor-pointer text-white"
              } ${isChannelDropdownOpen ? "border-[#5865F2]" : "border-transparent hover:border-[#404249]"}`}
            >
              <div className="flex items-center gap-2 truncate">
                {formData.report_channel_id && <span className="text-[#99AAB5] font-bold text-lg leading-none">#</span>}
                <span className="truncate">{isFetchingChannels ? "Loading channels..." : selectedChannelName}</span>
              </div>
              <svg className={`w-4 h-4 text-[#99AAB5] transition-transform ${isChannelDropdownOpen ? "rotate-180" : ""}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </div>

            {isChannelDropdownOpen && (
              <div className="absolute z-10 w-full mt-1 bg-[#2b2d31] border border-[#1e1f22] rounded-md shadow-xl max-h-48 overflow-y-auto custom-scrollbar py-1">
                {channels.map((c) => (
                  <div
                    key={c.id}
                    onClick={() => { setFormData({ ...formData, report_channel_id: c.id }); setIsChannelDropdownOpen(false); }}
                    className={`px-3 py-2 text-sm flex items-center gap-2 cursor-pointer transition-colors ${formData.report_channel_id === c.id ? "bg-[#404249] text-white" : "text-[#99AAB5] hover:bg-[#35373c]"}`}
                  >
                    <span className="font-bold opacity-50">#</span>
                    <span className="truncate">{c.name}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* QUESTIONS */}
          <div className="pt-2 border-t border-[#3f4147]">
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-3 flex justify-between items-center">
              <span>Questions</span>
              <span className="bg-[#232428] px-2 py-0.5 rounded text-[10px] font-mono border border-[#1e1f22]">
                {formData.questions.length}/5
              </span>
            </label>

            <DragDropContext onDragEnd={onDragEnd}>
              <Droppable droppableId="questions-list">
                {(provided) => (
                  <div {...provided.droppableProps} ref={provided.innerRef} className="space-y-2.5">
                    {formData.questions.map((q, index) => (
                      <Draggable key={`q-${index}`} draggableId={`q-${index}`} index={index}>
                        {(provided, snapshot) => (
                          <div
                            ref={provided.innerRef}
                            {...provided.draggableProps}
                            className={`flex items-center gap-3 bg-[#1e1f22] p-2.5 rounded-md border ${snapshot.isDragging ? "border-[#5865F2] shadow-xl" : "border-[#1e1f22]"}`}
                          >
                            <div {...provided.dragHandleProps} className="text-[#404249] hover:text-white cursor-grab px-1">
                              <svg width="10" height="16" fill="currentColor">
                                <circle cx="3" cy="3" r="1.5" /><circle cx="3" cy="8" r="1.5" /><circle cx="3" cy="13" r="1.5" />
                                <circle cx="7" cy="3" r="1.5" /><circle cx="7" cy="8" r="1.5" /><circle cx="7" cy="13" r="1.5" />
                              </svg>
                            </div>
                            <input
                              type="text"
                              className="flex-1 bg-transparent outline-none text-sm text-white"
                              value={q}
                              placeholder={`Question ${index + 1}...`}
                              onChange={(e) => handleQuestionChange(index, e.target.value)}
                              required
                            />
                            {formData.questions.length > 1 && (
                              <button type="button" onClick={() => removeQuestion(index)} className="text-[#404249] hover:text-[#da373c]"><svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" /></svg></button>
                            )}
                          </div>
                        )}
                      </Draggable>
                    ))}
                    {provided.placeholder}
                  </div>
                )}
              </Droppable>
            </DragDropContext>

            {formData.questions.length < 5 && (
              <button type="button" onClick={addQuestion} className="mt-4 text-xs text-[#5865F2] hover:text-[#4752C4] font-bold flex items-center gap-1 transition-colors">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M12 4v16m8-8H4" /></svg>
                Add another question
              </button>
            )}
          </div>

          {/* TRIGGER TIME */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Daily Trigger Time
            </label>
            <TimePicker
              value={formData.time}
              onChange={(newTime) => setFormData({ ...formData, time: newTime })}
            />
          </div>

          {/* FOOTER */}
          <div className="flex gap-3 pt-4 mt-6 border-t border-[#3f4147]">
            <button type="button" onClick={onClose} className="flex-1 py-2.5 rounded-md font-semibold text-sm text-[#99AAB5] hover:text-white hover:bg-[#2b2d31]">
              Cancel
            </button>
            <button
              type="submit"
              disabled={isCreating}
              className={`flex-1 bg-[#5865F2] hover:bg-[#4752C4] py-2.5 rounded-md font-bold text-sm text-white transition-all transform active:scale-95 shadow-lg ${isCreating ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              {isCreating ? "Creating..." : "Create Standup"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}