import { useState } from 'react';
import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle, timeText } from '@/utils';
import { Badge, ContextMenu } from '@/components/ui';
import { ConversationAvatar } from '@/components/conversation/ConversationAvatar';

export function ConversationList({ conversations, details, lastMsgMap, selectedID, onSelect, members, userCache, onlineMap, friendMap: _friendMap, currentUID, mentionMap, onTogglePin, onToggleMute, onHide }: {
  conversations: UserConvDTO[]; details: Record<number, ConversationDTO>; lastMsgMap: Record<number, string>; selectedID: number | null; onSelect: (id: number) => void;
  members: Record<number, MemberDTO[]>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>; currentUID: number; mentionMap?: Record<number, boolean>;
  onTogglePin: (convID?: number) => void; onToggleMute: (convID?: number) => void; onHide: (convID?: number) => void;
}) {
  const [ctx, setCtx] = useState<{ x: number; y: number; item: UserConvDTO } | null>(null);

  if (!conversations.length) return (
    <div className="m-auto grid place-items-center gap-2 p-8 text-center text-[#858c98]">
      <strong className="text-base font-semibold text-[#555f6d]">暂无聊天</strong>
      <span className="text-sm">可以从通讯录搜索用户并创建单聊，或点击 + 创建群聊。</span>
    </div>
  );

  return (
    <div className="overflow-auto px-2 py-1" onClick={() => setCtx(null)}>
      {conversations.map((item) => {
        const detail = details[item.conversation_id];
        const title = chatTitle(item, detail);
        const lastMsg = lastMsgMap[item.conversation_id];
        const convMembers = members[item.conversation_id] ?? [];

        return (
          <button key={item.conversation_id} className={`flex w-full items-center gap-3 rounded-lg px-2.5 py-3 text-left text-inherit transition hover:bg-[#e8e8e8] max-[760px]:px-2 max-[760px]:py-[9px] ${selectedID === item.conversation_id ? 'bg-[#d4edda]' : 'bg-transparent'}`} onClick={() => onSelect(item.conversation_id)} onContextMenu={(e) => { e.preventDefault(); setCtx({ x: e.clientX, y: e.clientY, item }); }}>
            <ConversationAvatar conversation={item} detail={detail} members={convMembers} currentUID={currentUID} userCache={userCache} onlineMap={onlineMap} badge={item.unread_count > 0 ? item.unread_count : undefined} />
            <div className="grid min-w-0 flex-1 gap-1">
              <div className="flex min-w-0 items-center justify-between gap-2">
                <strong className="min-w-0 truncate text-[15px] font-semibold text-[#1f2329]">{item.is_pinned && <span className="mr-0.5 text-xs">📌</span>}{title}</strong>
                <time className="shrink-0 text-xs text-[#999]">{timeText(item.last_msg_at)}</time>
              </div>
              <div className="flex min-w-0 items-center justify-between gap-2">
                <span className="min-w-0 flex-1 truncate text-[13px] text-[#858c98]">{mentionMap?.[item.conversation_id] && item.unread_count > 0 ? <span className="mention-badge">[有人@我]</span> : null}{lastMsg || '暂无消息'}</span>
                <Badge count={item.unread_count} muted={item.is_muted} />
              </div>
            </div>
          </button>
        );
      })}
      {ctx && <ContextMenu x={ctx.x} y={ctx.y} onClose={() => setCtx(null)} items={[
        { label: ctx.item.is_pinned ? '取消置顶' : '置顶会话', action: () => onTogglePin(ctx.item.conversation_id) },
        { label: ctx.item.is_muted ? '取消免打扰' : '消息免打扰', action: () => onToggleMute(ctx.item.conversation_id) },
        { label: '从列表移除', action: () => onHide(ctx.item.conversation_id), danger: true },
      ]} />}
    </div>
  );
}
