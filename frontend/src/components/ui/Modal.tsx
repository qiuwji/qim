import { useState } from 'react';
import type { ConversationDTO, FriendDTO, MemberDTO, UserConvDTO, UserDTO } from '@/api/types';
import type { ContextMenu as ContextMenuType, ModalState } from '@/types';
import { ContextMenu, Avatar } from '@/components/ui';
import { ConversationSummaryRow } from '@/components/conversation/ConversationSummaryRow';

export function AppModal({ modal, onClose }: { modal: ModalState; onClose: () => void }) {
  if (!modal) return null;
  if (modal.type === 'prompt') return <PromptModal fields={modal.fields} title={modal.title} onConfirm={modal.onConfirm} onClose={onClose} />;
  if (modal.type === 'confirm') return <ConfirmModal title={modal.title} text={modal.text} danger={modal.danger} onConfirm={modal.onConfirm} onClose={onClose} />;
  if (modal.type === 'friend-picker') return <FriendPickerModal title={modal.title} friends={modal.friends} userCache={modal.userCache} excludeUIDs={modal.excludeUIDs} requireGroupName={modal.requireGroupName} onConfirm={modal.onConfirm} onClose={onClose} />;
  if (modal.type === 'conversation-picker') return <ConversationPickerModal title={modal.title} conversations={modal.conversations} details={modal.details} members={modal.members} userCache={modal.userCache} onlineMap={modal.onlineMap} currentUID={modal.currentUID} lastMsgMap={modal.lastMsgMap} mentionMap={modal.mentionMap} onConfirm={modal.onConfirm} onClose={onClose} />;
  return null;
}

function PromptModal({ title, fields, onConfirm, onClose }: {
  title: string; fields: { key: string; label: string; placeholder?: string; defaultValue?: string }[]; onConfirm: (v: Record<string, string>) => void; onClose: () => void;
}) {
  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    for (const f of fields) init[f.key] = f.defaultValue ?? '';
    return init;
  });
  return (
    <div className="modal-overlay" onClick={onClose}><div className="modal-card" onClick={(e) => e.stopPropagation()}>
      <div className="modal-header"><strong>{title}</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="modal-body">{fields.map((f) => (<label key={f.key} className="modal-field"><span>{f.label}</span><input value={values[f.key] ?? ''} placeholder={f.placeholder} onChange={(e) => setValues((p) => ({ ...p, [f.key]: e.target.value }))} /></label>))}</div>
      <div className="modal-actions"><button className="modal-cancel" onClick={onClose}>取消</button><button className="primary-btn" onClick={() => { onConfirm(values); onClose(); }}>确认</button></div>
    </div></div>
  );
}

function FriendPickerModal({ title, friends, userCache, excludeUIDs, requireGroupName, onConfirm, onClose }: {
  title: string;
  friends: FriendDTO[];
  userCache: Record<number, UserDTO>;
  excludeUIDs?: number[];
  requireGroupName?: boolean;
  onConfirm: (values: { name?: string; usernames: string[] }) => void;
  onClose: () => void;
}) {
  const excluded = new Set(excludeUIDs ?? []);
  const candidates = friends
    .map((friend) => ({ friend, user: userCache[friend.friend_uid] }))
    .filter((item) => !excluded.has(item.friend.friend_uid) && !!item.user?.username);
  const [name, setName] = useState('');
  const [keyword, setKeyword] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const filtered = candidates.filter(({ friend, user }) => {
    const display = `${friend.remark || ''} ${user?.nickname || ''} ${user?.username || ''}`.toLowerCase();
    return display.includes(keyword.trim().toLowerCase());
  });

  function toggle(username: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(username)) next.delete(username);
      else next.add(username);
      return next;
    });
  }

  const canConfirm = (!requireGroupName || !!name.trim()) && selected.size > 0;

  return (
    <div className="modal-overlay" onClick={onClose}><div className="modal-card friend-picker-modal" onClick={(e) => e.stopPropagation()}>
      <div className="modal-header"><strong>{title}</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="modal-body">
        {requireGroupName && <label className="modal-field"><span>群聊名称</span><input value={name} placeholder="输入群聊名称" onChange={(e) => setName(e.target.value)} /></label>}
        <label className="modal-field"><span>选择好友</span><input value={keyword} placeholder="搜索好友昵称或账号" onChange={(e) => setKeyword(e.target.value)} /></label>
        <div className="friend-picker-list">
          {filtered.map(({ friend, user }) => {
            if (!user) return null;
            const dn = friend.remark || user.nickname || user.username;
            const checked = selected.has(user.username);
            return (
              <button key={friend.friend_uid} className={`friend-picker-item ${checked ? 'selected' : ''}`} onClick={() => toggle(user.username)}>
                <Avatar user={user} small />
                <span className="friend-picker-text"><strong>{dn}</strong><small>@{user.username}</small></span>
                <span className="friend-picker-check">{checked ? '✓' : ''}</span>
              </button>
            );
          })}
          {!filtered.length && <div className="empty-hint">没有可选择的好友</div>}
        </div>
      </div>
      <div className="modal-actions"><button className="modal-cancel" onClick={onClose}>取消</button><button className="primary-btn" disabled={!canConfirm} onClick={() => { if (!canConfirm) return; onConfirm({ name: name.trim(), usernames: [...selected] }); onClose(); }}>确认</button></div>
    </div></div>
  );
}

