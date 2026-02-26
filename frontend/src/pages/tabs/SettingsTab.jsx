import React, { useState, useEffect } from "react";
import { DragDropContext, Droppable, Draggable } from "@hello-pangea/dnd";
import TimePicker from "../../components/TimePicker";

export default function SettingsTab({ standup, channels, onSave, isSaving }) {
  const [formData, setFormData] = useState({
    name: "",
    time: "09:00",
    report_channel_id: "",
    questions: [],
  });

  useEffect(() => {
    if (standup) {
      setFormData({
        name: standup.name || standup.Name || "",
        time: standup.time || standup.Time || "09:00",
        report_channel_id:
          standup.report_channel_id || standup.ReportChannelID || "",
        questions: standup.questions || standup.Questions || [],
      });
    }
  }, [standup]);

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

  const handleSubmit = (e) => {
    e.preventDefault();
    const cleanedQuestions = formData.questions
      .map((q) => q.trim())
      .filter((q) => q.length > 0);

    if (cleanedQuestions.length === 0) {
      alert("You must have at least one valid question.");
      return;
    }

    onSave({ ...formData, questions: cleanedQuestions });
  };

  return (
    <div className="animate-fade-in">
      <div className="mb-6">
        <h2 className="text-xl font-bold text-white">Standup Settings</h2>
        <p className="text-[#99AAB5] text-sm mt-1">
          Update your team's configuration and daily prompt questions.
        </p>
      </div>

      <form
        onSubmit={handleSubmit}
        className="space-y-6 bg-[#2b2d31] p-6 md:p-8 rounded-xl border 
      border-[#1e1f22] shadow-sm"
      >
        {/* TEAM NAME & TIME */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
              Team Name
            </label>
            <input
              type="text"
              className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] 
              outline-none text-white text-sm transition-all shadow-inner"
              value={formData.name}
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              required
            />
          </div>

          {/* THE NEW TIME PICKER COMPONENT */}
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
        </div>

        {/* CHANNEL SELECTION */}
        <div>
          <label className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-2 block">
            Report Channel
          </label>
          <div className="relative">
            <select
              className="w-full bg-[#1e1f22] p-3 rounded-md border border-transparent focus:border-[#5865F2] 
              outline-none text-white text-sm cursor-pointer shadow-inner appearance-none"
              value={formData.report_channel_id}
              onChange={(e) =>
                setFormData({ ...formData, report_channel_id: e.target.value })
              }
              required
            >
              <option value="" disabled>
                Select a text channel...
              </option>
              {channels.map((c) => (
                <option key={c.id} value={c.id}>
                  # {c.name}
                </option>
              ))}
            </select>
            <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-[#99AAB5]">
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
        </div>

        {/* QUESTIONS (Drag and Drop) */}
        <div className="pt-6 border-t border-[#3f4147]">
          <label
            className="text-[11px] font-extrabold text-[#99AAB5] uppercase tracking-wider mb-4 flex 
          justify-between items-center"
          >
            <span>Questions</span>
            <span className="bg-[#1e1f22] px-2 py-0.5 rounded text-[10px] font-mono border border-[#3f4147]">
              {formData.questions.length}/5
            </span>
          </label>

          <DragDropContext onDragEnd={onDragEnd}>
            <Droppable droppableId="settings-questions">
              {(provided) => (
                <div
                  {...provided.droppableProps}
                  ref={provided.innerRef}
                  className="space-y-3"
                >
                  {formData.questions.map((q, index) => (
                    <Draggable
                      key={`sq-${index}`}
                      draggableId={`sq-${index}`}
                      index={index}
                    >
                      {(provided, snapshot) => (
                        <div
                          ref={provided.innerRef}
                          {...provided.draggableProps}
                          className={`flex items-center gap-3 bg-[#1e1f22] p-3 rounded-md border ${
                            snapshot.isDragging
                              ? "border-[#5865F2] shadow-xl scale-[1.02]"
                              : "border-transparent"
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
                            className="flex-1 bg-transparent outline-none text-sm text-white placeholder-[#404249]"
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
                                className="w-5 h-5"
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
              className="mt-4 text-sm text-[#5865F2] hover:text-[#4752C4] font-bold flex items-center gap-1 
              transition-colors"
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
                  strokeWidth="3"
                  d="M12 4v16m8-8H4"
                ></path>
              </svg>
              Add another question
            </button>
          )}
        </div>

        {/* SAVE BUTTON */}
        <div className="pt-4 border-t border-[#3f4147] flex justify-end">
          <button
            type="submit"
            disabled={isSaving}
            className={`px-6 py-2.5 rounded-md font-bold text-sm text-white transition-all shadow-lg ${
              isSaving
                ? "bg-[#404249] cursor-not-allowed"
                : "bg-[#23a559] hover:bg-[#1d8a4a] transform active:scale-95"
            }`}
          >
            {isSaving ? "Saving..." : "Save Changes"}
          </button>
        </div>
      </form>
    </div>
  );
}
