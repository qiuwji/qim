import React, { useEffect, useRef, useState } from 'react';
import type { ConversationDTO, FriendDTO, MemberDTO, MessageDTO, UserConvDTO, UserDTO } from '../api/types';
import { displayName, timeText } from '../utils';
import { EmptyState } from './EmptyState';
import { MessageBubble } from './MessageBubble';

const TYPING_THROTTLE = 5000;

export function ChatWindow({ user, conversation, detail, title, subtitle, messages, hasMore, typingText, memberCount, members, userCache, friendMap, detailOpen, replyTo, showChatSearch, chatSearch, chatSearchResult, onBack, onSend, onSendImage, onTyping, onToggleDetail, onContextMenu, onReply, onLoadMore, onChatSearch, onChatSearchChange, onToggleChatSearch, onAvatarEnter, onAvatarLeave, onAvatarClick }: {
  user: UserDTO; conversation: UserConvDTO | null; detail?: ConversationDTO; title: string; subtitle: string; messages: MessageDTO[]; hasMore: boolean; typingText?: string; memberCount?: number; members: MemberDTO[]; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; detailOpen: boolean; replyTo: MessageDTO | null; showChatSearch: boolean; chatSearch: string; chatSearchResult: MessageDTO[];
  onBack: () => void; onSend: (text: string) => Promise<void>; onSendImage: (file: File) => void; onTyping: () => void; onToggleDetail: () => void; onContextMenu: (e: React.MouseEvent, m: MessageDTO) => void; onReply: (m: MessageDTO | null) => void; onLoadMore: () => void; onChatSearch: () => void; onChatSearchChange: (v: string) => void; onToggleChatSearch: () => void;
  onAvatarEnter?: (uid: number, e: React.MouseEvent) => void;
  onAvatarLeave?: () => void;
  onAvatarClick?: (uid: number) => void;
}) {
  const [draft, setDraft] = useState('');
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const topRef = useRef<HTMLDivElement | null>(null);
  const imageRef = useRef<HTMLInputElement | null>(null);
  const lastTypingRef = useRef(0);

  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages.length, conversation?.conversation_id]);
  useEffect(() => {
    if (!topRef.current) return;
    const obs = new IntersectionObserver(([entry]) => { if (entry.isIntersecting && hasMore) onLoadMore(); }, { threshold: 0.1 });
    obs.observe(topRef.current); return () => obs.disconnect();
  }, [hasMore, conversation?.conversation_id]);

  async function submit(e: { preventDefault: () => void }) { e.preventDefault(); const text = draft; setDraft(''); await onSend(text); }

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
          <button className="icon-btn-sm" onClick={onToggleChatSearch} title="搜索">🔍</button>
          <button className={`detail-toggle ${detailOpen ? 'active' : ''}`} onClick={onToggleDetail} title="设置">···</button>
        </div>}
      </header>
      {showChatSearch && <div className="chat-search-bar"><input value={chatSearch} onChange={(e) => onChatSearchChange(e.target.value)} placeholder="搜索聊天记录" onKeyDown={(e) => e.key === 'Enter' && onChatSearch()} /><button onClick={onChatSearch}>搜索</button></div>}
      {chatSearchResult.length > 0 && <div className="search-results">{chatSearchResult.map((m) => (<div key={m.id} className="search-result-item" onClick={() => onReply(m)}><strong>{displayName(m.sender_id, userCache, friendMap)}</strong><span>{m.content.slice(0, 60)}</span></div>))}</div>}
      <div className="message-area">
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
              <MessageBubble message={message} mine={message.sender_id === user.id} senderName={displayName(message.sender_id, userCache, friendMap)} replySource={replySource} onContextMenu={onContextMenu}
                onAvatarEnter={onAvatarEnter ? (e) => onAvatarEnter(message.sender_id, e) : undefined}
                onAvatarLeave={onAvatarLeave}
                onAvatarClick={onAvatarClick ? () => onAvatarClick(message.sender_id) : undefined}
              />
            </React.Fragment>
          );
        })}
        <div ref={bottomRef} />
      </div>
      {replyMsg && <div className="reply-bar"><span>回复: {replyMsg.content.slice(0, 40)}{replyMsg.content.length > 40 ? '...' : ''}</span><button onClick={() => onReply(null)}>✕</button></div>}
      <form className="composer" onSubmit={submit}>
        <div className="composer-toolbar">
          <span onClick={() => imageRef.current?.click()} style={{ cursor: 'pointer' }}>图片</span>
          <span>表情</span>
          <span>文件</span>
          <input ref={imageRef} type="file" accept="image/*" hidden onChange={(e) => e.target.files?.[0] && onSendImage(e.target.files[0])} />
        </div>
        <textarea value={draft} disabled={!conversation} onChange={handleDraftChange} onKeyDown={handleKeyDown} placeholder={conversation ? '输入消息，Enter 发送，Shift+Enter 换行' : '请选择聊天'} rows={1} />
        <button className="primary-btn" disabled={!conversation || !draft.trim()}>发送</button>
      </form>
    </div>
  );
}
