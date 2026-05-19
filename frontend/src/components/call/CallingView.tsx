interface CallingViewProps {
  onCancel: () => void;
}

export function CallingView({ onCancel }: CallingViewProps) {
  return (
    <div className="call-overlay">
      <div className="call-modal">
        <div className="call-ringing-dots">
          <span /><span /><span />
        </div>
        <div className="call-info">
          <strong>呼叫中...</strong>
        </div>
        <div className="call-actions">
          <button className="call-btn-reject" onClick={onCancel}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="1" y1="1" x2="23" y2="23" /><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6A19.79 19.79 0 0 1 2 4.18 2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z" /></svg>
          </button>
        </div>
        <div className="call-hint">取消</div>
      </div>
    </div>
  );
}
