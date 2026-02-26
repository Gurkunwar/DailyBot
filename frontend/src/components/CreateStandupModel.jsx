import React, { useState, useEffect, useRef } from "react";
import { DragDropContext, Droppable, Draggable } from "@hello-pangea/dnd";
import TimePicker from "./TimePicker";

export default function CreateStandupModal({ isOpen, onClose, onRefresh }) {
  const [guilds, setGuilds] = useState([]);
  const [channels, setChannels] = useState([]);
  const [errorMsg, setErrorMsg] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isChannelDropdownOpen, setIsChannelDropdownOpen] = useState(false);
  const dropdownRef = useRef(null);

  const [formData, setFormData] = useState({
    name: "",
    time: "09:00",
    guild_id: "",
    report_channel_id: "",
    questions: ["What did you do yesterday?", "What will you do today?"],
  });

  const token = localStorage.getItem("token");

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
    if (isOpen) {
      setIsLoading(true);
      setErrorMsg("");
      fetch("http://localhost:8080/api/user-guilds", {
        headers: { Authorization: `Bearer ${token}` },
      })
        .then(async (res) => {
          if (!res.ok) throw new Error(await res.text());
          return res.json();
        })
        .then((data) => {
          setGuilds(data || []);
          setIsLoading(false);
        })
        .catch((err) => {
          console.error("Failed to load guilds:", err);
          setErrorMsg(
            "Please log out and log back in to refresh your Discord session.",
          );
          setIsLoading(false);
        });
    }
  }, [isOpen, token]);

  useEffect(() => {
    if (formData.guild_id) {
      fetch(
        `http://localhost:8080/api/guild-channels?guild_id=${formData.guild_id}`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      )
        .then(async (res) => {
          if (!res.ok) throw new Error(await res.text());
          return res.json();
        })
        .then((data) => setChannels(data || []))
        .catch((err) => console.error("Failed to load channels:", err));
    } else {
      setChannels([]);
      setFormData((prev) => ({ ...prev, report_channel_id: "" }));
    }
  }, [formData.guild_id, token]);

  const handleQuestionChange = (index, value) => {
    const newQuestions = [...formData.questions];
    newQuestions[index] = value;
    setFormData({ ...formData, questions: newQuestions });
  };

  const addQuestion = () => {
    setFormData({ ...formData, questions: [...formData.questions, ""] });
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

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();

    const cleanedQuestions = formData.questions
      .map((q) => q.trim())
      .filter((q) => q.length > 0);

    if (cleanedQuestions.length === 0) {
      alert("You must have at least one valid question.");
      return;
    }
    if (!formData.report_channel_id) {
      alert("Please select a report channel.");
      return;
    }

    const payload = { ...formData, questions: cleanedQuestions };

    try {
      const res = await fetch("http://localhost:8080/api/standups/create", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        onRefresh();
        onClose();
        setFormData({
          name: "",
          time: "09:00",
          guild_id: "",
          report_channel_id: "",
          questions: ["What did you do yesterday?", "What will you do today?"],
        });
      }
    } catch (err) {
      console.error("Creation failed", err);
    }
  };

  const selectedChannelName =
    channels.find((c) => c.id === formData.report_channel_id)?.name ||
    "Select a channel...";

  return (
    <div className="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
      <div
        className="bg-[#313338] w-full max-w-2xl rounded-xl p-6 md:p-8 border border-[#1e1f22] max-h-[90vh] 
      overflow-y-auto shadow-2xl custom-scrollbar"
      >
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
              className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] 
              outline-none text-white text-sm transition-all"
              placeholder="e.g. Backend Engineering Sync"
              value={formData.name}
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              required
            />
          </div>

          {/* SERVER SELECTION */}
          <div>
            <label
              className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 flex 
            items-center justify-between"
            >
              <span>Discord Server</span>
              <span className="normal-case font-medium text-[10px] text-[#5865F2]">
                Select to fetch channels
              </span>
            </label>
            <div
              className="bg-[#1e1f22] rounded-md border border-[#1e1f22] h-40 overflow-y-auto p-2 space-y-1 
            custom-scrollbar shadow-inner"
            >
              {errorMsg ? (
                <div className="h-full flex items-center justify-center p-4 text-center">
                  <p className="text-sm text-[#da373c] font-semibold">
                    {errorMsg}
                  </p>
                </div>
              ) : isLoading ? (
                <div className="h-full flex items-center justify-center gap-2 text-[#99AAB5]">
                  <svg
                    className="animate-spin h-5 w-5 text-[#5865F2]"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    ></circle>
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 
                    12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  <span className="text-sm font-medium">
                    Fetching servers...
                  </span>
                </div>
              ) : guilds.length === 0 ? (
                <div className="h-full flex items-center justify-center">
                  <p className="text-sm text-[#99AAB5]">
                    No servers found where you have Admin permissions.
                  </p>
                </div>
              ) : (
                guilds.map((g) => {
                  const isSelected = formData.guild_id === g.id;
                  return (
                    <div
                      key={g.id}
                      onClick={() => {
                        if (g.bot_present) {
                          setFormData({
                            ...formData,
                            guild_id: g.id,
                            report_channel_id: "",
                          });
                        }
                      }}
                      className={`flex items-center justify-between p-3 rounded-md transition-all ${
                        isSelected
                          ? "bg-[#5865F2]/20 border border-[#5865F2] cursor-default"
                          : g.bot_present
                            ? "hover:bg-[#2b2d31] border border-transparent cursor-pointer"
                            : "bg-[#2b2d31]/50 border border-transparent cursor-default opacity-80"
                      }`}
                    >
                      <div className="flex items-center gap-3 overflow-hidden">
                        <div
                          className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold 
                            ${isSelected ? "bg-[#5865F2] text-white" : "bg-[#313338] text-[#99AAB5]"}`}
                        >
                          {g.name.charAt(0)}
                        </div>
                        <span
                          className={`text-sm font-semibold truncate ${
                            isSelected ? "text-white" : "text-[#99AAB5]"
                          }`}
                        >
                          {g.name}
                        </span>
                      </div>

                      {g.bot_present ? (
                        isSelected && (
                          <span className="text-xs text-[#5865F2] font-bold px-2">
                            Selected
                          </span>
                        )
                      ) : (
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            const clientId =
                              import.meta.env.VITE_DISCORD_CLIENT_ID ||
                              "YOUR_CLIENT_ID_HERE";
                            const inviteUrl = `https://discord.com/api/oauth2/authorize?client_id=${clientId}&permissions=8&scope=bot%20applications.commands&guild_id=${g.id}&disable_guild_select=true`;
                            window.open(
                              inviteUrl,
                              "_blank",
                              "width=500,height=700",
                            );
                          }}
                          className="text-[10px] px-3 py-1.5 rounded font-bold uppercase tracking-wider bg-[#23a559]
                         hover:bg-[#1d8a4a] text-white transition-colors flex items-center gap-1 shadow-md"
                        >
                          Invite Bot ↗
                        </button>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          </div>

          {/* CUSTOM CHANNEL SELECTION */}
          <div className="relative" ref={dropdownRef}>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Report Channel
            </label>

            <div
              onClick={() => {
                if (formData.guild_id && channels.length > 0) {
                  setIsChannelDropdownOpen(!isChannelDropdownOpen);
                }
              }}
              className={`w-full bg-[#1e1f22] p-3 rounded-md border flex items-center justify-between 
                text-sm transition-all ${
                  !formData.guild_id
                    ? "opacity-50 cursor-not-allowed border-transparent text-[#99AAB5]"
                    : isChannelDropdownOpen
                      ? "border-[#5865F2] cursor-pointer text-white"
                      : "border-transparent hover:border-[#404249] cursor-pointer text-white"
                }`}
            >
              <div className="flex items-center gap-2 truncate">
                {formData.report_channel_id && (
                  <span className="text-[#99AAB5] font-bold text-lg leading-none">
                    #
                  </span>
                )}
                <span className="truncate">{selectedChannelName}</span>
              </div>

              <svg
                className={`w-4 h-4 text-[#99AAB5] transition-transform ${
                  isChannelDropdownOpen ? "rotate-180" : ""
                }`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 9l-7 7-7-7"
                />
              </svg>
            </div>

            {/* Dropdown Menu */}
            {isChannelDropdownOpen && (
              <div
                className="absolute z-10 w-full mt-1 bg-[#2b2d31] border border-[#1e1f22] rounded-md
                 shadow-xl max-h-48 overflow-y-auto custom-scrollbar py-1"
              >
                {channels.map((c) => (
                  <div
                    key={c.id}
                    onClick={() => {
                      setFormData({ ...formData, report_channel_id: c.id });
                      setIsChannelDropdownOpen(false);
                    }}
                    className={`px-3 py-2 text-sm flex items-center gap-2 cursor-pointer transition-colors ${
                      formData.report_channel_id === c.id
                        ? "bg-[#404249] text-white"
                        : "text-[#99AAB5] hover:bg-[#35373c] hover:text-white"
                    }`}
                  >
                    <span className="font-bold text-lg leading-none opacity-50">
                      #
                    </span>
                    <span className="truncate">{c.name}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* DYNAMIC DRAG AND DROP QUESTIONS */}
          <div className="pt-2 border-t border-[#3f4147]">
            <label
              className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider 
            mb-3 flex justify-between items-center"
            >
              <span>Questions</span>
              <span
                className="bg-[#232428] px-2 py-0.5 rounded text-[10px] font-mono border 
              border-[#1e1f22]"
              >
                {formData.questions.length}/5
              </span>
            </label>

            <DragDropContext onDragEnd={onDragEnd}>
              <Droppable droppableId="questions-list">
                {(provided) => (
                  <div
                    {...provided.droppableProps}
                    ref={provided.innerRef}
                    className="space-y-2.5"
                  >
                    {formData.questions.map((q, index) => (
                      <Draggable
                        key={`q-${index}`}
                        draggableId={`q-${index}`}
                        index={index}
                      >
                        {(provided, snapshot) => (
                          <div
                            ref={provided.innerRef}
                            {...provided.draggableProps}
                            className={`flex items-center gap-3 bg-[#1e1f22] p-2.5 rounded-md border ${
                              snapshot.isDragging
                                ? "border-[#5865F2] shadow-xl scale-[1.02]"
                                : "border-[#1e1f22]"
                            } transition-all duration-200 group`}
                          >
                            <div
                              {...provided.dragHandleProps}
                              className="text-[#404249] hover:text-white 
                            cursor-grab active:cursor-grabbing px-1"
                            >
                              <svg
                                width="10"
                                height="16"
                                viewBox="0 0 10 16"
                                fill="currentColor"
                              >
                                <circle cx="3" cy="3" r="1.5" />
                                <circle cx="3" cy="8" r="1.5" />
                                <circle cx="3" cy="13" r="1.5" />
                                <circle cx="7" cy="3" r="1.5" />
                                <circle cx="7" cy="8" r="1.5" />
                                <circle cx="7" cy="13" r="1.5" />
                              </svg>
                            </div>
                            <input
                              type="text"
                              className="flex-1 bg-transparent outline-none text-sm text-white 
                              placeholder-[#404249]"
                              value={q}
                              placeholder={`Enter question ${index + 1}...`}
                              onChange={(e) =>
                                handleQuestionChange(index, e.target.value)
                              }
                              required
                            />
                            {formData.questions.length > 1 && (
                              <button
                                type="button"
                                onClick={() => removeQuestion(index)}
                                className="text-[#404249] hover:text-[#da373c] p-1 transition-colors 
                                md:opacity-0 md:group-hover:opacity-100"
                                title="Remove Question"
                              >
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
                                    d="M6 18L18 6M6 6l12 12"
                                  ></path>
                                </svg>
                              </button>
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
              <button
                type="button"
                onClick={addQuestion}
                className="mt-4 text-xs text-[#5865F2] hover:text-[#4752C4] font-bold flex items-center 
                gap-1 transition-colors"
              >
                <svg
                  className="w-3.5 h-3.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="3"
                    d="M12 4v16m8-8H4"
                  ></path>
                </svg>
                Add another question
              </button>
            )}
          </div>

          {/* SCHEDULE SELECTION */}
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Daily Trigger Time
            </label>
            <TimePicker
              value={formData.time}
              onChange={(newTime) =>
                setFormData({ ...formData, time: newTime })
              }
            />
          </div>

          {/* FOOTER BUTTONS */}
          <div className="flex gap-3 pt-4 mt-6 border-t border-[#3f4147]">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2.5 rounded-md font-semibold text-sm text-[#99AAB5] hover:text-white 
              hover:bg-[#2b2d31] transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="flex-1 bg-[#5865F2] hover:bg-[#4752C4] py-2.5 rounded-md font-bold text-sm 
              text-white transition-all transform active:scale-95 shadow-lg"
            >
              Create Standup
            </button>
          </div>
        </form>
      </div>

      {/* Adding a global style for the custom scrollbar within this modal */}
      <style
        dangerouslySetInnerHTML={{
          __html: `
        .custom-scrollbar::-webkit-scrollbar {
          width: 6px;
        }
        .custom-scrollbar::-webkit-scrollbar-track {
          background: transparent;
        }
        .custom-scrollbar::-webkit-scrollbar-thumb {
          background-color: #2b2d31;
          border-radius: 10px;
        }
        .custom-scrollbar:hover::-webkit-scrollbar-thumb {
          background-color: #404249;
        }
      `,
        }}
      />
    </div>
  );
}
