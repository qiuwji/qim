import type { ConversationDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import { conversationAvatarUser, getConversationPeer, isPrivateConversation } from '@/hooks/chat/models/conversationViewModel';
import { Avatar } from '@/components/ui';

export function ConversationAvatar({ conversation, detail, members, currentUID, userCache, onlineMap, badge, large = false, small = false }: {
  conversation: UserConvDTO;
  detail?: ConversationDTO;
  members?: MemberDTO[];
  currentUID?: number;
  userCache?: Record<number, UserDTO>;
  onlineMap?: Record<number, boolean>;
  badge?: number;
  large?: boolean;
  small?: boolean;
}) {
  if (isPrivateConversation(detail) && members && currentUID && userCache) {
    const peer = getConversationPeer(members, currentUID);
    const peerUser = peer ? userCache[peer.uid] : undefined;
    if (peer && peerUser) {
      return <Avatar user={peerUser} online={onlineMap ? onlineMap[peer.uid] ?? false : undefined} badge={badge} large={large} small={small} />;
    }
  }

  return <Avatar user={conversationAvatarUser(conversation, detail)} badge={badge} large={large} small={small} />;
}
