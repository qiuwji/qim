import { useRef, useState, useCallback, useEffect } from 'react';
import type { CallEndedData, CallIncomingData, WsResponse } from '@/api/types';
import type { RealtimeClient } from '@/api/ws';
import { formatDuration, callTypeLabel } from './chat/models/callModel';

export type CallState = 'idle' | 'calling' | 'incoming' | 'connected' | 'ended';

export interface ActiveCall {
  call_id: string;
  peer_uid: number;
  call_type: 1 | 2;
  status: CallState;
  is_caller: boolean;
  started_at?: number;
  peer_nickname?: string;
  peer_avatar?: string;
}

const ICE_SERVERS: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
  ],
};

export function useCallStore(wsRef: { current: RealtimeClient }) {
  const [callState, setCallState] = useState<CallState>('idle');
  const [activeCall, setActiveCall] = useState<ActiveCall | null>(null);
  const activeCallRef = useRef<ActiveCall | null>(null);
  const [callDuration, setCallDuration] = useState(0);
  const [isMuted, setIsMuted] = useState(false);
  const [isCameraOff, setIsCameraOff] = useState(false);
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);

  const peerConnectionRef = useRef<RTCPeerConnection | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);
  const durationTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const offerTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const disconnectedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => { activeCallRef.current = activeCall; }, [activeCall]);

  const cleanupCall = useCallback(() => {
    if (durationTimerRef.current) { clearInterval(durationTimerRef.current); durationTimerRef.current = null; }
    if (offerTimerRef.current) { clearTimeout(offerTimerRef.current); offerTimerRef.current = null; }
    if (disconnectedTimerRef.current) { clearTimeout(disconnectedTimerRef.current); disconnectedTimerRef.current = null; }
    if (peerConnectionRef.current) { peerConnectionRef.current.close(); peerConnectionRef.current = null; }
    if (localStreamRef.current) { localStreamRef.current.getTracks().forEach(t => t.stop()); localStreamRef.current = null; }
    setRemoteStream(null);
    setCallDuration(0);
    setIsMuted(false);
    setIsCameraOff(false);
  }, []);

  const resetCall = useCallback(() => {
    cleanupCall();
    setCallState('idle');
    setActiveCall(null);
    activeCallRef.current = null;
  }, [cleanupCall]);

  const noticeRef = useRef<(text: string) => void>(() => {});
  const endCallRef = useRef<() => void>(() => {});

  const setNotice = useCallback((text: string) => {
    noticeRef.current(text);
  }, []);

  const createPeerConnection = useCallback(() => {
    const pc = new RTCPeerConnection(ICE_SERVERS);
    peerConnectionRef.current = pc;

    pc.onicecandidate = (event) => {
      const call = activeCallRef.current;
      if (event.candidate && call) {
        wsRef.current.sendIceCandidate(
          call.call_id,
          event.candidate.candidate,
          event.candidate.sdpMid ?? undefined,
          event.candidate.sdpMLineIndex ?? undefined,
        );
      }
    };

    pc.ontrack = (event) => {
      if (event.streams[0]) {
        setRemoteStream(event.streams[0]);
      }
    };

    pc.oniceconnectionstatechange = () => {
      if (pc.iceConnectionState === 'failed') {
        setNotice('网络不支持通话');
        endCallRef.current();
      } else if (pc.iceConnectionState === 'disconnected') {
        disconnectedTimerRef.current = setTimeout(() => {
          endCallRef.current();
        }, 5000);
      } else if (pc.iceConnectionState === 'connected' || pc.iceConnectionState === 'completed') {
        if (disconnectedTimerRef.current) { clearTimeout(disconnectedTimerRef.current); disconnectedTimerRef.current = null; }
      }
    };

    return pc;
  }, [wsRef, setNotice]);

  const initiateCall = useCallback(async (peerUID: number, callType: 1 | 2) => {
    if (typeof RTCPeerConnection === 'undefined') {
      setNotice('浏览器不支持通话');
      return;
    }
    setCallState('calling');
    const call: ActiveCall = {
      call_id: '',
      peer_uid: peerUID,
      call_type: callType,
      status: 'calling',
      is_caller: true,
    };
    setActiveCall(call);
    activeCallRef.current = call;
    wsRef.current.initiateCall(peerUID, callType);
  }, [wsRef, setNotice]);

  const acceptCall = useCallback(async () => {
    const call = activeCallRef.current;
    if (!call) return;
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: true,
        video: call.call_type === 2,
      });
      localStreamRef.current = stream;
    } catch {
      setNotice('无法访问麦克风/摄像头');
      wsRef.current.rejectCall(call.call_id);
      resetCall();
      return;
    }

    const pc = createPeerConnection();
    localStreamRef.current?.getTracks().forEach(track => {
      pc.addTrack(track, localStreamRef.current!);
    });

    setCallState('connected');
    setActiveCall(prev => prev ? { ...prev, status: 'connected', started_at: Date.now() / 1000 } : null);
    durationTimerRef.current = setInterval(() => {
      setCallDuration(d => d + 1);
    }, 1000);

    wsRef.current.acceptCall(call.call_id);
  }, [createPeerConnection, wsRef, resetCall, setNotice]);

  const rejectCall = useCallback(() => {
    const call = activeCallRef.current;
    if (!call) return;
    wsRef.current.rejectCall(call.call_id);
    resetCall();
  }, [wsRef, resetCall]);

  const cancelCall = useCallback(() => {
    const call = activeCallRef.current;
    if (!call) return;
    wsRef.current.cancelCall(call.call_id);
    resetCall();
  }, [wsRef, resetCall]);

  const endCall = useCallback(() => {
    const call = activeCallRef.current;
    if (!call) return;
    if (call.call_id) wsRef.current.endCall(call.call_id);
    resetCall();
  }, [wsRef, resetCall]);

  endCallRef.current = endCall;

  const toggleMute = useCallback(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getAudioTracks().forEach(t => { t.enabled = !t.enabled; });
      setIsMuted(m => !m);
    }
  }, []);

  const toggleCamera = useCallback(() => {
    if (localStreamRef.current) {
      localStreamRef.current.getVideoTracks().forEach(t => { t.enabled = !t.enabled; });
      setIsCameraOff(c => !c);
    }
  }, []);

  const handleCallSignal = useCallback(async (action: string, data: Record<string, unknown>) => {
    const pc = peerConnectionRef.current;
    if (!pc) return;

    try {
      if (action === 'offer') {
        if (offerTimerRef.current) { clearTimeout(offerTimerRef.current); }
        await pc.setRemoteDescription({ type: 'offer', sdp: data.sdp as string });
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        const call = activeCallRef.current;
        if (call) {
          wsRef.current.sendCallAnswer(call.call_id, answer.sdp!);
        }
      } else if (action === 'answer') {
        if (offerTimerRef.current) { clearTimeout(offerTimerRef.current); }
        await pc.setRemoteDescription({ type: 'answer', sdp: data.sdp as string });
      } else if (action === 'ice') {
        await pc.addIceCandidate({
          candidate: data.candidate as string,
          sdpMid: data.sdp_mid as string | undefined,
          sdpMLineIndex: data.sdp_m_line_index as number | undefined,
        });
      }
    } catch {
      // WebRTC negotiation error
    }
  }, [wsRef]);

  function handleCallWs(msg: WsResponse) {
    const data = msg.data as Record<string, unknown> | undefined;

    if (msg.type === 'system' && msg.action === 'closed') {
      if (callState !== 'idle') {
        setNotice('连接已断开');
        endCallRef.current();
      }
      return;
    }

    if (msg.type === 'call') {
      switch (msg.action) {
        case 'incoming': {
          const incoming = data as unknown as CallIncomingData;
          const call: ActiveCall = {
            call_id: incoming.call_id,
            peer_uid: incoming.caller_uid,
            call_type: incoming.call_type,
            status: 'incoming',
            is_caller: false,
            peer_nickname: incoming.caller_nickname,
            peer_avatar: incoming.caller_avatar,
          };
          setCallState('incoming');
          setActiveCall(call);
          activeCallRef.current = call;
          if (!document.hasFocus()) {
            try { new Notification('QIM', { body: `${incoming.caller_nickname} - ${callTypeLabel(incoming.call_type)}` }); } catch { /* */ }
          }
          try {
            const audio = new Audio('/sounds/call-ring.mp3');
            audio.loop = true;
            audio.play().catch(() => {});
            (window as unknown as Record<string, unknown>).__callRing = audio;
          } catch { /* */ }
          break;
        }
        case 'calling': {
          const call = activeCallRef.current;
          if (call) {
            setActiveCall(prev => prev ? { ...prev, call_id: (data?.call_id as string) || prev.call_id } : null);
          }
          break;
        }
        case 'accepted': {
          const call = activeCallRef.current;
          if (!call) break;
          setCallState('connected');
          setActiveCall(prev => prev ? { ...prev, status: 'connected', started_at: Date.now() / 1000 } : null);
          durationTimerRef.current = setInterval(() => setCallDuration(d => d + 1), 1000);

          offerTimerRef.current = setTimeout(() => {
            endCall();
          }, 10000);

          (async () => {
            try {
              const stream = await navigator.mediaDevices.getUserMedia({
                audio: true,
                video: call.call_type === 2,
              });
              localStreamRef.current = stream;
              const pc = createPeerConnection();
              stream.getTracks().forEach(track => pc.addTrack(track, stream));
              const offer = await pc.createOffer();
              await pc.setLocalDescription(offer);
              wsRef.current.sendCallOffer(call.call_id, offer.sdp!);
            } catch {
              setNotice('无法访问麦克风/摄像头');
              endCall();
            }
          })();
          break;
        }
        case 'rejected':
          setNotice('对方已拒绝');
          resetCall();
          break;
        case 'cancelled':
          setNotice('对方已取消');
          resetCall();
          break;
        case 'ended': {
          const ended = data as unknown as CallEndedData;
          setNotice(`通话结束 ${formatDuration(ended.duration)}`);
          resetCall();
          break;
        }
        case 'timeout':
          setNotice('无人接听');
          resetCall();
          break;
        case 'answered_elsewhere':
          if (callState === 'connected') break;
          setNotice('已在其他设备接听');
          resetCall();
          break;
        case 'offer':
        case 'answer':
        case 'ice':
          handleCallSignal(msg.action, data ?? {});
          break;
      }
    } else if (msg.type === 'error') {
      const errMsg = msg.error?.message || '通话失败';
      setNotice(errMsg);
      if (msg.action === 'initiate') resetCall();
    }

    stopRing();
  }

  function stopRing() {
    try {
      const audio = (window as unknown as Record<string, unknown>).__callRing as HTMLAudioElement | undefined;
      if (audio) { audio.pause(); audio.currentTime = 0; delete (window as unknown as Record<string, unknown>).__callRing; }
    } catch { /* */ }
  }

  return {
    callState, activeCall, callDuration, isMuted, isCameraOff, remoteStream,
    initiateCall, acceptCall, rejectCall, cancelCall, endCall,
    toggleMute, toggleCamera,
    handleCallWs,
    setNoticeRef: (fn: (text: string) => void) => { noticeRef.current = fn; },
    localStreamRef,
  };
}
