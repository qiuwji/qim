import { useEffect, useRef, useState } from 'react';
import type { FriendDTO, UserDTO } from '../api/types';
import { shortName } from '../utils';
import { Avatar } from './Avatar';

export type HoverCardData = { user: UserDTO; rect: DOMRect } | null;

export function UserCard({ data, isFriend, onStartPrivate, onAddFriend, onViewProfile }: {
  data: HoverCardData; isFriend?: boolean;
  onStartPrivate: (uid: number) => void;
  onAddFriend?: (uid: number) => void;
  onViewProfile: (uid: number) => void;
}) {
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [pos, setPos] = useState<{ left: number; top: number }>({ left: 0, top: 0 });

  useEffect(() => {
    if (!data || !cardRef.current) return;
    const card = cardRef.current.getBoundingClientRect();
    const gap = 8;
    let left = data.rect.right + gap;
    let top = data.rect.top;
    if (left + card.width > window.innerWidth - 12) {
      left = data.rect.left - card.width - gap;
    }
    if (left < 12) left = 12;
    if (top + card.height > window.innerHeight - 12) {
      top = window.innerHeight - card.height - 12;
    }
    if (top < 12) top = 12;
    setPos({ left, top });
  }, [data]);

  if (!data) return null;
  const u = data.user;

  return (
    <div ref={cardRef} className="user-card" style={{ left: pos.left, top: pos.top }}>
      <div className="user-card-header">
        <Avatar user={u} />
        <div className="user-card-info">
          <strong>{u.nickname || u.username}</strong>
          <span>@{u.username}</span>
          <span className="user-card-id">ID: {u.id}</span>
        </div>
      </div>
      {u.sign && <p className="user-card-sign">{u.sign}</p>}
      <div className="user-card-actions">
        {isFriend && <button className="primary-btn" onClick={() => onStartPrivate(u.id)}>发消息</button>}
        {!isFriend && onAddFriend && <button className="primary-btn" onClick={() => onAddFriend(u.id)}>加好友</button>}
        <button className="wide-btn" onClick={() => onViewProfile(u.id)}>查看资料</button>
      </div>
    </div>
  );
}
