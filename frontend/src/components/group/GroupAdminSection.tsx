import type { FriendDTO, MemberDTO, UserDTO } from '@/api/types';
import { displayName } from '@/utils';

export function GroupAdminSection({ members, currentUID, userCache, friendMap, isOwner, isAdmin, onRenameGroup, onSetMemberLimit, onSetMemberRole, onTransferOwner, onRemoveMember, onLeaveGroup, onDissolveGroup, onViewUser }: {
  members: MemberDTO[]; currentUID: number; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; isOwner: boolean; isAdmin: boolean;
  onRenameGroup: () => void; onSetMemberLimit: () => void; onSetMemberRole: (uid: number, role: number) => void; onTransferOwner: (uid: number) => void; onRemoveMember: (uid: number) => void; onLeaveGroup: () => void; onDissolveGroup: () => void; onViewUser?: (uid: number) => void;
}) {
  const otherMembers = members.filter((m) => m.uid !== currentUID && m.role !== 2);

  return (
    <div className="detail-card">
      <div className="card-title"><strong>群管理</strong></div>
      {isAdmin && <button className="wide-btn" onClick={onRenameGroup}>修改群名称</button>}
      {isAdmin && <button className="wide-btn" onClick={onSetMemberLimit}>成员上限设置</button>}
      {otherMembers.map((m) => {
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