import type { FriendDTO, FriendGroupDTO, FriendRequestDTO, UserDTO } from '@/api/types';

export function userFromCache(uid: number, userCache: Record<number, UserDTO>): UserDTO {
  return userCache[uid] ?? {
    id: uid,
    username: String(uid),
    nickname: `用户 ${uid}`,
    avatar: '',
    sign: '',
    status: 0,
    created_at: 0,
    last_online_at: 0,
  };
}

export function friendDisplayName(friend: FriendDTO, userCache: Record<number, UserDTO>): string {
  const user = userFromCache(friend.friend_uid, userCache);
  return friend.remark || user.nickname || user.username || `用户 ${friend.friend_uid}`;
}

export function groupFriendsByGroup(friends: FriendDTO[]): Record<number, FriendDTO[]> {
  const map: Record<number, FriendDTO[]> = {};
  for (const friend of friends) {
    const gid = friend.group_id || 0;
    if (!map[gid]) map[gid] = [];
    map[gid].push(friend);
  }
  return map;
}

export function sortFriendGroups(groups: FriendGroupDTO[]): FriendGroupDTO[] {
  return [...groups].sort((a, b) => a.sort_order - b.sort_order);
}

export function matchFriendKeyword(friend: FriendDTO, keyword: string, userCache: Record<number, UserDTO>): boolean {
  if (!keyword) return true;
  const lower = keyword.toLowerCase();
  if (friend.remark && friend.remark.toLowerCase().includes(lower)) return true;
  const user = userFromCache(friend.friend_uid, userCache);
  return Boolean(user.nickname?.toLowerCase().includes(lower) || user.username?.toLowerCase().includes(lower));
}

export function filterFriendsByKeyword(
  friendsByGroup: Record<number, FriendDTO[]>,
  keyword: string,
  userCache: Record<number, UserDTO>,
): Record<number, FriendDTO[]> {
  if (!keyword) return friendsByGroup;
  const result: Record<number, FriendDTO[]> = {};
  for (const [gid, list] of Object.entries(friendsByGroup)) {
    const filtered = list.filter((friend) => matchFriendKeyword(friend, keyword, userCache));
    if (filtered.length > 0) result[Number(gid)] = filtered;
  }
  return result;
}

export function friendRequestTimeline(incoming: FriendRequestDTO[], outgoing: FriendRequestDTO[]): {
  type: 'incoming' | 'outgoing';
  req: FriendRequestDTO;
  uid: number;
}[] {
  const items = [
    ...incoming.map((req) => ({ type: 'incoming' as const, req, uid: req.from_uid })),
    ...outgoing.map((req) => ({ type: 'outgoing' as const, req, uid: req.to_uid })),
  ];
  return items.sort((a, b) => b.req.created_at - a.req.created_at);
}
