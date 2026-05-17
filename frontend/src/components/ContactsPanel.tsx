import { ReactNode, useEffect, useMemo, useRef, useState } from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle } from '@/utils';
import { Avatar, Badge, CollapsibleSection, ContextMenu } from '@/components/ui';

export function ContactsPanel({ currentUID, keyword, setKeyword, onSearch, results, friends, friendGroups, requests, outgoingReqs, groupConversations, details, userCache, onlineMap, friendMap, onStartPrivate, onRequest, onHandleRequest, onSelectChat, onDeleteFriend, onUpdateRemark, onCreateGroup, onRenameGroup, onDeleteGroup, onViewUser, onViewFriendRequests }: {
  currentUID: number; keyword: string; setKeyword: (v: string) => void; onSearch: (v: string) => void; results: UserDTO[]; friends: FriendDTO[]; friendGroups: FriendGroupDTO[]; requests: FriendRequestDTO[]; outgoingReqs: FriendRequestDTO[]; groupConversations: UserConvDTO[]; details: Record<number, ConversationDTO>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>;
  onStartPrivate: (uid: number) => void; onRequest: (user: UserDTO) => Promise<void>; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onSelectChat: (id: number) => void;
  onDeleteFriend: (uid: number) => void; onUpdateRemark: (uid: number, cur: string) => void; onCreateGroup: () => void; onRenameGroup: (id: number, cur: string) => void; onDeleteGroup: (id: number) => void;
  onViewUser?: (uid: number) => void;
  onViewFriendRequests?: () => void;
}) {
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({ groups: false, fg_0: true });
  const toggle = (k: string) => setOpenSections((p) => ({ ...p, [k]: !p[k] }));
  const [friendCtx, setFriendCtx] = useState<{ x: number; y: number; friend: FriendDTO } | null>(null);
  const noticeCount = requests.length + outgoingReqs.length;
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => onSearch(keyword), 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [keyword]);

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
      <div className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-inherit transition hover:bg-[#e8e8e8]" key={item.id} onClick={() => onViewUser?.(item.friend_uid)} onContextMenu={(e) => { e.preventDefault(); setFriendCtx({ x: e.clientX, y: e.clientY, friend: item }); }}>
        <Avatar user={fu} small online={onlineMap[item.friend_uid] ?? false} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-[#1a1a1a]">{dn}</div>
          <div className="truncate text-xs text-[#999]">{fu?.sign || ''}</div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-hidden bg-[#f5f5f5]" onClick={() => setFriendCtx(null)}>
      <div className="shrink-0 border-b border-[#e8e8e8] px-3 py-2.5">
        <div className="flex items-center gap-2 rounded-lg bg-white px-3 py-2">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="#b0b5be" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
          <input
            className="min-w-0 flex-1 bg-transparent text-sm text-[#1a1a1a] outline-none placeholder:text-[#b0b5be]"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索用户名或昵称"
          />
          {keyword && (
            <button className="grid h-5 w-5 place-items-center rounded-full text-[#b0b5be] hover:bg-[#f0f0f0]" onClick={() => setKeyword('')}>
              <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
            </button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {keyword && results.length > 0 && (
          <div className="border-b-8 border-[#f2f3f5]">
            <div className="px-3 py-2 text-xs font-semibold text-[#858c98]">搜索结果</div>
            {results.map((item) => {
              const isFriend = !!friendMap?.[item.id];
              const isSelf = item.id === currentUID;
              return (
                <div key={item.id} className="flex items-center gap-3 px-3 py-2.5 hover:bg-[#e8e8e8] transition" onClick={() => onViewUser?.(item.id)}>
                  <Avatar user={item} small online={isFriend ? onlineMap[item.id] ?? false : undefined} />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium text-[#1a1a1a]">{item.nickname || item.username}</div>
                    <div className="truncate text-xs text-[#999]">{item.sign || `@${item.username}`}</div>
                  </div>
                  <div className="flex shrink-0 gap-1.5">
                    {!isSelf && isFriend && <button className="rounded-md bg-[#eef1f5] px-2.5 py-1.5 text-xs text-[#555f6d] hover:bg-[#e0e0e0]" onClick={(e) => { e.stopPropagation(); onStartPrivate(item.id); }}>发消息</button>}
                    {!isSelf && !isFriend && <button className="rounded-md bg-[rgba(7,193,96,0.08)] px-2.5 py-1.5 text-xs text-[#07c160] hover:bg-[rgba(7,193,96,0.16)]" onClick={(e) => { e.stopPropagation(); void onRequest(item); }}>加好友</button>}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {keyword && results.length === 0 && (
          <div className="px-3 py-8 text-center text-sm text-[#b0b5be]">未找到相关用户</div>
        )}

        {!keyword && (
          <>
            <div className="border-b-8 border-[#f2f3f5]">
              <button className="flex w-full items-center gap-3 px-3 py-3 text-left transition hover:bg-[#e8e8e8]" onClick={() => onViewFriendRequests?.()}>
                <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-[#19c37d] to-[#07a35a] text-lg font-bold text-white">新</span>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium text-[#1a1a1a]">新的朋友</div>
                  <div className="text-xs text-[#999]">{noticeCount > 0 ? `${noticeCount} 条好友通知` : '好友申请与验证消息'}</div>
                </div>
                <Badge count={noticeCount} />
              </button>
              <button className="flex w-full items-center gap-3 px-3 py-3 text-left transition hover:bg-[#e8e8e8]" onClick={() => toggle('groups')}>
                <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-[#4c8dff] to-[#2f65d9] text-lg font-bold text-white">群</span>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium text-[#1a1a1a]">群聊</div>
                  <div className="text-xs text-[#999]">{groupConversations.length > 0 ? `${groupConversations.length} 个群聊` : '暂无群聊'}</div>
                </div>
              </button>
            </div>

            {openSections.groups && (
              <div className="border-b-8 border-[#f2f3f5]">
                {groupConversations.map((item) => (
                  <button key={item.conversation_id} className="flex w-full items-center gap-3 px-3 py-2.5 text-left transition hover:bg-[#e8e8e8]" onClick={() => onSelectChat(item.conversation_id)}>
                    <div className="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-[#12b35f] text-base font-bold text-white">群</div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium text-[#1a1a1a]">{chatTitle(item, details[item.conversation_id])}</div>
                      <div className="text-xs text-[#999]">{details[item.conversation_id]?.member_count ?? 0} 位成员</div>
                    </div>
                  </button>
                ))}
                {groupConversations.length === 0 && <div className="px-3 py-4 text-center text-xs text-[#b0b5be]">暂无群聊</div>}
              </div>
            )}

            <div className="flex items-center justify-between px-3 py-2.5">
              <span className="text-xs font-semibold text-[#858c98]">好友</span>
              <button className="rounded-md px-2 py-1 text-xs text-[#1677c7] hover:bg-[#eef6ff]" onClick={onCreateGroup}>新建分组</button>
            </div>

            {friendGroups.map((g) => (
              <CollapsibleSection key={g.id} title={g.name || '默认分组'} count={(friendsByGroup[g.id] ?? []).length} open={openSections[`fg_${g.id}`] ?? true} onToggle={() => toggle(`fg_${g.id}`)} extraActions={g.id > 0 ? <><button className="rounded px-1.5 py-0.5 text-xs text-[#1677c7] hover:bg-[#eef6ff]" onClick={(e) => { e.stopPropagation(); onRenameGroup(g.id, g.name); }}>改名</button><button className="rounded px-1.5 py-0.5 text-xs text-[#e04344] hover:bg-[#fff1f0]" onClick={(e) => { e.stopPropagation(); onDeleteGroup(g.id); }}>删除</button></> : undefined}>
                {(friendsByGroup[g.id] ?? []).map(renderFriendItem)}
              </CollapsibleSection>
            ))}

            {(friendsByGroup[0] ?? []).length > 0 && !friendGroups.some((g) => g.id === 0) && (
              <CollapsibleSection title="默认分组" count={(friendsByGroup[0] ?? []).length} open={openSections.fg_0 ?? true} onToggle={() => toggle('fg_0')}>
                {(friendsByGroup[0] ?? []).map(renderFriendItem)}
              </CollapsibleSection>
            )}

            {friends.length === 0 && <div className="px-3 py-8 text-center text-sm text-[#b0b5be]">暂无好友，搜索用户名添加</div>}
          </>
        )}
      </div>

      {friendCtx && <ContextMenu x={friendCtx.x} y={friendCtx.y} onClose={() => setFriendCtx(null)} items={[
        { label: '发消息', action: () => onStartPrivate(friendCtx.friend.friend_uid) },
        { label: '修改备注', action: () => onUpdateRemark(friendCtx.friend.friend_uid, friendCtx.friend.remark) },
        { label: '删除好友', action: () => onDeleteFriend(friendCtx.friend.friend_uid), danger: true },
      ]} />}
    </div>
  );
}