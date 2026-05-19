import React from 'react';
import type { ConversationDTO, FriendDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO } from '@/api/types';
import { isGroupConversation, canManageGroup } from '@/hooks/chat/models/conversationViewModel';
import { useChatScroll } from '@/hooks/useChatScroll';
import { MessageComposer } from './MessageComposer';
import { MessageList } from './MessageList';
import { CallButton } from './CallButton';

export function ChatWindow({ user, conversation, detail, title, subtitle, messages, hasMore, typingText: _typingText, members, userCache, friendMap, detailOpen, replyTo, onBack, onSend, onSendImage, onTyping, onToggleDetail, onContextMenu, onReply, onLoadMore, onAvatarEnter, onAvatarLeave, onAvatarClick, callState, onVoiceCall, onVideoCall }: {
  user: UserDTO; conversation: UserConvDTO | null; detail?: ConversationDTO; title: string; subtitle: string; messages: MessageDTO[]; hasMore: boolean; typingText?: string; members: MemberDTO[]; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; detailOpen: boolean; replyTo: MessageDTO | null;
  onBack: () => void; onSend: (text: string, mentionUIDs?: number[], mentionAll?: boolean) => Promise<void>; onSendImage: (file: File) => void; onTyping: () => void; onToggleDetail: () => void; onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void; onReply: (m: MessageDTO | null) => void; onLoadMore: () => void | Promise<void>;
  onAvatarEnter?: (uid: number, e: React.MouseEvent) => void;
  onAvatarLeave?: (e: React.MouseEvent) => void;
  onAvatarClick?: (uid: number) => void;
  callState?: string; onVoiceCall?: () => void; onVideoCall?: () => void;
}) {
  const isGroup = isGroupConversation(detail);
  const { isAdmin } = canManageGroup(detail, members, user.id);
  const latestMessage = messages[messages.length - 1];
  const latestMessageKey = latestMessage
    ? `${latestMessage.id}-${latestMessage.client_id}-${latestMessage.seq}-${latestMessage.created_at}`
    : '';
  const { areaRef, contentRef, bottomRef, captureStickToBottom, handleAreaScroll } = useChatScroll({
    conversationID: conversation?.conversation_id,
    messagesLength: messages.length,
    latestMessageKey,
    hasMore,
    onLoadMore,
  });

  const replyMsg = replyTo ? messages.find((m) => m.id === replyTo.id) ?? replyTo : null;

  return (
    <div className="chat-window">
      {conversation && <header className="chat-header">
        <button className="back-btn" onClick={onBack}>
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="15 18 9 12 15 6" /></svg>
        </button>
        <div className="chat-title">
          <strong>{title}</strong>
          <span>{subtitle}</span>
        </div>
        <div className="chat-actions">
          <CallButton peerUID={detail?.type === 1 ? (members.find(m => m.uid !== user.id)?.uid ?? 0) : 0} isGroup={isGroup} callState={callState ?? 'idle'} onVoiceCall={() => onVoiceCall?.()} onVideoCall={() => onVideoCall?.()} />
          <button className={`detail-toggle ${detailOpen ? 'active' : ''}`} onClick={(e) => { e.stopPropagation(); onToggleDetail(); }} title="设置">···</button>
        </div>
      </header>}
      <div className="message-area" ref={areaRef} onScroll={handleAreaScroll}>
        <div ref={contentRef}>
          <MessageList
            conversationActive={Boolean(conversation)}
            messages={messages}
            hasMore={hasMore}
            currentUserID={user.id}
            userCache={userCache}
            friendMap={friendMap}
            onContextMenu={onContextMenu}
            onAvatarEnter={onAvatarEnter}
            onAvatarLeave={onAvatarLeave}
            onAvatarClick={onAvatarClick}
          />
          <div ref={bottomRef} />
        </div>
      </div>
      <MessageComposer
        disabled={!conversation}
        replyTo={replyMsg}
        isGroup={isGroup}
        isAdmin={isAdmin}
        members={members}
        currentUID={user.id}
        userCache={userCache}
        friendMap={friendMap}
        onSend={onSend}
        onSendImage={onSendImage}
        onTyping={onTyping}
        onReply={onReply}
        onBeforeSend={captureStickToBottom}
      />
    </div>
  );
}
