import React, { useState, useEffect, useRef } from "react";

export default function TimePicker({ value, onChange }) {
  const [isOpen, setIsOpen] = useState(false);
  const [tempTime, setTempTime] = useState({ h: "09", m: "00", ap: "AM" });
  const containerRef = useRef(null);

  const hoursRef = useRef(null);
  const minutesRef = useRef(null);

  const hoursArr = [
    "12",
    "01",
    "02",
    "03",
    "04",
    "05",
    "06",
    "07",
    "08",
    "09",
    "10",
    "11",
  ];
  const minutesArr = Array.from({ length: 60 }, (_, i) =>
    i.toString().padStart(2, "0"),
  );
  const ampmArr = ["AM", "PM"];

  useEffect(() => {
    function handleClickOutside(event) {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target)
      ) {
        setIsOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleOpen = () => {
    const [h24, m] = (value || "09:00").split(":");
    const h = parseInt(h24, 10);
    const ap = h >= 12 ? "PM" : "AM";
    const h12 = h % 12 || 12;

    setTempTime({
      h: h12.toString().padStart(2, "0"),
      m: m || "00",
      ap: ap,
    });
    setIsOpen(true);
  };

  useEffect(() => {
    if (isOpen) {
      setTimeout(() => {
        if (hoursRef.current) {
          const selectedHour = hoursRef.current.querySelector(".selected-time");
          if (selectedHour)
            selectedHour.scrollIntoView({
              block: "center",
              behavior: "instant",
            });
        }
        if (minutesRef.current) {
          const selectedMin =
            minutesRef.current.querySelector(".selected-time");
          if (selectedMin)
            selectedMin.scrollIntoView({
              block: "center",
              behavior: "instant",
            });
        }
      }, 10);
    }
  }, [isOpen]);

  const handleSave = () => {
    let h24 = parseInt(tempTime.h, 10);
    if (tempTime.ap === "PM" && h24 !== 12) h24 += 12;
    if (tempTime.ap === "AM" && h24 === 12) h24 = 0;

    const finalTime = `${h24.toString().padStart(2, "0")}:${tempTime.m}`;
    onChange(finalTime);
    setIsOpen(false);
  };

  const formatTimeLabel = (time24) => {
    if (!time24) return "Select Time";
    const [h, m] = time24.split(":");
    const hour = parseInt(h, 10);
    const ampm = hour >= 12 ? "PM" : "AM";
    const hour12 = hour % 12 || 12;
    return `${hour12.toString().padStart(2, "0")}:${m} ${ampm}`;
  };

  return (
    <div className="relative w-full" ref={containerRef}>
      <div
        onClick={handleOpen}
        className={`w-full bg-[#1e1f22] p-3 rounded-md border flex items-center justify-between text-sm 
            transition-all shadow-inner cursor-pointer ${
              isOpen
                ? "border-[#5865F2] text-white"
                : "border-transparent hover:border-[#404249] text-white"
            }`}
      >
        <div className="flex items-center gap-2">
          <svg
            className="w-4 h-4 text-[#99AAB5]"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
            ></path>
          </svg>
          <span className="font-semibold text-[15px]">
            {formatTimeLabel(value)}
          </span>
        </div>
      </div>

      {isOpen && (
        <div
          className="absolute z-50 left-0 w-full mt-2 bg-[#2b2d31] border border-[#1e1f22] rounded-lg 
        shadow-2xl overflow-hidden flex flex-col"
        >
          <div className="px-4 py-3 border-b border-[#1e1f22] flex justify-between items-center bg-[#232428]">
            <p className="text-white text-[22px] font-bold">
              {tempTime.h}:{tempTime.m} {tempTime.ap}
            </p>
            <svg
              className="w-5 h-5 text-[#99AAB5]"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="2"
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </div>

          <div className="flex h-55 bg-[#2b2d31]">
            <div
              className="flex-1 overflow-y-auto no-scrollbar border-r border-[#1e1f22] py-22.5"
              ref={hoursRef}
            >
              {hoursArr.map((h) => (
                <div
                  key={`h-${h}`}
                  onClick={() => setTempTime({ ...tempTime, h })}
                  className={`h-11 flex items-center justify-center cursor-pointer transition-colors ${
                    tempTime.h === h
                      ? "text-white text-xl font-bold bg-[#1e1f22] selected-time"
                      : "text-[#99AAB5] text-base hover:text-white"
                  }`}
                >
                  {h}
                </div>
              ))}
            </div>

            <div
              className="flex-1 overflow-y-auto no-scrollbar border-r border-[#1e1f22] py-22.5"
              ref={minutesRef}
            >
              {minutesArr.map((m) => (
                <div
                  key={`m-${m}`}
                  onClick={() => setTempTime({ ...tempTime, m })}
                  className={`h-11 flex items-center justify-center cursor-pointer transition-colors ${
                    tempTime.m === m
                      ? "text-white text-xl font-bold bg-[#1e1f22] selected-time"
                      : "text-[#99AAB5] text-base hover:text-white"
                  }`}
                >
                  {m}
                </div>
              ))}
            </div>

            <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col justify-center">
              {ampmArr.map((ap) => (
                <div
                  key={`ap-${ap}`}
                  onClick={() => setTempTime({ ...tempTime, ap })}
                  className={`h-14 flex items-center justify-center cursor-pointer transition-colors ${
                    tempTime.ap === ap
                      ? "text-white text-xl font-bold bg-[#1e1f22]"
                      : "text-[#99AAB5] text-base hover:text-white"
                  }`}
                >
                  {ap}
                </div>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-6 p-4 border-t border-[#1e1f22] bg-[#2b2d31]">
            <button
              type="button"
              onClick={() => setIsOpen(false)}
              className="text-[#99AAB5] 
            hover:text-white text-sm font-bold uppercase tracking-wider"
            >
              CANCEL
            </button>
            <button
              type="button"
              onClick={handleSave}
              className="text-[#5865F2] hover:text-[#4752C4] 
            text-sm font-bold uppercase tracking-wider"
            >
              OK
            </button>
          </div>
        </div>
      )}

      <style
        dangerouslySetInnerHTML={{
          __html: `
        .no-scrollbar::-webkit-scrollbar { display: none; }
        .no-scrollbar { -ms-overflow-style: none; scrollbar-width: none; }
      `,
        }}
      />
    </div>
  );
}
