import React from 'react';
import type { FriendDTO, MessageDTO, UserDTO } from '@/api/types';
import { EmptyState } from '@/components/ui';
import { displayName, scrollToMessage, timeText } from '@/utils';
import { MessageBubble } from './MessageBubble';

export function MessageList({
  conversationActive,
  messages,
  hasMore,
  currentUserID,
  userCache,
  friendMap,
  onContextMenu,
  onAvatarEnter,
  onAvatarLeave,
  onAvatarClick,
}: {
  conversationActive: boolean;
  messages: MessageDTO[];
  hasMore: boolean;
  currentUserID: number;
  userCache: Record<number, UserDTO>;
  friendMap?: Record<number, FriendDTO>;
  onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void;
  onAvatarEnter?: (uid: number, e: React.MouseEvent) => void;
  onAvatarLeave?: (e: React.MouseEvent) => void;
  onAvatarClick?: (uid: number) => void;
}) {
  function handleReplyClick(messageID: number) {
    scrollToMessage(messageID);
  }
  if (!conversationActive) {
    return (
      <div className="flex h-full items-center justify-center">
        <svg viewBox="0 0 24 24" width="64" height="64" fill="none" stroke="#c9ced6" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </div>
    );
  }

  return (
    <>
      {!messages.length && <EmptyState title="还没有消息" text="发送第一条消息，开始这段对话。" />}
      <div className="load-more-sentinel">{hasMore ? '↑ 加载更多' : ''}</div>
      {messages.map((message, idx) => {
        const prevMsg = idx > 0 ? messages[idx - 1] : null;
        const showTimeDivider = !prevMsg || message.created_at - prevMsg.created_at > 300;
        const replySource = message.reply_to > 0 ? messages.find((m) => m.id === message.reply_to) : undefined;
        return (
          <React.Fragment key={`${message.id}-${message.client_id}`}>
            {showTimeDivider && <div className="time-divider">{timeText(message.created_at)}</div>}
            <div id={`msg-${message.id}`}>
              <MessageBubble
                message={message}
                mine={message.sender_id === currentUserID}
                senderName={displayName(message.sender_id, userCache, friendMap)}
                senderUser={userCache[message.sender_id]}
                replySource={replySource}
                currentUID={currentUserID}
                userCache={userCache}
                friendMap={friendMap}
                onContextMenu={onContextMenu}
                onAvatarEnter={onAvatarEnter ? (e) => onAvatarEnter(message.sender_id, e) : undefined}
                onAvatarLeave={onAvatarLeave}
                onAvatarClick={onAvatarClick ? () => onAvatarClick(message.sender_id) : undefined}
                onReplyClick={handleReplyClick}
              />
            </div>
          </React.Fragment>
        );
      })}
    </>
  );
}
