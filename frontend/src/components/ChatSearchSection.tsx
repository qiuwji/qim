import type { FriendDTO, MessageDTO, UserDTO } from '@/api/types';
import { displayName } from '@/utils';
import { SearchInput } from '@/components/ui';

export function ChatSearchSection({ value, onChange, onSearch, results, userCache, friendMap, onJumpToMessage }: {
  value: string; onChange: (v: string) => void; onSearch: () => void; results: MessageDTO[]; userCache: Record<number, UserDTO>; friendMap?: Record<number, FriendDTO>; onJumpToMessage: (id: number) => void;
}) {
  return (
    <div className="detail-card">
      <div className="card-title"><strong>搜索聊天记录</strong></div>
      <SearchInput value={value} onChange={onChange} placeholder="输入关键词" onSearch={onSearch} />
      {results.length > 0 && (
        <div className="grid gap-1.5 max-h-44 overflow-auto">
          {results.map((m) => (
            <button key={m.id} type="button" className="grid gap-0.5 p-2 text-left rounded-lg bg-[#f3f4f6] hover:bg-[#eceff3]" onClick={() => onJumpToMessage(m.id)}>
              <strong className="text-[13px] text-[#1f2329]">{displayName(m.sender_id, userCache, friendMap)}</strong>
              <span className="text-[#858c98] text-xs overflow-hidden text-ellipsis whitespace-nowrap">{m.content.slice(0, 60)}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}