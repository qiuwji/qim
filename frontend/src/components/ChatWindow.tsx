import React, { useLayoutEffect, useRef, useState, useCallback } from 'react';
import type { ConversationDTO, FriendDTO, MessageDTO, UserConvDTO, UserDTO } from '@/api/types';
import { displayName, timeText } from '@/utils';
import { EmptyState, EmojiPicker } from '@/components/ui';
import { MessageBubble } from '@/components/MessageBubble';

const TYPING_THROTTLE = 5000;

export function ChatWindow({ user, conversation, detail, title, subtitle, messages, hasMore, typingText, userCache, friendMap, detailOpen, replyTo, onBack, onSend, onSendImage, onTyping, onToggleDetail, onContextMenu, onReply, onLoadMore, onAvatarEnter, onAvatarLeave, onAvatarClick }: {
  user: UserDTO; conversation: UserConvDTO | null; detail?: ConversationDTO; title: string; subtitle: string; messages: MessageDTO[]; hasMore: boolean; typingText?: string; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; detailOpen: boolean; replyTo: MessageDTO | null;
  onBack: () => void; onSend: (text: string) => Promise<void>; onSendImage: (file: File) => void; onTyping: () => void; onToggleDetail: () => void; onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void; onReply: (m: MessageDTO | null) => void; onLoadMore: () => void | Promise<void>;
  onAvatarEnter?: (uid: number, e: React.MouseEvent) => void;
  onAvatarLeave?: (e: React.MouseEvent) => void;
  onAvatarClick?: (uid: number) => void;
}) {
  const [draft, setDraft] = useState('');
  const [showEmoji, setShowEmoji] = useState(false);
  const areaRef = useRef<HTMLDivElement | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const topRef = useRef<HTMLDivElement | null>(null);
  const imageRef = useRef<HTMLInputElement | null>(null);
  const lastTypingRef = useRef(0);
  const prevLenRef = useRef(0);
  const loadingMoreRef = useRef(false);
  const pendingInitialScrollRef = useRef(false);
  const stickToBottomRef = useRef(true);

  const isNearBottom = useCallback(() => {
    const el = areaRef.current;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight < 120;
  }, []);

  const scrollToBottom = useCallback((smooth: boolean) => {
    const el = areaRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
    stickToBottomRef.current = true;
    if (smooth) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, []);

  const triggerLoadMore = useCallback(() => {
    const el = areaRef.current;
    if (!el || !hasMore || loadingMoreRef.current) return;
    const prevHeight = el.scrollHeight;
    const prevTop = el.scrollTop;
    loadingMoreRef.current = true;
    Promise.resolve(onLoadMore()).finally(() => {
      requestAnimationFrame(() => {
        const latest = areaRef.current;
        if (latest) {
          latest.scrollTop = latest.scrollHeight - prevHeight + prevTop;
        }
        loadingMoreRef.current = false;
      });
    });
  }, [hasMore, onLoadMore]);

  const handleAreaScroll = useCallback(() => {
    const el = areaRef.current;
    stickToBottomRef.current = isNearBottom();
    if (el && el.scrollTop <= 80) triggerLoadMore();
  }, [isNearBottom, triggerLoadMore]);

  useLayoutEffect(() => {
    pendingInitialScrollRef.current = true;
    loadingMoreRef.current = false;
    scrollToBottom(false);
  }, [conversation?.conversation_id, scrollToBottom]);

  useLayoutEffect(() => {
    if (pendingInitialScrollRef.current) {
      scrollToBottom(false);
      requestAnimationFrame(() => scrollToBottom(false));
      if (messages.length > 0 || !conversation) {
        pendingInitialScrollRef.current = false;
      }
      prevLenRef.current = messages.length;
      return;
    }
    if (loadingMoreRef.current) {
      loadingMoreRef.current = false;
      prevLenRef.current = messages.length;
      return;
    }
    if (messages.length > prevLenRef.current && stickToBottomRef.current) {
      scrollToBottom(true);
    }
    prevLenRef.current = messages.length;
  }, [conversation, messages.length, scrollToBottom]);

  async function submit(e: { preventDefault: () => void }) {
    e.preventDefault();
    stickToBottomRef.current = isNearBottom();
    const text = draft;
    setDraft('');
    await onSend(text);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); submit(e); }
  }

  function handleDraftChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setDraft(e.target.value);
    const now = Date.now();
    if (now - lastTypingRef.current >= TYPING_THROTTLE) {
      lastTypingRef.current = now;
      onTyping();
    }
  }

  const replyMsg = replyTo ? messages.find((m) => m.id === replyTo.id) ?? replyTo : null;

  return (
    <div className="chat-window">
      <header className="chat-header">
        <button className="back-btn" onClick={onBack}>返回</button>
        <div className="chat-title">
          <strong>{title}</strong>
          <span>{subtitle}</span>
        </div>
        {conversation && <div className="chat-actions">
          <button className={`detail-toggle ${detailOpen ? 'active' : ''}`} onClick={onToggleDetail} title="设置">···</button>
        </div>}
      </header>
      <div className="message-area" ref={areaRef} onScroll={handleAreaScroll}>
        {!conversation && <EmptyState title="欢迎使用 QIM" text="左侧选择聊天，或从通讯录里发起新的聊天。" />}
        {conversation && !messages.length && <EmptyState title="还没有消息" text="发送第一条消息，开始这段对话。" />}
        {conversation && <div ref={topRef} className="load-more-sentinel">{hasMore ? '↑ 加载更多' : ''}</div>}
        {messages.map((message, idx) => {
          const prevMsg = idx > 0 ? messages[idx - 1] : null;
          const showTimeDivider = !prevMsg || (message.created_at - prevMsg.created_at > 300);
          const replySource = message.reply_to > 0 ? messages.find((m) => m.id === message.reply_to) : undefined;
          return (
            <React.Fragment key={`${message.id}-${message.client_id}`}>
              {showTimeDivider && <div className="time-divider">{timeText(message.created_at)}</div>}
              <div id={`msg-${message.id}`}>
              <MessageBubble message={message} mine={message.sender_id === user.id} senderName={displayName(message.sender_id, userCache, friendMap)} senderUser={userCache[message.sender_id]} replySource={replySource} onContextMenu={onContextMenu}
                onAvatarEnter={onAvatarEnter ? (e) => onAvatarEnter(message.sender_id, e) : undefined}
                onAvatarLeave={onAvatarLeave}
                onAvatarClick={onAvatarClick ? () => onAvatarClick(message.sender_id) : undefined}
              />
              </div>
            </React.Fragment>
          );
        })}
        <div ref={bottomRef} />
      </div>
      {replyMsg && <div className="reply-bar"><span>回复: {replyMsg.content.slice(0, 40)}{replyMsg.content.length > 40 ? '...' : ''}</span><button onClick={() => onReply(null)}>✕</button></div>}
      <form className="composer" onSubmit={submit}>
        <textarea value={draft} disabled={!conversation} onChange={handleDraftChange} onKeyDown={handleKeyDown} placeholder={conversation ? '输入消息，Enter 发送，Shift+Enter 换行' : '请选择聊天'} rows={1} />
        <div className="composer-bottom">
          <div className="composer-toolbar">
            <button type="button" onClick={() => imageRef.current?.click()} title="图片">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" /><circle cx="8.5" cy="8.5" r="1.5" /><polyline points="21 15 16 10 5 21" /></svg>
            </button>
            <button type="button" title="表情" onClick={() => setShowEmoji((v) => !v)}>
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><path d="M8 14s1.5 2 4 2 4-2 4-2" /><line x1="9" y1="9" x2="9.01" y2="9" /><line x1="15" y1="9" x2="15.01" y2="9" /></svg>
            </button>
            <button type="button" title="文件">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" /></svg>
            </button>
            <input ref={imageRef} type="file" accept="image/*" hidden onChange={(e) => e.target.files?.[0] && onSendImage(e.target.files[0])} />
          </div>
          <button className="primary-btn" disabled={!conversation || !draft.trim()}>发送</button>
        </div>
        {showEmoji && <EmojiPicker onSelect={(emoji: string) => { setDraft((d) => d + emoji); setShowEmoji(false); }} onClose={() => setShowEmoji(false)} />}
      </form>
    </div>
  );
}
