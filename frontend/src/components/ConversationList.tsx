import { useState } from 'react';
import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO, UserDTO } from '../api/types';
import { chatTitle, shortName, timeText } from '../utils';
import { Avatar } from './Avatar';
import { EmptyState } from './EmptyState';

export function ConversationList({ conversations, details, lastMsgMap, selectedID, onSelect, members, userCache, onlineMap, friendMap, currentUID, onTogglePin, onToggleMute }: {
  conversations: UserConvDTO[]; details: Record<number, ConversationDTO>; lastMsgMap: Record<number, string>; selectedID: number | null; onSelect: (id: number) => void;
  members: Record<number, MemberDTO[]>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>; currentUID: number;
  onTogglePin: (convID?: number) => void; onToggleMute: (convID?: number) => void;
}) {
  const [ctx, setCtx] = useState<{ x: number; y: number; item: UserConvDTO } | null>(null);

  if (!conversations.length) return <EmptyState title="暂无聊天" text="可以从通讯录搜索用户并创建单聊，或点击 + 创建群聊。" />;
  return (
    <div className="conversation-list" onClick={() => setCtx(null)}>
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
          <button key={item.conversation_id} className={`conversation-item ${selectedID === item.conversation_id ? 'active' : ''}`} onClick={() => onSelect(item.conversation_id)} onContextMenu={(e) => { e.preventDefault(); setCtx({ x: e.clientX, y: e.clientY, item }); }}>
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
      {ctx && <ConvContextMenu x={ctx.x} y={ctx.y} item={ctx.item} onClose={() => setCtx(null)} onTogglePin={onTogglePin} onToggleMute={onToggleMute} />}
    </div>
  );
}

function ConvContextMenu({ x, y, item, onClose, onTogglePin, onToggleMute }: { x: number; y: number; item: UserConvDTO; onClose: () => void; onTogglePin: (id?: number) => void; onToggleMute: (id?: number) => void }) {
  return (
    <div className="ctx-overlay" onClick={onClose}>
      <ul className="ctx-menu" style={{ left: x, top: y }} onClick={(e) => e.stopPropagation()}>
        <li onClick={() => { onTogglePin(item.conversation_id); onClose(); }}>{item.is_pinned ? '取消置顶' : '置顶会话'}</li>
        <li onClick={() => { onToggleMute(item.conversation_id); onClose(); }}>{item.is_muted ? '取消免打扰' : '消息免打扰'}</li>
      </ul>
    </div>
  );
}