function ConversationPickerModal({ title, conversations, details, members, userCache, onlineMap, currentUID, lastMsgMap, mentionMap, onConfirm, onClose }: {
  title: string;
  conversations: UserConvDTO[];
  details: Record<number, ConversationDTO>;
  members: Record<number, MemberDTO[]>;
  userCache: Record<number, UserDTO>;
  onlineMap: Record<number, boolean>;
  currentUID: number;
  lastMsgMap: Record<number, string>;
  mentionMap?: Record<number, boolean>;
  onConfirm: (conversationID: number) => void;
  onClose: () => void;
}) {
  const [keyword, setKeyword] = useState('');
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const trimmedKeyword = keyword.trim().toLowerCase();
  const filtered = conversations.filter((conversation) => {
    if (!trimmedKeyword) return true;
    const detail = details[conversation.conversation_id];
    const titleText = detail?.name || `聊天 ${conversation.conversation_id}`;
    const lastMsg = lastMsgMap[conversation.conversation_id] || '';
    return `${titleText} ${lastMsg}`.toLowerCase().includes(trimmedKeyword);
  });

  return (
    <div className="modal-overlay" onClick={onClose}><div className="modal-card friend-picker-modal" onClick={(e) => e.stopPropagation()}>
      <div className="modal-header"><strong>{title}</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="modal-body">
        <label className="modal-field"><span>选择聊天</span><input value={keyword} placeholder="搜索聊天名称或消息内容" onChange={(e) => setKeyword(e.target.value)} /></label>
        <div className="friend-picker-list">
          {filtered.map((conversation) => (
            <ConversationSummaryRow
              key={conversation.conversation_id}
              conversation={conversation}
              detail={details[conversation.conversation_id]}
              members={members[conversation.conversation_id] ?? []}
              currentUID={currentUID}
              userCache={userCache}
              onlineMap={onlineMap}
              mentionMap={mentionMap}
              subtitle={lastMsgMap[conversation.conversation_id] || '暂无消息'}
              selected={selectedID === conversation.conversation_id}
              onClick={setSelectedID}
            />
          ))}
          {!filtered.length && <div className="empty-hint">没有可选择的聊天</div>}
        </div>
      </div>
      <div className="modal-actions"><button className="modal-cancel" onClick={onClose}>取消</button><button className="primary-btn" disabled={!selectedID} onClick={() => { if (!selectedID) return; onConfirm(selectedID); onClose(); }}>确认</button></div>
    </div></div>
  );
}

function ConfirmModal({ title, text, danger, onConfirm, onClose }: {
  title: string; text: string; danger?: boolean; onConfirm: () => void; onClose: () => void;
}) {
  return (
    <div className="modal-overlay" onClick={onClose}><div className="modal-card" onClick={(e) => e.stopPropagation()}>
      <div className="modal-header"><strong>{title}</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="modal-body"><p>{text}</p></div>
      <div className="modal-actions"><button className="modal-cancel" onClick={onClose}>取消</button><button className={danger ? 'primary-btn danger-btn' : 'primary-btn'} onClick={() => { onConfirm(); onClose(); }}>确认</button></div>
    </div></div>
  );
}

export function ContextMenuPopup({ menu, mine, onRevoke, onReply, onCopy, onDelete, onForward, onClose }: {
  menu: ContextMenuType; mine: boolean; onRevoke: () => void; onReply: () => void; onCopy: () => void; onDelete: () => void; onForward: () => void; onClose: () => void;
}) {
  if (!menu) return null;
  const canRevoke = mine && (Date.now() / 1000 - menu.message.created_at < 120);
  return (
    <ContextMenu x={menu.x} y={menu.y} onClose={onClose} items={[
      { label: '回复', action: onReply },
      { label: '复制', action: onCopy },
      { label: '转发', action: onForward },
      ...(canRevoke ? [{ label: '撤回', action: onRevoke, danger: true }] : []),
      { label: '删除', action: onDelete, danger: true },
    ]} />
  );
}
