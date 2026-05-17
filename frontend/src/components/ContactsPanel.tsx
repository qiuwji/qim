import { ReactNode, useMemo, useState } from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle } from '@/utils';
import { Avatar, Badge, CollapsibleSection, ContextMenu, SearchInput } from '@/components/ui';

const contactRow = 'flex w-full items-center justify-between gap-2.5 rounded-lg bg-transparent px-2.5 py-3 text-left text-inherit transition hover:bg-[#e8e8e8]';
const contactText = 'grid min-w-0 flex-1 gap-0.5 text-left [&>strong]:truncate [&>strong]:text-sm [&>strong]:font-semibold [&>strong]:text-[#1f2329] [&>span]:truncate [&>span]:text-xs [&>span]:text-[#858c98]';
const shortcutAvatar = 'grid h-9 w-9 shrink-0 place-items-center rounded-[10px] text-sm font-semibold text-white';
const actionButton = 'rounded-md bg-[#eef1f5] px-2.5 py-1.5 text-xs text-[#555f6d] hover:bg-[#e0e0e0]';
const primaryActionButton = `${actionButton} bg-[rgba(7,193,96,0.08)] text-[#07c160] hover:bg-[rgba(7,193,96,0.16)]`;

export function ContactsPanel({ currentUID, keyword, setKeyword, onSearch, results, friends, friendGroups, requests, outgoingReqs, groupConversations, details, userCache, onlineMap, friendMap, onStartPrivate, onRequest, onHandleRequest, onSelectChat, onDeleteFriend, onUpdateRemark, onCreateGroup, onRenameGroup, onDeleteGroup, onViewUser }: {
  currentUID: number; keyword: string; setKeyword: (v: string) => void; onSearch: () => void; results: UserDTO[]; friends: FriendDTO[]; friendGroups: FriendGroupDTO[]; requests: FriendRequestDTO[]; outgoingReqs: FriendRequestDTO[]; groupConversations: UserConvDTO[]; details: Record<number, ConversationDTO>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>;
  onStartPrivate: (uid: number) => void; onRequest: (user: UserDTO) => Promise<void>; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onSelectChat: (id: number) => void;
  onDeleteFriend: (uid: number) => void; onUpdateRemark: (uid: number, cur: string) => void; onCreateGroup: () => void; onRenameGroup: (id: number, cur: string) => void; onDeleteGroup: (id: number) => void;
  onViewUser?: (uid: number) => void;
}) {
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({ search: true, notices: false, groups: false, fg_0: true });
  const toggle = (k: string) => setOpenSections((p) => ({ ...p, [k]: !p[k] }));
  const [friendCtx, setFriendCtx] = useState<{ x: number; y: number; friend: FriendDTO } | null>(null);
  const noticeCount = requests.length + outgoingReqs.length;

  const friendsByGroup = useMemo(() => {
    const map: Record<number, FriendDTO[]> = { 0: [] };
    for (const f of friends) { const gid = f.group_id || 0; if (!map[gid]) map[gid] = []; map[gid].push(f); }
    return map;
  }, [friends]);

  function userFor(uid: number): UserDTO {
    return userCache[uid] ?? { id: uid, username: String(uid), nickname: `用户 ${uid}`, avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 };
  }

  function renderFriendItem(item: FriendDTO) {
    const fu = userFor(item.friend_uid);
    const dn = item.remark || fu?.nickname || fu?.username || `用户 ${item.friend_uid}`;
    return (
      <div className={contactRow} key={item.id} onClick={() => onViewUser?.(item.friend_uid)} onContextMenu={(e) => { e.preventDefault(); setFriendCtx({ x: e.clientX, y: e.clientY, friend: item }); }}>
        <Avatar user={fu} small online={onlineMap[item.friend_uid] ?? false} />
        <div className={contactText}>
          <strong>{dn}</strong>
          <span>{fu?.sign || `@${fu?.username}`}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="overflow-auto px-2 py-1" onClick={() => setFriendCtx(null)}>
      <div className="search-box">
        <SearchInput value={keyword} onChange={setKeyword} placeholder="搜索用户名或昵称" onSearch={onSearch} />
      </div>
      {results.length > 0 && <CollapsibleSection title="搜索结果" count={results.length} open={openSections.search ?? true} onToggle={() => toggle('search')}>{results.map((item) => { const isFriend = !!friendMap?.[item.id]; const isSelf = item.id === currentUID; return (<UserLine key={item.id} user={item} online={isFriend ? onlineMap[item.id] ?? false : undefined} actions={<>{!isSelf && isFriend && <button className={actionButton} onClick={() => onStartPrivate(item.id)}>聊天</button>}{!isSelf && !isFriend && <button className={primaryActionButton} onClick={() => void onRequest(item)}>加好友</button>}</>} onViewUser={onViewUser} />); })}</CollapsibleSection>}
      <div className="grid gap-0.5 border-b-8 border-[#f2f3f5] pt-1 pb-2">
        <button className={`${contactRow} rounded-none py-3`} onClick={() => toggle('notices')}>
          <span className={`${shortcutAvatar} bg-gradient-to-br from-[#19c37d] to-[#07a35a]`}>新</span>
          <span className={contactText}><strong>新的朋友</strong><span>{noticeCount > 0 ? `${noticeCount} 条好友通知` : '好友申请与验证消息'}</span></span>
          <Badge count={noticeCount} />
        </button>
        <button className={`${contactRow} rounded-none py-3`} onClick={() => toggle('groups')}>
          <span className={`${shortcutAvatar} bg-gradient-to-br from-[#4c8dff] to-[#2f65d9]`}>群</span>
          <span className={contactText}><strong>群聊</strong><span>{groupConversations.length > 0 ? `${groupConversations.length} 个群聊` : '暂无群聊'}</span></span>
        </button>
      </div>
      {openSections.notices && <NoticeSection requests={requests} outgoingReqs={outgoingReqs} userFor={userFor} onHandleRequest={onHandleRequest} onViewUser={onViewUser} />}
      {openSections.groups && <GroupSection groupConversations={groupConversations} details={details} onSelectChat={onSelectChat} />}
      <div className="contacts-divider"><span>好友</span><button className="tiny-btn" onClick={onCreateGroup}>新建分组</button></div>
      {friendGroups.map((g) => (<CollapsibleSection key={g.id} title={g.name || '默认分组'} count={(friendsByGroup[g.id] ?? []).length} open={openSections[`fg_${g.id}`] ?? true} onToggle={() => toggle(`fg_${g.id}`)} extraActions={<>{g.id > 0 && <><button className="tiny-btn" onClick={() => onRenameGroup(g.id, g.name)}>改名</button><button className="tiny-btn danger-text" onClick={() => onDeleteGroup(g.id)}>删除</button></>}</>}>
        {(friendsByGroup[g.id] ?? []).map(renderFriendItem)}
      </CollapsibleSection>))}
      {(friendsByGroup[0] ?? []).length > 0 && <CollapsibleSection title="默认分组" count={(friendsByGroup[0] ?? []).length} open={openSections.fg_0 ?? true} onToggle={() => toggle('fg_0')}>
        {(friendsByGroup[0] ?? []).map(renderFriendItem)}
      </CollapsibleSection>}
      {friends.length === 0 && <div className="empty-hint">暂无好友，搜索用户名添加</div>}
      {friendCtx && <ContextMenu x={friendCtx.x} y={friendCtx.y} onClose={() => setFriendCtx(null)} items={[
        { label: '发消息', action: () => onStartPrivate(friendCtx.friend.friend_uid) },
        { label: '修改备注', action: () => onUpdateRemark(friendCtx.friend.friend_uid, friendCtx.friend.remark) },
        { label: '删除好友', action: () => onDeleteFriend(friendCtx.friend.friend_uid), danger: true },
      ]} />}
    </div>
  );
}

function UserLine({ user, online, actions, onViewUser }: { user: UserDTO; online?: boolean; actions: ReactNode; onViewUser?: (uid: number) => void }) {
  return (<div className={contactRow} onClick={() => onViewUser?.(user.id)}><Avatar user={user} online={online} /><div className={contactText}><strong>{user.nickname || user.username}</strong><span>{user.sign || `@${user.username}`}</span></div><div className="flex shrink-0 gap-1.5">{actions}</div></div>);
}

function NoticeSection({ requests, outgoingReqs, userFor, onHandleRequest, onViewUser }: {
  requests: FriendRequestDTO[]; outgoingReqs: FriendRequestDTO[]; userFor: (uid: number) => UserDTO; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onViewUser?: (uid: number) => void;
}) {
  const noticeCount = requests.length + outgoingReqs.length;
  return (
    <div className="border-b-8 border-[#f2f3f5] px-2 py-1">
      {requests.map((item) => { const fu = userFor(item.from_uid); return (<div className={contactRow} key={`in-${item.id}`}><div className="flex min-w-0 flex-1 cursor-pointer items-center gap-2.5 hover:opacity-85" onClick={() => onViewUser?.(item.from_uid)}><Avatar user={fu} small /><div className={contactText}><strong>{fu.nickname || `用户 ${item.from_uid}`}</strong><span>{item.message || '请求添加你为好友'}</span></div></div><div className="flex shrink-0 gap-1.5"><button className={primaryActionButton} onClick={() => onHandleRequest(item.id, 'accept')}>同意</button><button className={actionButton} onClick={() => onHandleRequest(item.id, 'reject')}>拒绝</button></div></div>); })}
      {outgoingReqs.map((item) => { const tu = userFor(item.to_uid); return (<div className={contactRow} key={`out-${item.id}`}><div className="flex min-w-0 flex-1 cursor-pointer items-center gap-2.5 hover:opacity-85" onClick={() => onViewUser?.(item.to_uid)}><Avatar user={tu} small /><div className={contactText}><strong>{tu.nickname || `用户 ${item.to_uid}`}</strong><span>等待对方同意</span></div></div></div>); })}
      {noticeCount === 0 && <div className="empty-hint">暂无好友通知</div>}
    </div>
  );
}

function GroupSection({ groupConversations, details, onSelectChat }: {
  groupConversations: UserConvDTO[]; details: Record<number, ConversationDTO>; onSelectChat: (id: number) => void;
}) {
  return (
    <div className="border-b-8 border-[#f2f3f5] px-2 py-1">
      {groupConversations.map((item) => (<button key={item.conversation_id} className={contactRow} onClick={() => onSelectChat(item.conversation_id)}><div className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-[#12b35f] text-sm font-bold text-white">群</div><div className={contactText}><strong>{chatTitle(item, details[item.conversation_id])}</strong><span>{details[item.conversation_id]?.member_count ?? 0} 位成员</span></div></button>))}
      {groupConversations.length === 0 && <div className="empty-hint">暂无群聊</div>}
    </div>
  );
}
