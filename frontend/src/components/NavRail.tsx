import type { UserDTO } from '../api/types';
import type { MainTab } from '../types';
import { Avatar } from './Avatar';

export function NavRail({ user, tab, onTab, onLogout, unreadTotal, onMarkAllRead }: {
  user: UserDTO; tab: MainTab; onTab: (t: MainTab) => void; onLogout: () => void; unreadTotal: number; onMarkAllRead: () => void;
}) {
  return (
    <aside className="nav-rail">
      <Avatar user={user} />
      <button className={`nav-tab ${tab === 'chats' ? 'active' : ''}`} onClick={() => onTab('chats')}>
        消息{unreadTotal > 0 && <b className="nav-badge">{unreadTotal > 99 ? '99+' : unreadTotal}</b>}
      </button>
      <button className={`nav-tab ${tab === 'contacts' ? 'active' : ''}`} onClick={() => onTab('contacts')}>通讯录</button>
      <button className={`nav-tab ${tab === 'profile' ? 'active' : ''}`} onClick={() => onTab('profile')}>我</button>
      {unreadTotal > 0 && <button className="nav-action" onClick={onMarkAllRead} title="全部已读">已读</button>}
      <button className="logout" onClick={onLogout}>退出</button>
    </aside>
  );
}
