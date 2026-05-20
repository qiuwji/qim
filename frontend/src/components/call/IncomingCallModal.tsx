import type { ActiveCall } from '@/hooks/useCallStore';
import { callTypeLabel } from '@/hooks/chat/models/callModel';
import { Avatar } from '@/components/ui/Avatar';

interface IncomingCallModalProps {
  call: ActiveCall;
  peerName: string;
  peerAvatar: string;
  onAccept: () => void;
  onReject: () => void;
}

function makeUser(name: string, avatar: string) {
  return { id: 0, username: name, nickname: name, avatar, sign: '', status: 0, created_at: 0, last_online_at: 0 };
}

export function IncomingCallModal({ call, peerName, peerAvatar, onAccept, onReject }: IncomingCallModalProps) {
  return (
    <div className="call-overlay">
      <div className="call-modal">
        <Avatar user={makeUser(peerName, peerAvatar)} large />
        <div className="call-info">
          <strong>{peerName || '未知用户'}</strong>
          <span>{callTypeLabel(call.call_type)}</span>
        </div>
        <div className="call-actions incoming">
          <button className="call-btn-accept" onClick={onAccept}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z" /></svg>
          </button>
          <button className="call-btn-reject" onClick={onReject}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="1" y1="1" x2="23" y2="23" /><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6A19.79 19.79 0 0 1 2 4.18 2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z" /></svg>
          </button>
        </div>
        <div className="call-hint">接听 / 拒绝</div>
      </div>
    </div>
  );
}
