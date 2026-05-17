import { describe, expect, it } from 'vitest';
import type { FriendDTO, FriendRequestDTO, UserDTO } from '@/api/types';
import { buildFriendMap, collectMissingFriendUserIDs, ensureFriendOnlineEntries } from '../models/friendModel';

const friends: FriendDTO[] = [
  { id: 1, friend_uid: 2, remark: 'A', group_id: 0, created_at: 1 },
  { id: 2, friend_uid: 3, remark: 'B', group_id: 1, created_at: 1 },
];

const incoming: FriendRequestDTO[] = [
  { id: 1, from_uid: 4, to_uid: 1, message: '', status: 0, created_at: 1 },
];

const outgoing: FriendRequestDTO[] = [
  { id: 2, from_uid: 1, to_uid: 5, message: '', status: 0, created_at: 1 },
];

const userCache: Record<number, UserDTO> = {
  2: { id: 2, username: 'u2', nickname: 'U2', avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 },
};

describe('friendModel', () => {
  it('按 friend_uid 构建好友映射', () => {
    expect(buildFriendMap(friends)).toEqual({ 2: friends[0], 3: friends[1] });
  });

  it('为好友在线态补默认 false 且保留已有状态', () => {
    expect(ensureFriendOnlineEntries(friends, { 2: true })).toEqual({ 2: true, 3: false });
  });

  it('收集好友和申请中缺失缓存的用户 ID 并去重', () => {
    expect(collectMissingFriendUserIDs([...friends, friends[0]], incoming, outgoing, userCache)).toEqual([3, 4, 5]);
  });
});
