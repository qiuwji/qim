import type { FriendDTO, MessageDTO, UserDTO } from '@/api/types';

export type Notice = { kind: 'ok' | 'error' | 'info'; text: string } | null;

export type MainTab = 'chats' | 'contacts' | 'profile';

export type MobilePane = 'list' | 'chat' | 'contacts';

export type ContextMenu = { x: number; y: number; message: MessageDTO } | null;

export type ModalState = null | {
  type: 'prompt';
  title: string;
  fields: { key: string; label: string; placeholder?: string; defaultValue?: string }[];
  onConfirm: (values: Record<string, string>) => void;
} | {
  type: 'confirm';
  title: string;
  text: string;
  danger?: boolean;
  onConfirm: () => void;
} | {
  type: 'friend-picker';
  title: string;
  friends: FriendDTO[];
  userCache: Record<number, UserDTO>;
  excludeUIDs?: number[];
  requireGroupName?: boolean;
  onConfirm: (values: { name?: string; usernames: string[] }) => void;
};
