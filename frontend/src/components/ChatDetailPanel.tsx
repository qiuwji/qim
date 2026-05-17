import type { ConversationDTO, FriendDTO, MemberDTO, MessageDTO, UserConvDTO } from '@/api/types';
import { chatTitle, displayName, shortName } from '@/utils';
import { SwitchRow, SearchInput } from '@/components/ui';

export function ChatDetailPanel({ chat, detail, members, currentUID, userCache, friendMap, chatSearch, chatSearchResult, onChatSearch, onChatSearchChange, onJumpToMessage, onRenameGroup, onInviteMember, onRemoveMember, onLeaveGroup, onDissolveGroup, onTogglePin, onToggleMute, onSetMemberRole, onTransferOwner, onUploadGroupAvatar, onSetMemberLimit, onViewUser, onClose }: {
  chat: UserConvDTO | null; detail?: ConversationDTO; members: MemberDTO[]; currentUID: number; userCache: Record<number, import('@/api/types').UserDTO>; friendMap?: Record<number, FriendDTO>;
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
          <SwitchRow label="置顶聊天" on={chat.is_pinned} onClick={() => onTogglePin(chat.conversation_id)} />
          <SwitchRow label="消息免打扰" on={chat.is_muted} onClick={() => onToggleMute(chat.conversation_id)} />
        </div>
      </div>
      <ChatSearchSection value={chatSearch} onChange={onChatSearchChange} onSearch={onChatSearch} results={chatSearchResult} userCache={userCache} friendMap={friendMap} onJumpToMessage={onJumpToMessage} />
      <MemberSection members={members} currentUID={currentUID} userCache={userCache} friendMap={friendMap} isGroup={isGroup} isAdmin={isAdmin} onInviteMember={onInviteMember} onViewUser={onViewUser} />
      {isGroup && (isAdmin || isOwner) && <GroupAdminSection members={members} currentUID={currentUID} userCache={userCache} friendMap={friendMap} isOwner={isOwner} isAdmin={isAdmin} onRenameGroup={onRenameGroup} onSetMemberLimit={onSetMemberLimit} onSetMemberRole={onSetMemberRole} onTransferOwner={onTransferOwner} onRemoveMember={onRemoveMember} onLeaveGroup={onLeaveGroup} onDissolveGroup={onDissolveGroup} onViewUser={onViewUser} />}
    </div>
  );
}

function ChatSearchSection({ value, onChange, onSearch, results, userCache, friendMap, onJumpToMessage }: {
  value: string; onChange: (v: string) => void; onSearch: () => void; results: MessageDTO[]; userCache: Record<number, import('@/api/types').UserDTO>; friendMap?: Record<number, FriendDTO>; onJumpToMessage: (id: number) => void;
}) {
  return (
    <div className="detail-card">
      <div className="card-title"><strong>搜索聊天记录</strong></div>
      <SearchInput value={value} onChange={onChange} placeholder="输入关键词" onSearch={onSearch} />
      {results.length > 0 && <div className="grid gap-1.5 max-h-44 overflow-auto">{results.map((m) => (
        <button key={m.id} type="button" className="grid gap-0.5 p-2 text-left rounded-lg bg-[#f3f4f6] hover:bg-[#eceff3]" onClick={() => onJumpToMessage(m.id)}>
          <strong className="text-[13px] text-[#1f2329]">{displayName(m.sender_id, userCache, friendMap)}</strong>
          <span className="text-[#858c98] text-xs overflow-hidden text-ellipsis whitespace-nowrap">{m.content.slice(0, 60)}</span>
        </button>
      ))}</div>}
    </div>
  );
}

function MemberSection({ members, currentUID, userCache, friendMap, isGroup, isAdmin, onInviteMember, onViewUser }: {
  members: MemberDTO[]; currentUID: number; userCache: Record<number, import('@/api/types').UserDTO>; friendMap?: Record<number, FriendDTO>; isGroup: boolean; isAdmin: boolean; onInviteMember: () => void; onViewUser?: (uid: number) => void;
}) {
  return (
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
  );
}

function GroupAdminSection({ members, currentUID, userCache, friendMap, isOwner, isAdmin, onRenameGroup, onSetMemberLimit, onSetMemberRole, onTransferOwner, onRemoveMember, onLeaveGroup, onDissolveGroup, onViewUser }: {
  members: MemberDTO[]; currentUID: number; userCache: Record<number, import('@/api/types').UserDTO>; friendMap?: Record<number, FriendDTO>; isOwner: boolean; isAdmin: boolean;
  onRenameGroup: () => void; onSetMemberLimit: () => void; onSetMemberRole: (uid: number, role: number) => void; onTransferOwner: (uid: number) => void; onRemoveMember: (uid: number) => void; onLeaveGroup: () => void; onDissolveGroup: () => void; onViewUser?: (uid: number) => void;
}) {
  return (
    <div className="detail-card">
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
    </div>
  );
}
