import React, { useRef, useState } from 'react';
import type { MemberDTO, MessageDTO, UserDTO } from '@/api/types';
import { EmojiPicker } from '@/components/ui';
import { MentionPopover } from './MentionPopover';
import { messageDisplayText } from '@/hooks/chat/models/messageModel';

const TYPING_THROTTLE = 5000;

interface MentionEntry {
  uid: number;
  start: number;
  length: number;
}

export function MessageComposer({
  disabled,
  replyTo,
  isGroup,
  isAdmin,
  members,
  currentUID,
  userCache,
  friendMap,
  onSend,
  onSendImage,
  onTyping,
  onReply,
  onBeforeSend,
}: {
  disabled: boolean;
  replyTo: MessageDTO | null;
  isGroup?: boolean;
  isAdmin?: boolean;
  members?: MemberDTO[];
  currentUID?: number;
  userCache?: Record<number, UserDTO>;
  friendMap?: Record<number, import('@/api/types').FriendDTO>;
  onSend: (text: string, mentionUIDs?: number[], mentionAll?: boolean) => Promise<void>;
  onSendImage: (file: File) => void;
  onTyping: () => void;
  onReply: (m: MessageDTO | null) => void;
  onBeforeSend: () => void;
}) {
  const [draft, setDraft] = useState('');
  const [showEmoji, setShowEmoji] = useState(false);
  const [mentionMode, setMentionMode] = useState(false);
  const [mentionKeyword, setMentionKeyword] = useState('');
  const [mentions, setMentions] = useState<MentionEntry[]>([]);
  const imageRef = useRef<HTMLInputElement | null>(null);
  const lastTypingRef = useRef(0);
  const replyPreview = replyTo ? messageDisplayText(replyTo) : '';

  function syncMentions(text: string, prev: MentionEntry[]): MentionEntry[] {
    return prev.filter((m) => {
      const fragment = text.substring(m.start, m.start + m.length);
      return fragment.startsWith('@');
    });
  }

  async function submit(e: { preventDefault: () => void }) {
    e.preventDefault();
    onBeforeSend();
    const text = draft;
    const mentionUIDs = mentions.map((m) => m.uid).filter((uid) => uid !== 0);
    const mentionAll = mentions.some((m) => m.uid === 0);
    setDraft('');
    setMentions([]);
    setMentionMode(false);
    setMentionKeyword('');
    await onSend(text, mentionUIDs.length > 0 || mentionAll ? mentionUIDs : undefined, mentionAll || undefined);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (mentionMode && e.key !== 'Escape') return;
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      submit(e);
    }
  }

  function handleDraftChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    const text = e.target.value;
    setDraft(text);
    setMentions((prev) => syncMentions(text, prev));

    const now = Date.now();
    if (now - lastTypingRef.current > TYPING_THROTTLE) {
      lastTypingRef.current = now;
      onTyping();
    }

    const cursorPos = e.target.selectionStart;
    const lastAt = text.lastIndexOf('@', cursorPos - 1);
    if (lastAt >= 0 && isGroup) {
      const afterAt = text.substring(lastAt + 1, cursorPos);
      if (!afterAt.includes(' ') && !afterAt.includes('\n')) {
        setMentionMode(true);
        setMentionKeyword(afterAt);
        return;
      }
    }
    setMentionMode(false);
    setMentionKeyword('');
  }

  function handleMentionSelect(uid: number, name: string) {
    const textarea = document.querySelector<HTMLTextAreaElement>('.composer textarea');
    if (!textarea) return;
    const cursorPos = textarea.selectionStart;
    const lastAt = draft.lastIndexOf('@', cursorPos - 1);
    if (lastAt < 0) return;

    const before = draft.substring(0, lastAt);
    const after = draft.substring(cursorPos);
    const mentionText = `@${name} `;
    const newDraft = before + mentionText + after;
    setDraft(newDraft);
    setMentionMode(false);
    setMentionKeyword('');

    const entry: MentionEntry = { uid, start: lastAt, length: mentionText.length };
    setMentions((prev) => syncMentions(newDraft, [...prev, entry]));

    requestAnimationFrame(() => {
      const pos = lastAt + mentionText.length;
      textarea.selectionStart = pos;
      textarea.selectionEnd = pos;
      textarea.focus();
    });
  }

  function handleMentionSelectAll() {
    const textarea = document.querySelector<HTMLTextAreaElement>('.composer textarea');
    if (!textarea) return;
    const cursorPos = textarea.selectionStart;
    const lastAt = draft.lastIndexOf('@', cursorPos - 1);
    if (lastAt < 0) return;

    const before = draft.substring(0, lastAt);
    const after = draft.substring(cursorPos);
    const mentionText = '@全体成员 ';
    const newDraft = before + mentionText + after;
    setDraft(newDraft);
    setMentionMode(false);
    setMentionKeyword('');

    const entry: MentionEntry = { uid: 0, start: lastAt, length: mentionText.length };
    setMentions((prev) => syncMentions(newDraft, [...prev, entry]));

    requestAnimationFrame(() => {
      const pos = lastAt + mentionText.length;
      textarea.selectionStart = pos;
      textarea.selectionEnd = pos;
      textarea.focus();
    });
  }

  return (
    <>
      {replyTo && (
        <div className="reply-bar">
          <span>回复: {replyPreview.slice(0, 40)}{replyPreview.length > 40 ? '...' : ''}</span>
          <button onClick={() => onReply(null)}>✕</button>
        </div>
      )}
      {!disabled && (
        <form className="composer" onSubmit={submit}>
          {mentionMode && isGroup && members && currentUID && userCache && (
            <MentionPopover
              members={members}
              currentUID={currentUID}
              isGroup={!!isGroup}
              isAdmin={!!isAdmin}
              userCache={userCache}
              friendMap={friendMap}
              keyword={mentionKeyword}
              onSelect={handleMentionSelect}
              onSelectAll={handleMentionSelectAll}
              onClose={() => { setMentionMode(false); setMentionKeyword(''); }}
            />
          )}
          <textarea value={draft} onChange={handleDraftChange} onKeyDown={handleKeyDown} placeholder="输入消息，Enter 发送，Shift+Enter 换行" rows={1} />
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
            <button className="primary-btn" disabled={!draft.trim()}>发送</button>
          </div>
          {showEmoji && <EmojiPicker onSelect={(emoji: string) => { setDraft((d) => d + emoji); setShowEmoji(false); }} onClose={() => setShowEmoji(false)} />}
        </form>
      )}
    </>
  );
}
