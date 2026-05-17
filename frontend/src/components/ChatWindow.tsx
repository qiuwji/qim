import React from 'react';
import type { ConversationDTO, FriendDTO, MessageDTO, UserConvDTO, UserDTO } from '@/api/types';
import { useChatScroll } from '@/hooks/useChatScroll';
import { MessageComposer } from './MessageComposer';
import { MessageList } from './MessageList';

export function ChatWindow({ user, conversation, detail: _detail, title, subtitle, messages, hasMore, typingText: _typingText, userCache, friendMap, detailOpen, replyTo, onBack, onSend, onSendImage, onTyping, onToggleDetail, onContextMenu, onReply, onLoadMore, onAvatarEnter, onAvatarLeave, onAvatarClick }: {
  user: UserDTO; conversation: UserConvDTO | null; detail?: ConversationDTO; title: string; subtitle: string; messages: MessageDTO[]; hasMore: boolean; typingText?: string; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; detailOpen: boolean; replyTo: MessageDTO | null;
  onBack: () => void; onSend: (text: string) => Promise<void>; onSendImage: (file: File) => void; onTyping: () => void; onToggleDetail: () => void; onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void; onReply: (m: MessageDTO | null) => void; onLoadMore: () => void | Promise<void>;
  onAvatarEnter?: (uid: number, e: React.MouseEvent) => void;
  onAvatarLeave?: (e: React.MouseEvent) => void;
  onAvatarClick?: (uid: number) => void;
}) {
  const { areaRef, bottomRef, captureStickToBottom, handleAreaScroll } = useChatScroll({
    conversationID: conversation?.conversation_id,
    messagesLength: messages.length,
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
          <button className={`detail-toggle ${detailOpen ? 'active' : ''}`} onClick={(e) => { e.stopPropagation(); onToggleDetail(); }} title="设置">···</button>
        </div>
      </header>}
      <div className="message-area" ref={areaRef} onScroll={handleAreaScroll}>
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
      <MessageComposer
        disabled={!conversation}
        replyTo={replyMsg}
        onSend={onSend}
        onSendImage={onSendImage}
        onTyping={onTyping}
        onReply={onReply}
        onBeforeSend={captureStickToBottom}
      />
    </div>
  );
}
