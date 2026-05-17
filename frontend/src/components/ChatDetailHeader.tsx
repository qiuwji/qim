export function ChatDetailHeader({ title, onClose }: {
  title: string; onClose: () => void;
}) {
  return (
    <div className="detail-header">
      <strong>{title}</strong>
      <button className="modal-close" onClick={onClose}>✕</button>
    </div>
  );
}