import { ReactNode } from 'react';

export function CollapsibleSection({ title, count, open, onToggle, extraActions, children }: {
  title: string; count: number; open: boolean; onToggle: () => void; extraActions?: ReactNode; children: ReactNode;
}) {
  return (
    <div className="collapsible-section">
      <button className="section-toggle" onClick={onToggle}>
        <span className={`chevron ${open ? 'open' : ''}`}>▸</span>
        <strong>{title}</strong>
        <span className="section-count">{count}</span>
        {extraActions && <span className="section-actions" onClick={(e) => e.stopPropagation()}>{extraActions}</span>}
      </button>
      {open && <div className="section-content">{children}</div>}
    </div>
  );
}
