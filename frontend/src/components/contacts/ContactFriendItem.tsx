import React from 'react';
import type { FriendDTO, UserDTO } from '@/api/types';
import { friendDisplayName, userFromCache } from '@/hooks/chat/models/contactViewModel';
import { Avatar } from '@/components/ui';

export function ContactFriendItem({
  friend,
  userCache,
  onlineMap,
  onViewUser,
  onContextMenu,
}: {
  friend: FriendDTO;
  userCache: Record<number, UserDTO>;
  onlineMap: Record<number, boolean>;
  onViewUser?: (uid: number) => void;
  onContextMenu: (e: React.MouseEvent, friend: FriendDTO) => void;
}) {
  const user = userFromCache(friend.friend_uid, userCache);
  const name = friendDisplayName(friend, userCache);

  return (
    <div
      className="flex w-full cursor-pointer items-center gap-3 rounded-lg px-3 py-2 text-left text-inherit transition hover:bg-[#e8e8e8]"
      onClick={() => onViewUser?.(friend.friend_uid)}
      onContextMenu={(e) => onContextMenu(e, friend)}
    >
      <Avatar user={user} small online={onlineMap[friend.friend_uid] ?? false} />
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium text-[#1a1a1a]">{name}</div>
        {!friend.remark && user?.nickname && user.nickname !== name && (
          <div className="truncate text-xs text-[#b0b5be]">{user.nickname}</div>
        )}
      </div>
    </div>
  );
}
