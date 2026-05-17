import { useMemo } from 'react';
import type { FriendRequestDTO, UserDTO } from '@/api/types';
import { Avatar } from '@/components/ui';

const STATUS_LABELS: Record<number, string> = { 0: '待处理', 1: '已同意', 2: '已拒绝' };
const STATUS_COLORS: Record<number, string> = { 0: 'bg-[#fdf6ec] text-[#e6a23c]', 1: 'bg-[#e8f8ef] text-[#07c160]', 2: 'bg-[#fef0f0] text-[#e04344]' };

export function FriendRequestsView({ currentUID, incoming, outgoing, userCache, onBack, onHandleRequest, onViewUser }: {
  currentUID: number; incoming: FriendRequestDTO[]; outgoing: FriendRequestDTO[]; userCache: Record<number, UserDTO>;
  onBack: () => void; onHandleRequest: (id: number, a: 'accept' | 'reject') => void; onViewUser?: (uid: number) => void;
}) {
  const allItems = useMemo(() => {
    const items: { type: 'incoming' | 'outgoing'; req: FriendRequestDTO; uid: number }[] = [];
    for (const r of incoming) items.push({ type: 'incoming', req: r, uid: r.from_uid });
    for (const r of outgoing) items.push({ type: 'outgoing', req: r, uid: r.to_uid });
    items.sort((a, b) => b.req.created_at - a.req.created_at);
    return items;
  }, [incoming, outgoing]);

  function userFor(uid: number): UserDTO {
    return userCache[uid] ?? { id: uid, username: String(uid), nickname: `用户 ${uid}`, avatar: '', sign: '', status: 0, created_at: 0, last_online_at: 0 };
  }

  return (
    <div className="flex h-full flex-col bg-[#f3f4f6]">
      <header className="flex shrink-0 items-center gap-3 border-b border-[#dfe3e8] bg-[#f9fafb] px-4 py-3">
        <button className="back-btn" onClick={onBack}>
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="15 18 9 12 15 6" /></svg>
        </button>
        <div>
          <strong className="text-base font-semibold text-[#1a1a1a]">新的朋友</strong>
          <div className="text-xs text-[#999]">好友申请记录</div>
        </div>
      </header>
      <div className="flex-1 overflow-auto">
        {allItems.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-3 py-16 text-[#b0b5be]">
            <svg viewBox="0 0 24 24" width="48" height="48" fill="none" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>
            <span className="text-sm">暂无好友申请记录</span>
          </div>
        )}
        {allItems.map(({ type, req, uid }) => {
          const u = userFor(uid);
          const isPending = req.status === 0;
          return (
            <div key={`${type}-${req.id}`} className="flex items-center gap-3 border-b border-[#f0f1f3] bg-white px-4 py-3">
              <div className="cursor-pointer" onClick={() => onViewUser?.(uid)}>
                <Avatar user={u} small />
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-[#1a1a1a]">{u.nickname || u.username || `用户 ${uid}`}</div>
                <div className="truncate text-xs text-[#999]">
                  {type === 'incoming' ? (req.message || '请求添加你为好友') : '你发送了好友申请'}
                  <span className="ml-2 text-[#b0b5be]">{new Date(req.created_at * 1000).toLocaleDateString('zh-CN')}</span>
                </div>
              </div>
              <span className={`shrink-0 rounded-md px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[req.status] || 'bg-[#f0f1f3] text-[#999]'}`}>{STATUS_LABELS[req.status] || '未知'}</span>
              {isPending && type === 'incoming' && (
                <div className="flex shrink-0 gap-1.5">
                  <button className="rounded-md bg-[rgba(7,193,96,0.08)] px-3 py-1.5 text-xs font-medium text-[#07c160] hover:bg-[rgba(7,193,96,0.16)]" onClick={() => onHandleRequest(req.id, 'accept')}>同意</button>
                  <button className="rounded-md bg-[#eef1f5] px-3 py-1.5 text-xs text-[#555f6d] hover:bg-[#e0e0e0]" onClick={() => onHandleRequest(req.id, 'reject')}>拒绝</button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}