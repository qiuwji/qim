import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle, displayName } from '@/utils';

export function isPrivateConversation(detail?: ConversationDTO): boolean {
  return detail?.type === 1;
}

export function isGroupConversation(detail?: ConversationDTO): boolean {
  return detail?.type === 2;
}

export function getConversationPeer(members: MemberDTO[], currentUID: number): MemberDTO | undefined {
  return members.find((member) => member.uid !== currentUID);
}

export function fallbackUser(uid: number, name?: string, avatar = ''): UserDTO {
  return {
    id: uid,
    username: name || String(uid),
    nickname: name || `用户 ${uid}`,
    avatar,
    sign: '',
    status: 0,
    created_at: 0,
    last_online_at: 0,
  };
}

export function memberDisplayUser(
  uid: number,
  currentUID: number,
  userCache: Record<number, UserDTO>,
  friendMap?: Record<number, FriendDTO>,
): UserDTO {
  const name = uid === currentUID ? '我' : displayName(uid, userCache, friendMap);
  const cached = userCache[uid];
  return cached ? { ...cached, nickname: uid === currentUID ? '我' : cached.nickname } : fallbackUser(uid, name);
}

export function conversationAvatarUser(conv: UserConvDTO, detail?: ConversationDTO): UserDTO {
  const title = chatTitle(conv, detail);
  return fallbackUser(detail?.id ?? conv.conversation_id, title, detail?.avatar ?? '');
}

export function conversationMemberText(detail: ConversationDTO | undefined, memberCount: number): string {
  return isGroupConversation(detail) ? `${memberCount} 位成员` : '私聊';
}

export function conversationSubtitle({
  detail,
  members,
  currentUID,
  userCache,
  friendMap,
  onlineMap,
  typingText,
}: {
  detail?: ConversationDTO;
  members: MemberDTO[];
  currentUID: number;
  userCache: Record<number, UserDTO>;
  friendMap?: Record<number, FriendDTO>;
  onlineMap: Record<number, boolean>;
  typingText?: string;
}): string {
  if (typingText) return typingText;
  if (isPrivateConversation(detail)) {
    const peer = getConversationPeer(members, currentUID);
    return peer ? `${displayName(peer.uid, userCache, friendMap)} · ${onlineMap[peer.uid] ? '在线' : '离线'}` : '私聊';
  }
  return `${members.length} 位成员`;
}

export function currentUserGroupRole(members: MemberDTO[], currentUID: number): number | undefined {
  return members.find((member) => member.uid === currentUID)?.role;
}

export function canManageGroup(detail: ConversationDTO | undefined, members: MemberDTO[], currentUID: number): {
  isGroup: boolean;
  isOwner: boolean;
  isAdmin: boolean;
} {
  const isGroup = isGroupConversation(detail);
  const role = currentUserGroupRole(members, currentUID);
  return {
    isGroup,
    isOwner: isGroup && role === 2,
    isAdmin: isGroup && (role === 1 || role === 2),
  };
}
