import { ReactNode } from 'react';

export function CollapsibleSection({ title, count, open, onToggle, extraActions, small, children }: {
  title: string; count: number; open: boolean; onToggle: () => void; extraActions?: ReactNode; small?: boolean; children: ReactNode;
}) {
  return (
    <div className="border-b border-[#e4e7ec]">
      <button className={`flex w-full items-center gap-1.5 rounded-md px-2 text-left text-[#555f6d] hover:bg-[#eef1f5] ${small ? 'py-1.5 text-xs' : 'py-2 text-sm'}`} onClick={onToggle}>
        <span className={`text-[10px] transition-transform ${open ? 'rotate-90' : ''}`}>▸</span>
        <strong className="font-semibold">{title}</strong>
        <span className="text-xs text-[#9aa1ad]">{count}</span>
        {extraActions && <span className="ml-auto flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>{extraActions}</span>}
      </button>
      {open && <div className={small ? 'pl-3' : undefined}>{children}</div>}
    </div>
  );
}
