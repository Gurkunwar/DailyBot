import React, { useState, useEffect, useRef } from "react";
import {
  useGetUserGuildsQuery,
  useGetGuildChannelsQuery,
  useCreatePollMutation,
} from "../store/apiSlice";

export default function CreatePollModal({ isOpen, onClose }) {
  const dropdownRef = useRef(null);

  // Wizard State
  const [currentStep, setCurrentStep] = useState(1);

  // Form State
  const [formData, setFormData] = useState({
    guild_id: "",
    report_channel_id: "",
    question: "",
    duration: 24, // Default 24 hours
    options: ["", ""], // Start with 2 empty options
  });

  const [isChannelDropdownOpen, setIsChannelDropdownOpen] = useState(false);

  // RTK Query Hooks
  const { data: guilds = [], isLoading: isLoadingGuilds } = useGetUserGuildsQuery(undefined, { skip: !isOpen });
  const { data: channels = [], isFetching: isFetchingChannels } = useGetGuildChannelsQuery(formData.guild_id, { skip: !formData.guild_id });
  const [createPoll, { isLoading: isCreating }] = useCreatePollMutation();

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

  // Reset modal when opened/closed
  useEffect(() => {
    if (!isOpen) {
      setCurrentStep(1);
      setFormData({
        guild_id: "",
        report_channel_id: "",
        question: "",
        duration: 24,
        options: ["", ""],
      });
      setIsChannelDropdownOpen(false);
    }
  }, [isOpen]);

  // --- Handlers ---
  const handleOptionChange = (index, value) => {
    const newOptions = [...formData.options];
    newOptions[index] = value;
    setFormData({ ...formData, options: newOptions });
  };

  const addOption = () => {
    if (formData.options.length < 10) {
      setFormData({ ...formData, options: [...formData.options, ""] });
    }
  };

  const removeOption = (index) => {
    if (formData.options.length <= 2) return;
    const newOptions = formData.options.filter((_, i) => i !== index);
    setFormData({ ...formData, options: newOptions });
  };

  const handleNext = () => {
    if (currentStep === 1 && (!formData.guild_id || !formData.report_channel_id)) {
      alert("Please select a server and a channel before continuing.");
      return;
    }
    if (currentStep === 2 && !formData.question.trim()) {
      alert("Please enter a question for your poll.");
      return;
    }
    setCurrentStep((prev) => Math.min(prev + 1, 3));
  };

  const handleBack = () => setCurrentStep((prev) => Math.max(prev - 1, 1));

  const handleSubmit = async () => {
    const cleanedOptions = formData.options.map((o) => o.trim()).filter((o) => o.length > 0);

    if (cleanedOptions.length < 2) {
      alert("Please provide at least 2 valid options for the poll.");
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

  const selectedChannelName = channels.find((c) => c.id === formData.report_channel_id)?.name || "Select a channel...";

  const steps = [
    { num: 1, title: "Location", desc: "Server & Channel" },
    { num: 2, title: "Details", desc: "Question & Duration" },
    { num: 3, title: "Options", desc: "Poll Answers" },
  ];

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
      <div className="bg-[#313338] w-full max-w-4xl rounded-xl border border-[#1e1f22] shadow-2xl flex overflow-hidden min-h-125">
        
        {/* LEFT SIDEBAR: Stepper */}
        <div className="w-1/3 bg-[#2b2d31] p-8 border-r border-[#1e1f22] hidden md:block">
          <h2 className="text-xl font-extrabold text-white mb-8 flex items-center gap-2">
            <span className="text-[#5865F2]">🗳️</span> New Poll
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
        <div className="flex-1 flex flex-col p-6 md:p-8 bg-[#313338] max-h-[90vh] overflow-y-auto custom-scrollbar">
          <div className="flex-1">
            
            {/* STEP 1: LOCATION */}
            {currentStep === 1 && (
              <div className="space-y-6 animate-fade-in">
                <h3 className="text-2xl font-bold text-white mb-2">Where should we post this?</h3>

                <div>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 flex justify-between items-center">
                    <span>Discord Server</span>
                    <span className="normal-case font-medium text-[10px] text-[#5865F2]">Must invite bot to select</span>
                  </label>
                  <div className="bg-[#1e1f22] rounded-md h-40 overflow-y-auto p-2 space-y-1 custom-scrollbar shadow-inner">
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
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">Post In Channel</label>
                  <div
                    onClick={() => formData.guild_id && channels.length > 0 && setIsChannelDropdownOpen(!isChannelDropdownOpen)}
                    className={`w-full bg-[#1e1f22] p-3 rounded-md flex items-center justify-between text-sm shadow-inner ${!formData.guild_id ? "opacity-50 cursor-not-allowed" : "cursor-pointer text-white"}`}
                  >
                    <span>{isFetchingChannels ? "Loading..." : selectedChannelName}</span>
                  </div>
                  {isChannelDropdownOpen && (
                    <div className="absolute z-10 w-full mt-1 bg-[#2b2d31] border border-[#1e1f22] rounded-md shadow-xl max-h-40 overflow-y-auto py-1 custom-scrollbar">
                      {channels.map((c) => (
                        <div
                          key={c.id}
                          onClick={() => { setFormData({ ...formData, report_channel_id: c.id }); setIsChannelDropdownOpen(false); }}
                          className="px-3 py-2 text-sm text-[#99AAB5] hover:bg-[#35373c] hover:text-white cursor-pointer transition-colors"
                        >
                          # {c.name}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* STEP 2: DETAILS */}
            {currentStep === 2 && (
              <div className="space-y-6 animate-fade-in">
                <h3 className="text-2xl font-bold text-white mb-2">What's the question?</h3>
                
                <div>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">Question</label>
                  <input
                    type="text"
                    className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm shadow-inner transition-colors"
                    placeholder="e.g. What are we doing for lunch?"
                    value={formData.question}
                    onChange={(e) => setFormData({ ...formData, question: e.target.value })}
                    required
                  />
                </div>

                <div>
                  <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">Duration</label>
                  <select
                    className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm cursor-pointer shadow-inner appearance-none"
                    value={formData.duration}
                    onChange={(e) => setFormData({ ...formData, duration: parseInt(e.target.value) })}
                  >
                    <option value={1}>1 Hour</option>
                    <option value={4}>4 Hours</option>
                    <option value={8}>8 Hours</option>
                    <option value={24}>24 Hours</option>
                    <option value={72}>3 Days</option>
                    <option value={168}>1 Week</option>
                  </select>
                  <p className="text-xs text-[#99AAB5] mt-2">Discord will automatically lock the poll after this time.</p>
                </div>
              </div>
            )}

            {/* STEP 3: OPTIONS */}
            {currentStep === 3 && (
              <div className="space-y-4 animate-fade-in flex flex-col h-full">
                <h3 className="text-2xl font-bold text-white mb-2">Add your answers</h3>
                
                <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider flex justify-between items-center">
                  <span>Options</span>
                  <span className="text-[10px] bg-[#1e1f22] px-2 py-0.5 rounded border border-[#404249]">
                    {formData.options.length}/10
                  </span>
                </label>

                <div className="space-y-3">
                  {formData.options.map((opt, index) => (
                    <div key={index} className="flex items-center gap-3">
                      <div className="text-[#404249] font-bold w-4 text-center">{index + 1}.</div>
                      <input
                        type="text"
                        className="flex-1 bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] outline-none text-white text-sm transition-colors"
                        placeholder={`Option ${index + 1}`}
                        value={opt}
                        onChange={(e) => handleOptionChange(index, e.target.value)}
                        required
                      />
                      {formData.options.length > 2 && (
                        <button
                          type="button"
                          onClick={() => removeOption(index)}
                          className="text-[#404249] hover:text-[#da373c] p-2 transition-colors"
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
                    className="text-sm text-[#5865F2] hover:text-[#4752C4] font-bold text-left pt-2 transition-colors"
                  >
                    + Add another option
                  </button>
                )}
              </div>
            )}
          </div>

          {/* FOOTER NAVIGATION */}
          <div className="flex justify-between items-center pt-6 mt-6 border-t border-[#3f4147]">
            <button type="button" onClick={currentStep === 1 ? onClose : handleBack} className="px-6 py-2.5 rounded-md font-semibold text-sm text-[#99AAB5] hover:text-white hover:bg-[#2b2d31] transition-colors">
              {currentStep === 1 ? "Cancel" : "← Back"}
            </button>
            
            {currentStep < 3 ? (
              <button type="button" onClick={handleNext} className="bg-[#5865F2] hover:bg-[#4752C4] px-8 py-2.5 rounded-md font-bold text-sm text-white transition-all shadow-lg">
                Continue
              </button>
            ) : (
              <button type="button" onClick={handleSubmit} disabled={isCreating} className={`bg-[#23a559] hover:bg-[#1d8a4a] px-8 py-2.5 rounded-md font-bold text-sm text-white transition-all shadow-lg ${isCreating ? 'opacity-50' : ''}`}>
                {isCreating ? "Publishing..." : "Publish to Discord"}
              </button>
            )}
          </div>

        </div>
      </div>

      <style dangerouslySetInnerHTML={{ __html: `
        @keyframes fadeIn { from { opacity: 0; transform: translateX(10px); } to { opacity: 1; transform: translateX(0); } }
        .animate-fade-in { animation: fadeIn 0.2s ease-out forwards; }
      `}} />
    </div>
  );
}