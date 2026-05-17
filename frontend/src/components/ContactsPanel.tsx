import { useEffect, useMemo, useRef, useState } from 'react';
import type { ConversationDTO, FriendDTO, FriendGroupDTO, FriendRequestDTO, UserConvDTO, UserDTO } from '@/api/types';
import { chatTitle } from '@/utils';
import { Avatar, Badge, CollapsibleSection, ContextMenu } from '@/components/ui';

export function ContactsPanel({ currentUID, keyword, setKeyword, onSearch, results, friends, friendGroups, requests, outgoingReqs, groupConversations, details, userCache, onlineMap, friendMap, onStartPrivate, onRequest, onHandleRequest, onSelectChat, onDeleteFriend, onUpdateRemark, onCreateGroup, onViewUser, onViewFriendRequests, onViewGroupManage }: {
  currentUID: number; keyword: string; setKeyword: (v: string) => void; onSearch: (v: string) => void; results: UserDTO[]; friends: FriendDTO[]; friendGroups: FriendGroupDTO[]; requests: FriendRequestDTO[]; outgoingReqs: FriendRequestDTO[]; groupConversations: UserConvDTO[]; details: Record<number, ConversationDTO>; userCache: Record<number, UserDTO>; onlineMap: Record<number, boolean>; friendMap?: Record<number, FriendDTO>;
  onStartPrivate: (uid: number) => void; onRequest: (user: UserDTO) => Promise<void>; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onSelectChat: (id: number) => void;
  onDeleteFriend: (uid: number) => void; onUpdateRemark: (uid: number, cur: string) => void; onCreateGroup: () => void;
  onViewUser?: (uid: number) => void;
  onViewFriendRequests?: () => void;
  onViewGroupManage?: () => void;
}) {
  const [contactsTab, setContactsTab] = useState<'friends' | 'groups'>('friends');
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({});
  const toggle = (k: string) => setOpenSections((p) => ({ ...p, [k]: !p[k] }));
  const [friendCtx, setFriendCtx] = useState<{ x: number; y: number; friend: FriendDTO } | null>(null);
  const [groupCtx, setGroupCtx] = useState<{ x: number; y: number } | null>(null);
  const [moreOpen, setMoreOpen] = useState(false);
  const noticeCount = requests.length + outgoingReqs.length;
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => onSearch(keyword), 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [keyword]);

  const friendsByGroup = useMemo(() => {
    const map: Record<number, FriendDTO[]> = {};
    for (const f of friends) { const gid = f.group_id || 0; if (!map[gid]) map[gid] = []; map[gid].push(f); }
    return map;
  }, [friends]);

  const sortedGroups = useMemo(() => {
    return [...friendGroups].sort((a, b) => a.sort_order - b.sort_order);
  }, [friendGroups]);

  function userFor(uid: number): UserDTO {
    return userCache[uid] ?? { id: uid, username: String(uid), nickname: `用户 ${uid}`, avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 };
  }

  function matchRemark(friend: FriendDTO, kw: string): boolean {
    if (!kw) return true;
    const lower = kw.toLowerCase();
    if (friend.remark && friend.remark.toLowerCase().includes(lower)) return true;
    const u = userFor(friend.friend_uid);
    if (u.nickname?.toLowerCase().includes(lower)) return true;
    if (u.username?.toLowerCase().includes(lower)) return true;
    return false;
  }

  const filteredFriendsByGroup = useMemo(() => {
    if (!keyword) return friendsByGroup;
    const result: Record<number, FriendDTO[]> = {};
    for (const [gid, list] of Object.entries(friendsByGroup)) {
      const filtered = list.filter((f) => matchRemark(f, keyword));
      if (filtered.length > 0) result[Number(gid)] = filtered;
    }
    return result;
  }, [friendsByGroup, keyword]);

  const filteredGroupConversations = useMemo(() => {
    if (!keyword) return groupConversations;
    const lower = keyword.toLowerCase();
    return groupConversations.filter((gc) => {
      const title = chatTitle(gc, details[gc.conversation_id]);
      return title.toLowerCase().includes(lower);
    });
  }, [groupConversations, keyword, details]);

  const hasSearchResults = keyword && (
    Object.values(filteredFriendsByGroup).some((l) => l.length > 0) ||
    filteredGroupConversations.length > 0
  );

  function renderFriendItem(item: FriendDTO) {
    const fu = userFor(item.friend_uid);
    const dn = item.remark || fu?.nickname || fu?.username || `用户 ${item.friend_uid}`;
    return (
      <div className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-inherit transition hover:bg-[#e8e8e8]" key={item.id} onClick={() => onViewUser?.(item.friend_uid)} onContextMenu={(e) => { e.preventDefault(); setFriendCtx({ x: e.clientX, y: e.clientY, friend: item }); }}>
        <Avatar user={fu} small online={onlineMap[item.friend_uid] ?? false} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-[#1a1a1a]">{dn}</div>
          {!item.remark && fu?.nickname && fu?.nickname !== dn && (
            <div className="truncate text-xs text-[#b0b5be]">{fu.nickname}</div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col overflow-hidden bg-[#f5f5f5]" onClick={() => { setFriendCtx(null); setGroupCtx(null); setMoreOpen(false); }}>
      <div className="shrink-0 border-b border-[#e8e8e8] px-3 py-2.5">
        <div className="flex items-center gap-2 rounded-lg bg-white px-3 py-2">
          <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="#b0b5be" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
          <input
            className="min-w-0 flex-1 bg-transparent text-[13px] text-[#1a1a1a] outline-none placeholder:text-[#b0b5be]"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索"
          />
          {keyword && (
            <button className="grid h-5 w-5 place-items-center rounded-full text-[#b0b5be] hover:bg-[#f0f0f0]" onClick={() => setKeyword('')}>
              <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
            </button>
          )}
        </div>
      </div>

      <button className="flex w-full items-center gap-3 border-b border-[#e8e8e8] px-3 py-2.5 text-left transition hover:bg-[#e8e8e8] mb-2" onClick={() => onViewFriendRequests?.()}>
        <span className="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-gradient-to-br from-[#19c37d] to-[#07a35a] text-base font-bold text-white">新</span>
        <div className="min-w-0 flex-1">
          <div className="text-sm font-medium text-[#1a1a1a]">新的朋友</div>
          <div className="text-xs text-[#999]">{noticeCount > 0 ? `${noticeCount} 条好友通知` : '好友申请与验证消息'}</div>
        </div>
        <Badge count={noticeCount} />
      </button>

      <div className="flex shrink-0 border-b border-[#e8e8e8] mb-1">
        <button
          className={`flex-1 py-2.5 text-center text-[13px] font-medium transition ${contactsTab === 'friends' ? 'border-b-2 border-[#07c160] text-[#07c160]' : 'text-[#858c98] hover:text-[#1a1a1a]'}`}
          onClick={() => setContactsTab('friends')}
        >好友</button>
        <button
          className={`flex-1 py-2.5 text-center text-[13px] font-medium transition ${contactsTab === 'groups' ? 'border-b-2 border-[#07c160] text-[#07c160]' : 'text-[#858c98] hover:text-[#1a1a1a]'}`}
          onClick={() => setContactsTab('groups')}
        >群聊</button>
      </div>

      <div className="flex-1 overflow-auto">
        {keyword && (
          <>
            {!hasSearchResults && (
              <div className="px-3 py-8 text-center text-[13px] text-[#b0b5be]">未找到相关结果</div>
            )}
            {Object.entries(filteredFriendsByGroup).map(([gid, list]) => {
              const g = friendGroups.find((fg) => fg.id === Number(gid));
              return (
                <div key={`sf-${gid}`}>
                  <div className="px-3 py-1.5 text-xs font-semibold text-[#858c98]">{g?.name || '默认分组'}</div>
                  {list.map(renderFriendItem)}
                </div>
              );
            })}
            {filteredGroupConversations.length > 0 && (
              <div>
                <div className="px-3 py-1.5 text-xs font-semibold text-[#858c98]">群聊</div>
                {filteredGroupConversations.map((item) => (
                  <button key={`sg-${item.conversation_id}`} className="flex w-full items-center gap-3 border-b border-[#f0f1f3] px-3 py-2.5 text-left transition hover:bg-[#e8e8e8]" onClick={() => onSelectChat(item.conversation_id)}>
                    <div className="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-[#12b35f] text-sm font-bold text-white">群</div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium text-[#1a1a1a]">{chatTitle(item, details[item.conversation_id])}</div>
                      <div className="text-xs text-[#999]">{details[item.conversation_id]?.member_count ?? 0} 位成员</div>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </>
        )}

        {!keyword && contactsTab === 'friends' && (
          <>
            <div className="flex items-center justify-between px-3 py-1.5">
              <span className="text-xs font-semibold text-[#858c98]">好友分组</span>
              <div className="relative">
                <button className="grid h-6 w-6 place-items-center rounded text-[#858c98] hover:bg-[#e0e0e0]" onClick={(e) => { e.stopPropagation(); setMoreOpen((v) => !v); }}>
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><circle cx="12" cy="5" r="1.5" /><circle cx="12" cy="12" r="1.5" /><circle cx="12" cy="19" r="1.5" /></svg>
                </button>
                {moreOpen && (
                  <div className="absolute right-0 top-full z-20 mt-1 min-w-[120px] rounded-lg border border-[#e8e8e8] bg-white py-1 shadow-lg" onClick={(e) => e.stopPropagation()}>
                    <button className="w-full px-4 py-2 text-left text-xs text-[#1a1a1a] hover:bg-[#f5f5f5]" onClick={() => { setMoreOpen(false); onCreateGroup(); }}>新建分组</button>
                    <button className="w-full px-4 py-2 text-left text-xs text-[#1a1a1a] hover:bg-[#f5f5f5]" onClick={() => { setMoreOpen(false); onViewGroupManage?.(); }}>管理分组</button>
                  </div>
                )}
              </div>
            </div>

            {sortedGroups.map((g) => (
              <CollapsibleSection key={g.id} title={g.name || '默认分组'} count={(friendsByGroup[g.id] ?? []).length} open={openSections[`fg_${g.id}`] ?? true} onToggle={() => toggle(`fg_${g.id}`)} small>
                {(friendsByGroup[g.id] ?? []).map(renderFriendItem)}
              </CollapsibleSection>
            ))}

            {(friendsByGroup[0] ?? []).length > 0 && !friendGroups.some((g) => g.id === 0) && (
              <CollapsibleSection title="默认分组" count={(friendsByGroup[0] ?? []).length} open={openSections.fg_0 ?? true} onToggle={() => toggle('fg_0')} small>
                {(friendsByGroup[0] ?? []).map(renderFriendItem)}
              </CollapsibleSection>
            )}

            {friends.length === 0 && <div className="px-3 py-8 text-center text-[13px] text-[#b0b5be]">暂无好友，搜索账号添加</div>}
          </>
        )}

        {!keyword && contactsTab === 'groups' && (
          <>
            {filteredGroupConversations.map((item) => (
              <button key={item.conversation_id} className="flex w-full items-center gap-3 border-b border-[#f0f1f3] px-3 py-2.5 text-left transition hover:bg-[#e8e8e8]" onClick={() => onSelectChat(item.conversation_id)}>
                <div className="grid h-9 w-9 shrink-0 place-items-center rounded-xl bg-[#12b35f] text-sm font-bold text-white">群</div>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium text-[#1a1a1a]">{chatTitle(item, details[item.conversation_id])}</div>
                  <div className="text-xs text-[#999]">{details[item.conversation_id]?.member_count ?? 0} 位成员</div>
                </div>
              </button>
            ))}
            {groupConversations.length === 0 && (
              <div className="px-3 py-8 text-center text-[13px] text-[#b0b5be]">暂无群聊</div>
            )}
          </>
        )}
      </div>

      {friendCtx && <ContextMenu x={friendCtx.x} y={friendCtx.y} onClose={() => setFriendCtx(null)} items={[
        { label: '发消息', action: () => onStartPrivate(friendCtx.friend.friend_uid) },
        { label: '修改备注', action: () => onUpdateRemark(friendCtx.friend.friend_uid, friendCtx.friend.remark) },
        { label: '删除好友', action: () => onDeleteFriend(friendCtx.friend.friend_uid), danger: true },
      ]} />}

      {groupCtx && <ContextMenu x={groupCtx.x} y={groupCtx.y} onClose={() => setGroupCtx(null)} items={[
        { label: '分组管理', action: () => onViewGroupManage?.() },
      ]} />}
    </div>
  );
}