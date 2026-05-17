import type { ConversationDTO, FriendDTO, MemberDTO, MessageDTO, UserConvDTO } from '@/api/types';
import { ChatDetailHeader } from '@/components/ChatDetailHeader';
import { ChatProfileCard } from '@/components/ChatProfileCard';
import { ChatSearchSection } from '@/components/ChatSearchSection';
import { MemberSection } from '@/components/MemberSection';
import { GroupAdminSection } from '@/components/GroupAdminSection';
import { canManageGroup } from '@/hooks/chat/models/conversationViewModel';

export function ChatDetailPanel({ chat, detail, members, currentUID, userCache, friendMap, chatSearch, chatSearchResult, onChatSearch, onChatSearchChange, onJumpToMessage, onRenameGroup, onInviteMember, onRemoveMember, onLeaveGroup, onDissolveGroup, onTogglePin, onToggleMute, onHide, onSetMemberRole, onTransferOwner, onUploadGroupAvatar, onSetMemberLimit, onViewUser, onClose }: {
  chat: UserConvDTO | null; detail?: ConversationDTO; members: MemberDTO[]; currentUID: number; userCache: Record<number, import('@/api/types').UserDTO>; friendMap?: Record<number, FriendDTO>;
  chatSearch: string; chatSearchResult: MessageDTO[]; onChatSearch: () => void; onChatSearchChange: (v: string) => void; onJumpToMessage: (id: number) => void;
  onRenameGroup: () => void; onInviteMember: () => void; onRemoveMember: (uid: number) => void; onLeaveGroup: () => void; onDissolveGroup: () => void; onTogglePin: (convID: number) => void; onToggleMute: (convID: number) => void; onHide: (convID: number) => void;
  onSetMemberRole: (uid: number, role: number) => void; onTransferOwner: (uid: number) => void; onUploadGroupAvatar: (file: File) => void; onSetMemberLimit: () => void; onViewUser?: (uid: number) => void; onClose: () => void;
}) {
  if (!chat) {
    return (
      <div className="detail-inner">
        <ChatDetailHeader title="聊天设置" onClose={onClose} />
        <div className="detail-card"><span>选择一个聊天后查看设置。</span></div>
      </div>
    );
  }

  const { isGroup, isOwner, isAdmin } = canManageGroup(detail, members, currentUID);

  return (
    <div className="detail-inner">
      <ChatDetailHeader title="聊天设置" onClose={onClose} />
      <ChatProfileCard chat={chat} detail={detail} members={members} isAdmin={isAdmin} onTogglePin={onTogglePin} onToggleMute={onToggleMute} onHide={onHide} onUploadGroupAvatar={onUploadGroupAvatar} />
      <ChatSearchSection value={chatSearch} onChange={onChatSearchChange} onSearch={onChatSearch} results={chatSearchResult} userCache={userCache} friendMap={friendMap} onJumpToMessage={onJumpToMessage} />
      <MemberSection members={members} currentUID={currentUID} userCache={userCache} friendMap={friendMap} isGroup={isGroup} isAdmin={isAdmin} onInviteMember={onInviteMember} onViewUser={onViewUser} />
      {isGroup && (isAdmin || isOwner) && (
        <GroupAdminSection members={members} currentUID={currentUID} userCache={userCache} friendMap={friendMap} isOwner={isOwner} isAdmin={isAdmin} onRenameGroup={onRenameGroup} onSetMemberLimit={onSetMemberLimit} onSetMemberRole={onSetMemberRole} onTransferOwner={onTransferOwner} onRemoveMember={onRemoveMember} onLeaveGroup={onLeaveGroup} onDissolveGroup={onDissolveGroup} onViewUser={onViewUser} />
      )}
    </div>
  );
}
