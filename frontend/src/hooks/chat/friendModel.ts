import type { FriendDTO, FriendRequestDTO, UserDTO } from '../../api/types';

export function buildFriendMap(friends: FriendDTO[]): Record<number, FriendDTO> {
  const map: Record<number, FriendDTO> = {};
  for (const friend of friends) {
    map[friend.friend_uid] = friend;
  }
  return map;
}

export function ensureFriendOnlineEntries(
  friends: FriendDTO[],
  onlineMap: Record<number, boolean>,
): Record<number, boolean> {
  const next = { ...onlineMap };
  for (const friend of friends) {
    if (next[friend.friend_uid] === undefined) {
      next[friend.friend_uid] = false;
    }
  }
  return next;
}

export function collectMissingFriendUserIDs(
  friendList: FriendDTO[],
  incomingReqs: FriendRequestDTO[],
  outgoingReqs: FriendRequestDTO[],
  userCache: Record<number, UserDTO>,
): number[] {
  const uids = new Set<number>();
  for (const friend of friendList) uids.add(friend.friend_uid);
  for (const request of incomingReqs) uids.add(request.from_uid);
  for (const request of outgoingReqs) uids.add(request.to_uid);
  return [...uids].filter((uid) => !userCache[uid]);
}
