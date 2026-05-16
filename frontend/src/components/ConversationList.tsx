import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO, UserDTO } from '../api/types';
import { chatTitle, displayName, shortName, timeText } from '../utils';
import { Avatar } from './Avatar';
import { EmptyState } from './EmptyState';

export function ConversationList({ conversations, details, lastMsgMap, selectedID, onSelect, members, userCache, onlineMap, friendMap, currentUID }: {
  conversations: UserConvDTO[]; details: Record<number, ConversationDTO>; lastMsgMap: Record<number, string>; selectedID: number | null; onSelect: (id: number) => void;
  members: Record<number, MemberDTO[]>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>; currentUID: number;
}) {
  if (!conversations.length) return <EmptyState title="暂无聊天" text="可以从通讯录搜索用户并创建单聊，或点击 + 创建群聊。" />;
  return (
    <div className="conversation-list">
      {conversations.map((item) => {
        const detail = details[item.conversation_id];
        const title = chatTitle(item, detail);
        const lastMsg = lastMsgMap[item.conversation_id];
        const isPrivate = detail?.type === 1;
        const convMembers = members[item.conversation_id] ?? [];
        const peer = isPrivate ? convMembers.find((m) => m.uid !== currentUID) : undefined;
        const peerUser = peer ? userCache[peer.uid] : undefined;
        const peerOnline = peer ? onlineMap[peer.uid] : undefined;

        return (
          <button key={item.conversation_id} className={`conversation-item ${selectedID === item.conversation_id ? 'active' : ''}`} onClick={() => onSelect(item.conversation_id)}>
            {peerUser
              ? <Avatar user={peerUser} online={peerOnline} badge={item.unread_count > 0 ? item.unread_count : undefined} />
              : <div className="avatar-wrap"><div className="avatar fallback">{shortName(title)}</div>{item.unread_count > 0 && <b className="avatar-badge">{item.unread_count > 99 ? '99+' : item.unread_count}</b>}</div>
            }
            <div className="conversation-main">
              <div className="row between">
                <strong>{item.is_pinned && <span className="pin-icon">📌</span>}{title}</strong>
                <time>{timeText(item.last_msg_at)}</time>
              </div>
              <div className="row between">
                <span className="last-msg">{lastMsg || '暂无消息'}</span>
                {item.unread_count > 0 && <b className={`badge ${item.is_muted ? 'muted' : ''}`}>{item.unread_count}</b>}
              </div>
            </div>
          </button>
        );
      })}
    </div>
  );
}
