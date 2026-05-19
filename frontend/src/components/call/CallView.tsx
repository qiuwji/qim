import { useRef, useEffect } from 'react';
import type { ActiveCall } from '@/hooks/useCallStore';
import { formatDuration } from '@/hooks/chat/models/callModel';

interface CallViewProps {
  call: ActiveCall;
  remoteStream: MediaStream | null;
  localStreamRef: { current: MediaStream | null };
  isMuted: boolean;
  isCameraOff: boolean;
  duration: number;
  onToggleMute: () => void;
  onToggleCamera: () => void;
  onHangup: () => void;
}

export function CallView({ call, remoteStream, localStreamRef, isMuted, isCameraOff, duration, onToggleMute, onToggleCamera, onHangup }: CallViewProps) {
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const remoteAudioRef = useRef<HTMLAudioElement>(null);
  const localVideoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    if (remoteVideoRef.current && remoteStream) {
      remoteVideoRef.current.srcObject = remoteStream;
      remoteVideoRef.current.play().catch(() => {});
    }
    if (remoteAudioRef.current && remoteStream) {
      remoteAudioRef.current.srcObject = remoteStream;
      remoteAudioRef.current.play().catch(() => {});
    }
  }, [remoteStream]);

  useEffect(() => {
    if (localVideoRef.current && localStreamRef.current) {
      localVideoRef.current.srcObject = localStreamRef.current;
    }
  });

  const isVideo = call.call_type === 2;

  return (
    <div className="call-overlay">
      <div className={`call-view ${isVideo ? 'video' : 'voice'}`}>
        {isVideo && (
          <video
            ref={remoteVideoRef}
            className="call-remote-video"
            autoPlay
            playsInline
          />
        )}
        {isVideo && (
          <video
            ref={localVideoRef}
            className="call-local-video"
            autoPlay
            playsInline
            muted
          />
        )}
        {!isVideo && (
          <div className="call-voice-avatars">
            <div className="call-avatar xlarge">
              <span className="avatar-text">?</span>
            </div>
          </div>
        )}
        <audio ref={remoteAudioRef} autoPlay playsInline />
        <div className="call-duration">{formatDuration(duration)}</div>
        <div className="call-controls">
          <button
            className={`call-ctrl-btn ${isMuted ? 'active' : ''}`}
            onClick={onToggleMute}
            title={isMuted ? '取消静音' : '静音'}
          >
            {isMuted ? (
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="1" y1="1" x2="23" y2="23" /><path d="M11 5L6 9H2v6h4l5 4V5z" /><path d="M19.07 4.93a10 10 0 0 1 0 14.14" /><path d="M15.54 8.46a5 5 0 0 1 0 7.07" /></svg>
            ) : (
              <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" /><path d="M19 10v2a7 7 0 0 1-14 0v-2" /><line x1="12" y1="19" x2="12" y2="23" /><line x1="8" y1="23" x2="16" y2="23" /></svg>
            )}
          </button>
          {isVideo && (
            <button
              className={`call-ctrl-btn ${isCameraOff ? 'active' : ''}`}
              onClick={onToggleCamera}
              title={isCameraOff ? '开启摄像头' : '关闭摄像头'}
            >
              {isCameraOff ? (
                <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="1" y1="1" x2="23" y2="23" /><path d="M21 21H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h3l2-3h8l2 3h3a2 2 0 0 1 2 2v9.34" /></svg>
              ) : (
                <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M23 7l-7 5 7 5V7z" /><rect x="1" y="5" width="15" height="14" rx="2" ry="2" /></svg>
              )}
            </button>
          )}
          <button className="call-ctrl-btn hangup" onClick={onHangup} title="挂断">
            <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6A19.79 19.79 0 0 1 2 4.18 2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z" /></svg>
          </button>
        </div>
      </div>
    </div>
  );
}
