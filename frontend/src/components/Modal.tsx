import { useState } from 'react';
import type { MessageDTO } from '../api/types';
import type { ContextMenu, ModalState } from '../types';

export function AppModal({ modal, onClose }: { modal: ModalState; onClose: () => void }) {
  if (!modal) return null;
  if (modal.type === 'prompt') return <PromptModal fields={modal.fields} title={modal.title} onConfirm={modal.onConfirm} onClose={onClose} />;
  if (modal.type === 'confirm') return <ConfirmModal title={modal.title} text={modal.text} danger={modal.danger} onConfirm={modal.onConfirm} onClose={onClose} />;
  return null;
}

function PromptModal({ title, fields, onConfirm, onClose }: {
  title: string; fields: { key: string; label: string; placeholder?: string; defaultValue?: string }[]; onConfirm: (v: Record<string, string>) => void; onClose: () => void;
}) {
  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    for (const f of fields) init[f.key] = f.defaultValue ?? '';
    return init;
  });
  return (
    <div className="modal-overlay" onClick={onClose}><div className="modal-card" onClick={(e) => e.stopPropagation()}>
      <div className="modal-header"><strong>{title}</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="modal-body">{fields.map((f) => (<label key={f.key} className="modal-field"><span>{f.label}</span><input value={values[f.key] ?? ''} placeholder={f.placeholder} onChange={(e) => setValues((p) => ({ ...p, [f.key]: e.target.value }))} /></label>))}</div>
      <div className="modal-actions"><button className="modal-cancel" onClick={onClose}>取消</button><button className="primary-btn" onClick={() => { onConfirm(values); onClose(); }}>确认</button></div>
    </div></div>
  );
}

function ConfirmModal({ title, text, danger, onConfirm, onClose }: {
  title: string; text: string; danger?: boolean; onConfirm: () => void; onClose: () => void;
}) {
  return (
    <div className="modal-overlay" onClick={onClose}><div className="modal-card" onClick={(e) => e.stopPropagation()}>
      <div className="modal-header"><strong>{title}</strong><button className="modal-close" onClick={onClose}>✕</button></div>
      <div className="modal-body"><p>{text}</p></div>
      <div className="modal-actions"><button className="modal-cancel" onClick={onClose}>取消</button><button className={danger ? 'primary-btn danger-btn' : 'primary-btn'} onClick={() => { onConfirm(); onClose(); }}>确认</button></div>
    </div></div>
  );
}

export function ContextMenuPopup({ menu, mine, onRevoke, onReply, onCopy, onDelete, onForward, onClose }: {
  menu: ContextMenu; mine: boolean; onRevoke: () => void; onReply: () => void; onCopy: () => void; onDelete: () => void; onForward: () => void; onClose: () => void;
}) {
  if (!menu) return null;
  const canRevoke = mine && (Date.now() / 1000 - menu.message.created_at < 120);
  const items = [
    { label: '回复', action: onReply },
    { label: '复制', action: onCopy },
    { label: '转发', action: onForward },
    ...(canRevoke ? [{ label: '撤回', action: onRevoke }] : []),
    { label: '删除', action: onDelete },
  ];
  return (
    <div className="context-overlay" onClick={onClose}>
      <div className="context-menu" style={{ left: menu.x, top: menu.y }} onClick={(e) => e.stopPropagation()}>
        {items.map((item, i) => (<button key={i} className={`ctx-item ${item.label === '删除' || item.label === '撤回' ? 'danger' : ''}`} onClick={() => { item.action(); onClose(); }}>{item.label}</button>))}
      </div>
    </div>
  );
}
