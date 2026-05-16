import type { UserDTO } from '../api/types';
import type { MainTab } from '../types';
import { Avatar } from './Avatar';

export function NavRail({ user, tab, onTab, onLogout, unreadTotal, onMarkAllRead }: {
  user: UserDTO; tab: MainTab; onTab: (t: MainTab) => void; onLogout: () => void; unreadTotal: number; onMarkAllRead: () => void;
}) {
  return (
    <aside className="nav-rail">
      <div className="nav-avatar">
        <Avatar user={user} />
      </div>
      <div className="nav-tabs">
        <button className={`nav-tab ${tab === 'chats' ? 'active' : ''}`} onClick={() => onTab('chats')} title="消息">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></svg>
          {unreadTotal > 0 && <b className="nav-badge">{unreadTotal > 99 ? '99+' : unreadTotal}</b>}
        </button>
        <button className={`nav-tab ${tab === 'contacts' ? 'active' : ''}`} onClick={() => onTab('contacts')} title="通讯录">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>
        </button>
        <button className={`nav-tab ${tab === 'profile' ? 'active' : ''}`} onClick={() => onTab('profile')} title="我">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></svg>
        </button>
      </div>
      <div className="nav-bottom">
        {unreadTotal > 0 && <button className="nav-action-btn" onClick={onMarkAllRead} title="全部已读">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
        </button>}
        <button className="nav-logout" onClick={onLogout} title="退出登录">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" /></svg>
        </button>
      </div>
    </aside>
  );
}
