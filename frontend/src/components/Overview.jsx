import { useGetDashboardStatsQuery } from '../store/apiSlice';

export default function Overview() {
  const { data: stats, isLoading } = useGetDashboardStatsQuery();

  if (isLoading) return <div className="p-8 text-[#99AAB5]">Loading stats...</div>;

  const cards = [
    { label: "Active Teams", value: stats?.total_teams, icon: "👥" },
    { label: "Managed Members", value: stats?.total_members, icon: "👤" },
    { label: "Reports (Last 7d)", value: stats?.recent_reports, icon: "📜" },
  ];

  return (
    <div className="p-8 animate-fade-in">
      <h1 className="text-2xl font-extrabold text-white mb-8">Performance Overview</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {cards.map((card) => (
          <div key={card.label} className="bg-[#2b2d31] p-6 rounded-xl border border-[#1e1f22] shadow-sm">
            <div className="text-xs font-extrabold text-[#99AAB5] uppercase tracking-widest mb-2">
               {card.label}
            </div>
            <div className="flex items-center justify-between">
              <span className="text-3xl font-black text-white">{card.value || 0}</span>
              <span className="text-2xl">{card.icon}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}