import { useEffect } from 'react';
import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import type { MessageDTO, WsResponse } from '@/api/types';
import type { RealtimeClient } from '@/api/ws';
import type { Notice } from '@/types';
import type { ChatAction, ChatState } from '../reducers/chatReducer';

export function useChatLifecycleEffects({
  chatState,
  chatStateRef,
  selectedID,
  selectedIDRef,
  messages,
  messagesRef,
  unreadTotal,
  wsRef,
  realtimeHandlerRef,
  dispatchChat,
  refreshBase,
  loadMessages,
  loadChatMembers,
  notice,
  setNotice,
}: {
  chatState: ChatState;
  chatStateRef: MutableRefObject<ChatState>;
  selectedID: number | null;
  selectedIDRef: MutableRefObject<number | null>;
  messages: Record<number, MessageDTO[]>;
  messagesRef: MutableRefObject<Record<number, MessageDTO[]>>;
  unreadTotal: number;
  wsRef: MutableRefObject<RealtimeClient>;
  realtimeHandlerRef: MutableRefObject<(msg: WsResponse) => void>;
  dispatchChat: Dispatch<ChatAction>;
  refreshBase: () => Promise<void>;
  loadMessages: (cid: number, beforeSeq?: number) => Promise<void>;
  loadChatMembers: (cid: number, force?: boolean) => Promise<void>;
  notice: Notice;
  setNotice: Dispatch<SetStateAction<Notice>>;
}) {
  useEffect(() => { chatStateRef.current = chatState; }, [chatState, chatStateRef]);
  useEffect(() => { selectedIDRef.current = selectedID; }, [selectedID, selectedIDRef]);
  useEffect(() => { messagesRef.current = messages; }, [messages, messagesRef]);

  useEffect(() => {
    if (unreadTotal > 0) document.title = `(${unreadTotal}) QIM`;
    else document.title = 'QIM';
  }, [unreadTotal]);

  useEffect(() => {
    const ws = wsRef.current;
    const off = ws.on((msg) => realtimeHandlerRef.current(msg));
    ws.connect(() => {
      dispatchChat({ type: 'setField', key: 'onlineMap', value: {} });
      void refreshBase();
    });
    void refreshBase();
    return () => { off(); ws.close(); };
  }, [dispatchChat, refreshBase, realtimeHandlerRef, wsRef]);

  useEffect(() => {
    if (!selectedID) return;
    void loadMessages(selectedID);
    void loadChatMembers(selectedID);
  }, [selectedID, loadMessages, loadChatMembers]);

  useEffect(() => {
    if (!selectedID) return;
    const timer = window.setInterval(() => void loadChatMembers(selectedID, true), 5000);
    return () => window.clearInterval(timer);
  }, [selectedID, loadChatMembers]);

  useEffect(() => {
    if ('Notification' in window && Notification.permission === 'default') Notification.requestPermission();
  }, []);

  useEffect(() => {
    if (!notice) return;
    const timer = setTimeout(() => setNotice(null), 3000);
    return () => clearTimeout(timer);
  }, [notice, setNotice]);
}
