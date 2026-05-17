import type { ConversationDTO, FriendDTO, MemberDTO, MessageDTO, UserConvDTO } from '../api/types';
import { chatTitle, displayName, shortName } from '../utils';

export function ChatDetailPanel({ chat, detail, members, currentUID, userCache, friendMap, chatSearch, chatSearchResult, onChatSearch, onChatSearchChange, onJumpToMessage, onRenameGroup, onInviteMember, onRemoveMember, onLeaveGroup, onDissolveGroup, onTogglePin, onToggleMute, onSetMemberRole, onTransferOwner, onUploadGroupAvatar, onSetMemberLimit, onViewUser, onClose }: {
  chat: UserConvDTO | null; detail?: ConversationDTO; members: MemberDTO[]; currentUID: number; userCache: Record<number, import('../api/types').UserDTO>; friendMap?: Record<number, FriendDTO>;
  chatSearch: string; chatSearchResult: MessageDTO[]; onChatSearch: () => void; onChatSearchChange: (v: string) => void; onJumpToMessage: (id: number) => void;
  onRenameGroup: () => void; onInviteMember: () => void; onRemoveMember: (uid: number) => void; onLeaveGroup: () => void; onDissolveGroup: () => void; onTogglePin: (convID: number) => void; onToggleMute: (convID: number) => void;
  onSetMemberRole: (uid: number, role: number) => void; onTransferOwner: (uid: number) => void; onUploadGroupAvatar: (file: File) => void; onSetMemberLimit: () => void; onViewUser?: (uid: number) => void; onClose: () => void;
}) {
  if (!chat) return (
    <div className="detail-inner">
      <div className="detail-header"><strong>聊天设置</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="detail-card"><span>选择一个聊天后查看设置。</span></div>
    </div>
  );
  const isGroup = detail?.type === 2;
  const currentMember = members.find((m) => m.uid === currentUID);
  const isOwner = isGroup && currentMember?.role === 2;
  const isAdmin = isGroup && (currentMember?.role === 1 || currentMember?.role === 2);

  return (
    <div className="detail-inner">
      <div className="detail-header"><strong>聊天设置</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="detail-card chat-profile-card">
        <div className="detail-avatar">{shortName(chatTitle(chat, detail))}</div>
        <strong>{chatTitle(chat, detail)}</strong>
        <span>{isGroup ? `${members.length} 位成员` : '私聊'}</span>
        {isAdmin && <label className="upload-btn small">上传群头像<input type="file" accept="image/*" hidden onChange={(e) => e.target.files?.[0] && onUploadGroupAvatar(e.target.files[0])} /></label>}
        <div className="switch-list">
          <div className="switch-row"><span>置顶聊天</span><button type="button" className={`switch-btn ${chat.is_pinned ? 'on' : ''}`} onClick={() => onTogglePin(chat.conversation_id)} aria-label="置顶聊天"></button></div>
          <div className="switch-row"><span>消息免打扰</span><button type="button" className={`switch-btn ${chat.is_muted ? 'on' : ''}`} onClick={() => onToggleMute(chat.conversation_id)} aria-label="消息免打扰"></button></div>
        </div>
      </div>
      <div className="detail-card">
        <div className="card-title"><strong>搜索聊天记录</strong></div>
        <div className="grid grid-cols-[1fr_auto] gap-2">
          <input className="min-w-0 border border-[#d8dde4] rounded-lg px-2.5 py-2 outline-none focus:border-[#12b35f]" value={chatSearch} onChange={(e) => onChatSearchChange(e.target.value)} placeholder="输入关键词" onKeyDown={(e) => e.key === 'Enter' && onChatSearch()} />
          <button type="button" className="px-3 py-2 rounded-lg text-white bg-[#12b35f] font-semibold" onClick={onChatSearch}>搜索</button>
        </div>
        {chatSearchResult.length > 0 && <div className="grid gap-1.5 max-h-44 overflow-auto">{chatSearchResult.map((m) => (
          <button key={m.id} type="button" className="grid gap-0.5 p-2 text-left rounded-lg bg-[#f3f4f6] hover:bg-[#eceff3]" onClick={() => onJumpToMessage(m.id)}>
            <strong className="text-[13px] text-[#1f2329]">{displayName(m.sender_id, userCache, friendMap)}</strong>
            <span className="text-[#858c98] text-xs overflow-hidden text-ellipsis whitespace-nowrap">{m.content.slice(0, 60)}</span>
          </button>
        ))}</div>}
      </div>
      <div className="detail-card">
        <div className="card-title"><strong>{isGroup ? '群成员' : '聊天成员'}</strong>{isAdmin && <button onClick={onInviteMember}>添加</button>}</div>
        <div className="member-grid">
          {members.slice(0, 12).map((member) => {
            const dn = displayName(member.uid, userCache, friendMap);
            return (
              <button key={member.uid} className="member-chip avatar-interactive" onClick={() => onViewUser?.(member.uid)}>
                <span>{member.uid === currentUID ? '我' : shortName(dn)}</span>
                <small>{member.role === 2 ? '群主' : member.role === 1 ? '管理员' : '成员'}</small>
              </button>
            );
          })}
          {!members.length && <span className="muted">暂无成员信息</span>}
        </div>
      </div>
      {isGroup && (isAdmin || isOwner) && (<div className="detail-card">
        <div className="card-title"><strong>群管理</strong></div>
        {isAdmin && <button className="wide-btn" onClick={onRenameGroup}>修改群名称</button>}
        {isAdmin && <button className="wide-btn" onClick={onSetMemberLimit}>成员上限设置</button>}
        {members.filter((m) => m.uid !== currentUID && m.role !== 2).map((m) => {
          const dn = displayName(m.uid, userCache, friendMap);
          return (
            <div key={m.uid} className="member-action-row">
              <span className="clickable" onClick={() => onViewUser?.(m.uid)}>{dn} ({m.role === 1 ? '管理员' : '成员'})</span>
              <div className="line-actions">
                {isOwner && m.role === 0 && <button onClick={() => onSetMemberRole(m.uid, 1)}>设管理员</button>}
                {isOwner && m.role === 1 && <button onClick={() => onSetMemberRole(m.uid, 0)}>取消管理员</button>}
                {isOwner && <button onClick={() => onTransferOwner(m.uid)}>转让群主</button>}
                {isAdmin && <button className="danger-text" onClick={() => onRemoveMember(m.uid)}>移除</button>}
              </div>
            </div>
          );
        })}
        {!isOwner && <button className="wide-btn danger-soft" onClick={onLeaveGroup}>退出群聊</button>}
        {isOwner && <button className="wide-btn danger" onClick={onDissolveGroup}>解散群聊</button>}
      </div>)}
    </div>
  );
}
