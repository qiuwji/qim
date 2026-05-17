import type { FriendDTO, MemberDTO, UserDTO } from '@/api/types';
import { displayName, shortName } from '@/utils';

export function MemberSection({ members, currentUID, userCache, friendMap, isGroup, isAdmin, onInviteMember, onViewUser }: {
  members: MemberDTO[]; currentUID: number; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; isGroup: boolean; isAdmin: boolean; onInviteMember: () => void; onViewUser?: (uid: number) => void;
}) {
  return (
    <div className="detail-card">
      <div className="card-title">
        <strong>{isGroup ? '群成员' : '聊天成员'}</strong>
        {isAdmin && <button onClick={onInviteMember}>添加</button>}
      </div>
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