import type { FriendGroupDTO } from '@/api/types';

export function GroupManagePanel({ groups, onBack, onRenameGroup, onDeleteGroup }: {
  groups: FriendGroupDTO[]; onBack: () => void;
  onRenameGroup: (id: number, cur: string) => void; onDeleteGroup: (id: number) => void;
}) {
  return (
    <div className="flex h-full flex-col bg-[#f3f4f6]">
      <header className="flex shrink-0 items-center gap-3 border-b border-[#dfe3e8] bg-[#f9fafb] px-4 py-3">
        <button className="back-btn" onClick={onBack}>
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="15 18 9 12 15 6" /></svg>
        </button>
        <strong className="text-base font-semibold text-[#1a1a1a]">分组管理</strong>
      </header>
      <div className="flex-1 overflow-auto">
        {groups.map((g) => (
          <div key={g.id} className="flex items-center justify-between border-b border-[#f0f1f3] bg-white px-4 py-3">
            <span className="text-sm font-medium text-[#1a1a1a]">{g.name || '默认分组'}</span>
            <div className="flex gap-2">
              {g.id > 0 && (
                <>
                  <button
                    className="rounded-md px-3 py-1.5 text-xs text-[#1677c7] hover:bg-[#eef6ff]"
                    onClick={() => onRenameGroup(g.id, g.name)}
                  >重命名</button>
                  <button
                    className="rounded-md px-3 py-1.5 text-xs text-[#e04344] hover:bg-[#fff1f0]"
                    onClick={() => onDeleteGroup(g.id)}
                  >删除</button>
                </>
              )}
              {g.id === 0 && <span className="text-xs text-[#b0b5be]">默认分组不可操作</span>}
            </div>
          </div>
        ))}
        {groups.length === 0 && (
          <div className="py-16 text-center text-sm text-[#b0b5be]">暂无分组</div>
        )}
      </div>
    </div>
  );
}