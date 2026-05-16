import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO } from '../api/types';
import { chatTitle, displayName, shortName } from '../utils';

export function ChatDetailPanel({ chat, detail, members, currentUID, userCache, friendMap, onRenameGroup, onInviteMember, onRemoveMember, onLeaveGroup, onDissolveGroup, onTogglePin, onToggleMute, onSetMemberRole, onTransferOwner, onUploadGroupAvatar, onSetMemberLimit, onViewUser, onClose }: {
  chat: UserConvDTO | null; detail?: ConversationDTO; members: MemberDTO[]; currentUID: number; userCache: Record<number, import('../api/types').UserDTO>; friendMap?: Record<number, FriendDTO>;
  onRenameGroup: () => void; onInviteMember: () => void; onRemoveMember: (uid: number) => void; onLeaveGroup: () => void; onDissolveGroup: () => void; onTogglePin: () => void; onToggleMute: () => void;
  onSetMemberRole: (uid: number, role: number) => void; onTransferOwner: (uid: number) => void; onUploadGroupAvatar: (file: File) => void; onSetMemberLimit: () => void; onViewUser?: (uid: number) => void; onClose: () => void;
}) {
  if (!chat) return (
    <div className="detail-inner">
      <div className="detail-header"><strong>聊天设置</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="detail-card"><span>选择一个聊天后查看设置。</span></div>
    </div>
  );
  const isGroup = detail?.type === 2;
  const isOwner = isGroup && members.find((m) => m.uid === currentUID)?.role === 2;

  return (
    <div className="detail-inner">
      <div className="detail-header"><strong>聊天设置</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="detail-card chat-profile-card">
        <div className="detail-avatar">{shortName(chatTitle(chat, detail))}</div>
        <strong>{chatTitle(chat, detail)}</strong>
        <span>{isGroup ? `${members.length} 位成员` : '私聊'}</span>
        {isGroup && <label className="upload-btn small">上传群头像<input type="file" accept="image/*" hidden onChange={(e) => e.target.files?.[0] && onUploadGroupAvatar(e.target.files[0])} /></label>}
        <div className="switch-list">
          <label><span>置顶聊天</span><button className={`switch-btn ${chat.is_pinned ? 'on' : ''}`} onClick={onTogglePin}></button></label>
          <label><span>消息免打扰</span><button className={`switch-btn ${chat.is_muted ? 'on' : ''}`} onClick={onToggleMute}></button></label>
        </div>
      </div>
      <div className="detail-card">
        <div className="card-title"><strong>{isGroup ? '群成员' : '聊天成员'}</strong>{isGroup && <button onClick={onInviteMember}>添加</button>}</div>
        <div className="member-grid">
          {members.slice(0, 12).map((member) => {
            const mu = userCache[member.uid];
            const dn = displayName(member.uid, userCache, friendMap);
            return (
              <button key={member.uid} className="member-chip avatar-interactive" onClick={() => {
                if (member.uid === currentUID) { onViewUser?.(member.uid); return; }
                if (isGroup && isOwner) {
                  if (member.role === 0) onSetMemberRole(member.uid, 1);
                  else if (member.role === 1) onSetMemberRole(member.uid, 0);
                } else { onViewUser?.(member.uid); }
              }}>
                <span>{member.uid === currentUID ? '我' : shortName(dn)}</span>
                <small>{member.role === 2 ? '群主' : member.role === 1 ? '管理员' : '成员'}</small>
              </button>
            );
          })}
          {!members.length && <span className="muted">暂无成员信息</span>}
        </div>
      </div>
      {isGroup && (<div className="detail-card">
        <div className="card-title"><strong>群管理</strong></div>
        <button className="wide-btn" onClick={onRenameGroup}>修改群名称</button>
        <button className="wide-btn" onClick={onSetMemberLimit}>成员上限设置</button>
        {isOwner && members.filter((m) => m.uid !== currentUID).map((m) => {
          const mu = userCache[m.uid];
          const dn = displayName(m.uid, userCache, friendMap);
          return (
            <div key={m.uid} className="member-action-row">
              <span className="clickable" onClick={() => onViewUser?.(m.uid)}>{dn} ({m.role === 1 ? '管理员' : '成员'})</span>
              <div className="line-actions">
                {m.role === 0 && <button onClick={() => onSetMemberRole(m.uid, 1)}>设管理员</button>}
                {m.role === 1 && <button onClick={() => onSetMemberRole(m.uid, 0)}>取消管理员</button>}
                <button onClick={() => onTransferOwner(m.uid)}>转让群主</button>
                <button className="danger-text" onClick={() => onRemoveMember(m.uid)}>移除</button>
              </div>
            </div>
          );
        })}
        <button className="wide-btn danger-soft" onClick={onLeaveGroup}>退出群聊</button>
        {isOwner && <button className="wide-btn danger" onClick={onDissolveGroup}>解散群聊</button>}
      </div>)}
    </div>
  );
}
