import { ReactNode } from 'react';

export function ContextMenu({ x, y, items, onClose }: { x: number; y: number; items: { label: string; action: () => void; danger?: boolean }[]; onClose: () => void }) {
  return (
    <div className="ctx-overlay" onClick={onClose}>
      <ul className="ctx-menu" style={{ left: x, top: y }} onClick={(e) => e.stopPropagation()}>
        {items.map((item, i) => (
          <li key={i} className={item.danger ? 'danger' : ''} onClick={() => { item.action(); onClose(); }}>{item.label}</li>
        ))}
      </ul>
    </div>
  );
}

export function OverlayMenu({ x, y, children, onClose }: { x: number; y: number; children: ReactNode; onClose: () => void }) {
  return (
    <div className="ctx-overlay" onClick={onClose}>
      <div className="ctx-menu" style={{ left: x, top: y }} onClick={(e) => e.stopPropagation()}>
        {children}
      </div>
    </div>
  );
}
