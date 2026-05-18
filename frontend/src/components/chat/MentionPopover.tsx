import React, { useEffect, useRef, useState } from 'react';
import type { MemberDTO, UserDTO } from '@/api/types';
import { displayName } from '@/utils';
import { Avatar } from '@/components/ui';

interface MentionPopoverProps {
  members: MemberDTO[];
  currentUID: number;
  isGroup: boolean;
  isAdmin: boolean;
  userCache: Record<number, UserDTO>;
  friendMap?: Record<number, import('@/api/types').FriendDTO>;
  keyword: string;
  onSelect: (uid: number, name: string) => void;
  onSelectAll: () => void;
  onClose: () => void;
}

export function MentionPopover({ members, currentUID, isAdmin, userCache, friendMap, keyword, onSelect, onSelectAll, onClose }: MentionPopoverProps) {
  const [highlightIndex, setHighlightIndex] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = members
    .filter((m) => m.uid !== currentUID)
    .map((m) => ({ ...m, name: displayName(m.uid, userCache, friendMap) }))
    .filter((m) => !keyword || m.name.toLowerCase().includes(keyword.toLowerCase()));

  useEffect(() => {
    setHighlightIndex(0);
  }, [keyword]);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setHighlightIndex((i) => Math.min(i + 1, filtered.length + (isAdmin ? 1 : 0) - 1));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setHighlightIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        if (highlightIndex < filtered.length) {
          onSelect(filtered[highlightIndex].uid, filtered[highlightIndex].name);
        } else if (isAdmin) {
          onSelectAll();
        }
      } else if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    }
    document.addEventListener('keydown', handleKey, true);
    return () => document.removeEventListener('keydown', handleKey, true);
  }, [filtered, highlightIndex, isAdmin, onClose, onSelect, onSelectAll]);

  useEffect(() => {
    const el = listRef.current?.children[highlightIndex] as HTMLElement | undefined;
    el?.scrollIntoView({ block: 'nearest' });
  }, [highlightIndex]);

  if (!filtered.length && !isAdmin) return null;

  return (
    <div className="mention-popover">
      <div ref={listRef} className="mention-list">
        {filtered.map((m, i) => (
          <button key={m.uid} className={`mention-item ${i === highlightIndex ? 'highlighted' : ''}`} onClick={() => onSelect(m.uid, m.name)}>
            {userCache[m.uid]
              ? <Avatar user={userCache[m.uid]} small />
              : <span className="mini-avatar">{m.name.slice(0, 1)}</span>
            }
            <span className="mention-item-name">{m.name}</span>
          </button>
        ))}
        {isAdmin && (
          <button className={`mention-item mention-all ${highlightIndex === filtered.length ? 'highlighted' : ''}`} onClick={onSelectAll}>
            <span className="mini-avatar" style={{ background: 'linear-gradient(135deg, #f59e0b, #d97706)' }}>全</span>
            <span className="mention-item-name">全体成员</span>
          </button>
        )}
      </div>
    </div>
  );
}
