import React, { useState, useEffect, useRef } from "react";
import { DragDropContext, Droppable, Draggable } from "@hello-pangea/dnd";
import TimePicker from "./TimePicker";
import {
  useGetUserGuildsQuery,
  useGetGuildChannelsQuery,
  useCreateStandupMutation,
} from "../store/apiSlice";

const DAYS_OF_WEEK = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

export default function CreateStandupModal({ isOpen, onClose }) {
  const dropdownRef = useRef(null);

  // Wizard State
  const [currentStep, setCurrentStep] = useState(1);

  // Form State
  const [formData, setFormData] = useState({
    name: "",
    time: "09:00",
    days: ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"],
    guild_id: "",
    report_channel_id: "",
    questions: ["What did you accomplish yesterday?", "What will you do today?", "Are you stuck anywhere?"],
  });

  const [isChannelDropdownOpen, setIsChannelDropdownOpen] = useState(false);

  // RTK Query Hooks
  const { data: guilds = [], isLoading: isLoadingGuilds, isError: isGuildError } = useGetUserGuildsQuery(undefined, { skip: !isOpen });
  const { data: channels = [], isFetching: isFetchingChannels } = useGetGuildChannelsQuery(formData.guild_id, { skip: !formData.guild_id });
  const [createStandup, { isLoading: isCreating }] = useCreateStandupMutation();

  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsChannelDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Reset modal when opened/closed
  useEffect(() => {
    if (!isOpen) {
      setCurrentStep(1);
      setFormData({
        name: "", time: "09:00", days: ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"],
        guild_id: "", report_channel_id: "",
        questions: ["What did you accomplish yesterday?", "What will you do today?", "Are you stuck anywhere?"],
      });
    }
  }, [isOpen]);

  // --- Handlers ---
  const handleDayToggle = (day) => {
    setFormData((prev) => {
      const newDays = prev.days.includes(day) ? prev.days.filter((d) => d !== day) : [...prev.days, day];
      return { ...prev, days: newDays };
    });
  };

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

  const handleNext = () => {
    if (currentStep === 1 && (!formData.name || !formData.report_channel_id)) {
      alert("Please enter a name and select a report channel before continuing.");
      return;
    }
    if (currentStep === 2 && formData.days.length === 0) {
      alert("Please select at least one active day.");
      return;
    }
    setCurrentStep((prev) => Math.min(prev + 1, 3));
  };

  const handleBack = () => setCurrentStep((prev) => Math.max(prev - 1, 1));

  const handleSubmit = async () => {
    const cleanedQuestions = formData.questions.map((q) => q.trim()).filter((q) => q.length > 0);
    if (cleanedQuestions.length === 0) {
      alert("Please ensure you have at least one question.");
      return;
    }

    try {
      await createStandup({ 
        ...formData, 
        questions: cleanedQuestions,
        days: formData.days.join(",") // Convert array to string for Go backend
      }).unwrap();
      onClose();
    } catch (err) {
      console.error("Creation failed", err);
    }
  };

  if (!isOpen) return null;

  const selectedChannelName = channels.find((c) => c.id === formData.report_channel_id)?.name || "Select a channel...";

  const steps = [
    { num: 1, title: "Basics", desc: "Name & Location" },
    { num: 2, title: "Schedule", desc: "When to run it" },
    { num: 3, title: "Questions", desc: "What to ask" },
  ];

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
      <div className="bg-[#313338] w-full max-w-4xl rounded-xl border border-[#1e1f22] shadow-2xl flex overflow-hidden min-h-125">
        
        {/* LEFT SIDEBAR: Stepper */}
        <div className="w-1/3 bg-[#2b2d31] p-8 border-r border-[#1e1f22] hidden md:block">
          <h2 className="text-xl font-extrabold text-white mb-8 flex items-center gap-2">
            <span className="text-[#5865F2]">✨</span> New Standup
          </h2>
          <div className="space-y-8 relative before:absolute before:inset-0 before:ml-3.75 before:-translate-x-px md:before:mx-auto md:before:translate-x-0 before:h-full before:w-0.5 before:bg-linear-to-b before:from-transparent before:via-[#404249] before:to-transparent">
            {steps.map((step) => (
              <div key={step.num} className="relative flex items-center gap-4">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-sm shrink-0 z-10 transition-colors ${
                  currentStep >= step.num ? "bg-[#5865F2] text-white" : "bg-[#1e1f22] text-[#99AAB5] border border-[#404249]"
                }`}>
                  {currentStep > step.num ? "✓" : step.num}
                </div>
                <div>
                  <h3 className={`font-bold ${currentStep >= step.num ? "text-white" : "text-[#99AAB5]"}`}>{step.title}</h3>
                  <p className="text-xs text-[#99AAB5]">{step.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* RIGHT CONTENT: Form Area */}
        <div className="flex-1 flex flex-col p-6 md:p-8 bg-[#313338]">
          <div className="flex-1">
            
            {/* STEP 1: BASICS */}
            {currentStep === 1 && (
              <div className="space-y-6 animate-fade-in">
                <h3 className="text-2xl font-bold text-white mb-2">Let's set up the basics</h3>
                <div>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">Team Name</label>
                  <input
                    type="text"
                    className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm"
                    placeholder="e.g. Frontend Sync"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  />
                </div>

                <div>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 flex justify-between items-center">
                    <span>Discord Server</span>
                    <span className="normal-case font-medium text-[10px] text-[#5865F2]">Must invite bot to select</span>
                  </label>
                  <div className="bg-[#1e1f22] rounded-md h-40 overflow-y-auto p-2 space-y-1 custom-scrollbar">
                    {isLoadingGuilds ? <div className="p-4 text-center text-[#99AAB5]">Loading...</div> : 
                      guilds.map((g) => (
                        <div
                          key={g.id}
                          onClick={() => g.bot_present && setFormData({ ...formData, guild_id: g.id, report_channel_id: "" })}
                          className={`flex items-center justify-between p-3 rounded-md transition-all ${
                            formData.guild_id === g.id 
                              ? "bg-[#5865F2]/20 border border-[#5865F2] cursor-default" 
                              : g.bot_present ? "hover:bg-[#2b2d31] cursor-pointer border border-transparent" : "opacity-60 border border-transparent"
                          }`}
                        >
                          <span className={`text-sm font-semibold truncate ${g.bot_present ? "text-white" : "text-[#99AAB5]"}`}>
                            {g.name}
                          </span>
                          
                          {/* The missing Invite Button is back! */}
                          {!g.bot_present && (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation();
                                const inviteUrl = `https://discord.com/api/oauth2/authorize?client_id=${import.meta.env.VITE_DISCORD_CLIENT_ID}&permissions=8&scope=bot%20applications.commands&guild_id=${g.id}&disable_guild_select=true`;
                                window.open(inviteUrl, "_blank", "width=500,height=700");
                              }}
                              className="text-[10px] px-3 py-1.5 rounded font-bold uppercase bg-[#23a559] hover:bg-[#1d8a4a] text-white shadow-sm transition-transform active:scale-95"
                            >
                              Invite Bot
                            </button>
                          )}
                        </div>
                      ))
                    }
                  </div>
                </div>

                <div className="relative" ref={dropdownRef}>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">Report Channel</label>
                  <div
                    onClick={() => formData.guild_id && channels.length > 0 && setIsChannelDropdownOpen(!isChannelDropdownOpen)}
                    className={`w-full bg-[#1e1f22] p-3 rounded-md flex items-center justify-between text-sm ${!formData.guild_id ? "opacity-50" : "cursor-pointer text-white"}`}
                  >
                    <span>{isFetchingChannels ? "Loading..." : selectedChannelName}</span>
                  </div>
                  {isChannelDropdownOpen && (
                    <div className="absolute z-10 w-full mt-1 bg-[#2b2d31] border border-[#1e1f22] rounded-md shadow-xl max-h-40 overflow-y-auto py-1">
                      {channels.map((c) => (
                        <div
                          key={c.id}
                          onClick={() => { setFormData({ ...formData, report_channel_id: c.id }); setIsChannelDropdownOpen(false); }}
                          className="px-3 py-2 text-sm text-[#99AAB5] hover:bg-[#35373c] hover:text-white cursor-pointer"
                        >
                          # {c.name}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* STEP 2: SCHEDULE */}
            {currentStep === 2 && (
              <div className="space-y-6 animate-fade-in">
                <h3 className="text-2xl font-bold text-white mb-2">When should the standup happen?</h3>
                
                <div>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">Trigger Time</label>
                  <TimePicker value={formData.time} onChange={(val) => setFormData({ ...formData, time: val })} />
                  <p className="text-xs text-[#99AAB5] mt-2">Team members will receive DMs at this time in their local timezone.</p>
                </div>

                <div className="pt-4 border-t border-[#3f4147]">
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-3 block">Every week on...</label>
                  <div className="flex flex-wrap gap-2">
                    {DAYS_OF_WEEK.map((day) => (
                      <button
                        key={day}
                        type="button"
                        onClick={() => handleDayToggle(day)}
                        className={`px-4 py-2 rounded-md text-sm font-bold transition-all border ${
                          formData.days.includes(day)
                            ? "bg-[#5865F2]/20 border-[#5865F2] text-white"
                            : "bg-[#1e1f22] border-transparent text-[#99AAB5] hover:text-white hover:border-[#404249]"
                        }`}
                      >
                        {day.substring(0, 3)}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* STEP 3: QUESTIONS */}
            {currentStep === 3 && (
              <div className="space-y-4 animate-fade-in flex flex-col h-full">
                <h3 className="text-2xl font-bold text-white mb-2">What to ask your team</h3>
                <DragDropContext onDragEnd={onDragEnd}>
                  <Droppable droppableId="questions-list">
                    {(provided) => (
                      <div {...provided.droppableProps} ref={provided.innerRef} className="space-y-3">
                        {formData.questions.map((q, index) => (
                          <Draggable key={`q-${index}`} draggableId={`q-${index}`} index={index}>
                            {(provided, snapshot) => (
                              <div ref={provided.innerRef} {...provided.draggableProps} className={`flex items-center gap-3 bg-[#1e1f22] p-3 rounded-md border ${snapshot.isDragging ? "border-[#5865F2] shadow-xl" : "border-transparent"}`}>
                                <div {...provided.dragHandleProps} className="text-[#404249] cursor-grab px-1">
                                  <svg width="10" height="16" fill="currentColor"><circle cx="3" cy="3" r="1.5"/><circle cx="3" cy="8" r="1.5"/><circle cx="3" cy="13" r="1.5"/><circle cx="7" cy="3" r="1.5"/><circle cx="7" cy="8" r="1.5"/><circle cx="7" cy="13" r="1.5"/></svg>
                                </div>
                                <input type="text" className="flex-1 bg-transparent outline-none text-sm text-white" value={q} onChange={(e) => handleQuestionChange(index, e.target.value)} />
                                {formData.questions.length > 1 && (
                                  <button type="button" onClick={() => removeQuestion(index)} className="text-[#404249] hover:text-[#da373c]">✕</button>
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
                  <button type="button" onClick={addQuestion} className="text-sm text-[#5865F2] hover:text-[#4752C4] font-bold text-left">+ Add another question</button>
                )}
              </div>
            )}
          </div>

          {/* FOOTER NAVIGATION */}
          <div className="flex justify-between items-center pt-6 mt-6 border-t border-[#3f4147]">
            <button type="button" onClick={currentStep === 1 ? onClose : handleBack} className="px-6 py-2.5 rounded-md font-semibold text-sm text-[#99AAB5] hover:text-white hover:bg-[#2b2d31]">
              {currentStep === 1 ? "Cancel" : "← Back"}
            </button>
            
            {currentStep < 3 ? (
              <button type="button" onClick={handleNext} className="bg-[#5865F2] hover:bg-[#4752C4] px-8 py-2.5 rounded-md font-bold text-sm text-white transition-all shadow-lg">
                Continue
              </button>
            ) : (
              <button type="button" onClick={handleSubmit} disabled={isCreating} className={`bg-[#23a559] hover:bg-[#1d8a4a] px-8 py-2.5 rounded-md font-bold text-sm text-white transition-all shadow-lg ${isCreating ? 'opacity-50' : ''}`}>
                {isCreating ? "Creating..." : "Finish Setup"}
              </button>
            )}
          </div>

        </div>
      </div>
    </div>
  );
}