export type ID = number;

export type ApiCode = 'ok' | string;

export interface ApiResponse<T> {
  code: ApiCode;
  data?: T;
  message?: string;
}

export interface UserDTO {
  id: ID;
  username: string;
  nickname: string;
  avatar: string;
  sign: string;
  status: number;
  created_at: number;
  last_online_at: number;
}

export interface LoginData {
  token: string;
  user: UserDTO;
}

export interface UserConvDTO {
  conversation_id: ID;
  is_pinned: boolean;
  is_muted: boolean;
  unread_count: number;
  last_msg_at: number;
  conv?: ConversationDTO;
}

export interface ConversationDTO {
  id: ID;
  type: 1 | 2;
  name: string;
  avatar: string;
  owner_id: ID;
  member_count: number;
  member_limit: number;
  max_seq: number;
  created_at: number;
}

export interface MessageDTO {
  id: ID;
  conversation_id: ID;
  seq: number;
  sender_id: ID;
  msg_type: number;
  content: string;
  reply_to: ID;
  revoked?: boolean;
  edited?: boolean;
  client_id: string;
  created_at: number;
  mention_uids?: number[];
  mention_all?: boolean;
}

export interface FriendDTO {
  id: ID;
  friend_uid: ID;
  remark: string;
  group_id: ID;
  created_at: number;
}

export interface FriendRequestDTO {
  id: ID;
  from_uid: ID;
  to_uid: ID;
  message: string;
  status: number;
  created_at: number;
}

export interface FriendGroupDTO {
  id: ID;
  name: string;
  sort_order: number;
}

export interface MemberDTO {
  uid: ID;
  role: number;
  last_read_seq: number;
  join_time: number;
}

export interface UploadDTO {
  url: string;
  filename: string;
  size: number;
  mime_type: string;
}

export interface FriendGroupSortItem {
  group_id: ID;
  sort_order: number;
}

export interface WsErrorPayload {
  code: string;
  message: string;
}

export interface WsResponse<T = unknown> {
  type: string;
  action?: string;
  data?: T;
  error?: WsErrorPayload;
  log_id?: string;
}
