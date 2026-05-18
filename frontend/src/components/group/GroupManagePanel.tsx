import type { FriendGroupDTO } from '@/api/types';
import { EmptyState, PanelHeader } from '@/components/ui';

export function GroupManagePanel({ groups, onBack, onRenameGroup, onDeleteGroup }: {
  groups: FriendGroupDTO[]; onBack: () => void;
  onRenameGroup: (id: number, cur: string) => void; onDeleteGroup: (id: number) => void;
}) {
  return (
    <div className="flex h-full flex-col bg-[#f3f4f6]">
      <PanelHeader title="分组管理" onBack={onBack} />
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
        {groups.length === 0 && <EmptyState title="暂无分组" text="还没有创建任何好友分组" />}
      </div>
    </div>
  );
}
