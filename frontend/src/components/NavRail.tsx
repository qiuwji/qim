import type { UserDTO } from '@/api/types';
import type { MainTab } from '@/types';
import { Avatar, Badge } from '@/components/ui';

export function NavRail({ user, tab, onTab, onLogout, unreadTotal, onMarkAllRead }: {
  user: UserDTO; tab: MainTab; onTab: (t: MainTab) => void; onLogout: () => void; unreadTotal: number; onMarkAllRead: () => void;
}) {
  const tabButton = (active: boolean) => `relative grid h-11 w-11 place-items-center rounded-[10px] bg-transparent transition ${active ? 'bg-[rgba(7,193,96,0.1)] text-[#07c160]' : 'text-[#8c8c8c] hover:bg-white/[0.06] hover:text-[#b0b0b0]'}`;
  return (
    <aside className="nav-rail flex flex-col items-center bg-[#262626] px-0 pt-4 pb-3 text-[#8c8c8c] [-webkit-app-region:drag]">
      <div className="nav-avatar mb-5 [-webkit-app-region:no-drag]">
        <Avatar user={user} />
      </div>
      <div className="nav-tabs flex flex-1 flex-col items-center gap-1">
        <button className={tabButton(tab === 'chats')} onClick={() => onTab('chats')} title="消息">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></svg>
          {unreadTotal > 0 && <span className="absolute top-1 right-0.5"><Badge count={unreadTotal} /></span>}
        </button>
        <button className={tabButton(tab === 'contacts')} onClick={() => onTab('contacts')} title="通讯录">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>
        </button>
        <button className={tabButton(tab === 'profile')} onClick={() => onTab('profile')} title="我">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></svg>
        </button>
      </div>
      <div className="nav-bottom mt-auto flex flex-col items-center gap-1">
        {unreadTotal > 0 && <button className="grid h-11 w-11 place-items-center rounded-[10px] bg-transparent text-[#666] hover:bg-[rgba(7,193,96,0.1)] hover:text-[#07c160]" onClick={onMarkAllRead} title="全部已读">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
        </button>}
        <button className="grid h-11 w-11 place-items-center rounded-[10px] bg-transparent text-[#666] hover:bg-[rgba(245,108,108,0.1)] hover:text-[#f56c6c]" onClick={onLogout} title="退出登录">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" /></svg>
        </button>
      </div>
    </aside>
  );
}
