import type {
  ApiResponse,
  ConversationDTO,
  FriendDTO,
  FriendGroupDTO,
  FriendGroupSortItem,
  FriendRequestDTO,
  LoginData,
  MemberDTO,
  MessageDTO,
  UploadDTO,
  UserConvDTO,
  UserDTO
} from './types';

const TOKEN_KEY = 'qim_token';
const USER_KEY = 'qim_user';
const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/$/, '') ?? '';

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly logID?: string | null
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? '';
}

export function saveSession(token: string, user: UserDTO) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export function getSavedUser(): UserDTO | null {
  const raw = localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as UserDTO;
  } catch {
    clearSession();
    return null;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers = new Headers(init.headers);
  if (!headers.has('Content-Type') && init.body && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const resp = await fetch(`${API_BASE}${path}`, { ...init, headers });
  const logID = resp.headers.get('X-Log-ID');
  const json = (await resp.json().catch(() => ({ code: 'internal_error', message: '响应不是 JSON' }))) as ApiResponse<T>;
  if (json.code !== 'ok') {
    throw new ApiError(json.code, json.message || '请求失败', logID);
  }
  return json.data as T;
}

export const api = {
  register: (payload: { username: string; password: string; nickname: string }) =>
    request<UserDTO>('/api/auth/register', { method: 'POST', body: JSON.stringify(payload) }),
  login: (payload: { username: string; password: string }) =>
    request<LoginData>('/api/auth/login', { method: 'POST', body: JSON.stringify(payload) }),
  profile: () => request<UserDTO>('/api/user/profile'),
  updateProfile: (payload: Partial<Pick<UserDTO, 'nickname' | 'avatar' | 'sign'>>) =>
    request<boolean>('/api/user/profile', { method: 'PUT', body: JSON.stringify(payload) }),
  changePassword: (payload: { old_password: string; new_password: string }) =>
    request<boolean>('/api/user/password', { method: 'PUT', body: JSON.stringify(payload) }),
  searchUsers: (keyword: string) => request<UserDTO[]>(`/api/users/search?keyword=${encodeURIComponent(keyword)}`),
  getUser: (uid: number) => request<UserDTO>(`/api/users/${uid}`),
  chats: () => request<UserConvDTO[]>('/api/conversations'),
  conversations: () => request<UserConvDTO[]>('/api/conversations'),
  createPrivateChat: (uid: number) =>
    request<ConversationDTO>('/api/conversations/private', { method: 'POST', body: JSON.stringify({ uid }) }),
  createPrivateConversation: (uid: number) =>
    request<ConversationDTO>('/api/conversations/private', { method: 'POST', body: JSON.stringify({ uid }) }),
  createGroupChat: (payload: { name: string; avatar?: string; members: number[] }) =>
    request<ConversationDTO>('/api/conversations/group', { method: 'POST', body: JSON.stringify(payload) }),
  createGroupConversation: (payload: { name: string; avatar?: string; members: number[] }) =>
    request<ConversationDTO>('/api/conversations/group', { method: 'POST', body: JSON.stringify(payload) }),
  pinChat: (id: number, pinned: boolean) =>
    request<boolean>(`/api/conversations/${id}/pin`, { method: 'PUT', body: JSON.stringify({ pinned }) }),
  pinConversation: (id: number, pinned: boolean) =>
    request<boolean>(`/api/conversations/${id}/pin`, { method: 'PUT', body: JSON.stringify({ pinned }) }),
  muteChat: (id: number, muted: boolean) =>
    request<boolean>(`/api/conversations/${id}/mute`, { method: 'PUT', body: JSON.stringify({ muted }) }),
  muteConversation: (id: number, muted: boolean) =>
    request<boolean>(`/api/conversations/${id}/mute`, { method: 'PUT', body: JSON.stringify({ muted }) }),
  markRead: (id: number, seq: number) =>
    request<boolean>(`/api/conversations/${id}/read`, { method: 'PUT', body: JSON.stringify({ seq }) }),
  markAllRead: () => request<boolean>('/api/conversations/read-all', { method: 'PUT' }),
  updateGroupInfo: (id: number, payload: { name?: string; avatar?: string; member_limit?: number }) =>
    request<boolean>(`/api/conversations/${id}/info`, { method: 'PUT', body: JSON.stringify(payload) }),
  members: (conversationID: number) => request<MemberDTO[]>(`/api/conversations/${conversationID}/members`),
  addMember: (conversationID: number, payload: { uid: number; role: number }) =>
    request<boolean>(`/api/conversations/${conversationID}/members`, { method: 'POST', body: JSON.stringify(payload) }),
  removeMember: (conversationID: number, uid: number) =>
    request<boolean>(`/api/conversations/${conversationID}/members/${uid}`, { method: 'DELETE' }),
  leaveGroup: (conversationID: number) => request<boolean>(`/api/conversations/${conversationID}/leave`, { method: 'DELETE' }),
  setMemberRole: (conversationID: number, uid: number, role: number) =>
    request<boolean>(`/api/conversations/${conversationID}/members/${uid}/role`, { method: 'PUT', body: JSON.stringify({ role }) }),
  transferOwner: (conversationID: number, new_owner_id: number) =>
    request<boolean>(`/api/conversations/${conversationID}/owner`, { method: 'PUT', body: JSON.stringify({ new_owner_id }) }),
  dissolveGroup: (conversationID: number) => request<boolean>(`/api/conversations/${conversationID}/dissolve`, { method: 'DELETE' }),
  messages: (conversationID: number, beforeSeq = 0, limit = 30) =>
    request<MessageDTO[]>(`/api/conversations/${conversationID}/messages?before_seq=${beforeSeq}&limit=${limit}`),
  searchMessages: (conversationID: number, keyword: string, limit = 20) =>
    request<MessageDTO[]>(`/api/messages/search?conversation_id=${conversationID}&keyword=${encodeURIComponent(keyword)}&limit=${limit}`),
  friends: () => request<FriendDTO[]>('/api/friends'),
  incomingRequests: () => request<FriendRequestDTO[]>('/api/friends/requests/incoming'),
  outgoingRequests: () => request<FriendRequestDTO[]>('/api/friends/requests/outgoing'),
  sendFriendRequest: (to_uid: number, message: string) =>
    request<FriendRequestDTO>('/api/friends/request', { method: 'POST', body: JSON.stringify({ to_uid, message }) }),
  handleFriendRequest: (reqID: number, action: 'accept' | 'reject') =>
    request<boolean>(`/api/friends/requests/${reqID}`, { method: 'PUT', body: JSON.stringify({ action }) }),
  deleteFriend: (friendUID: number) => request<boolean>(`/api/friends/${friendUID}`, { method: 'DELETE' }),
  updateFriendRemark: (friendUID: number, remark: string) =>
    request<boolean>(`/api/friends/${friendUID}/remark`, { method: 'PUT', body: JSON.stringify({ remark }) }),
  moveFriendGroup: (friendUID: number, group_id: number) =>
    request<boolean>(`/api/friends/${friendUID}/group`, { method: 'PUT', body: JSON.stringify({ group_id }) }),
  friendGroups: () => request<FriendGroupDTO[]>('/api/friend/groups'),
  createFriendGroup: (name: string) => request<FriendGroupDTO>('/api/friend/groups', { method: 'POST', body: JSON.stringify({ name }) }),
  renameFriendGroup: (groupID: number, name: string) =>
    request<boolean>(`/api/friend/groups/${groupID}`, { method: 'PUT', body: JSON.stringify({ name }) }),
  deleteFriendGroup: (groupID: number) => request<boolean>(`/api/friend/groups/${groupID}`, { method: 'DELETE' }),
  sortFriendGroups: (groups: FriendGroupSortItem[]) =>
    request<boolean>('/api/friend/groups/sort', { method: 'PUT', body: JSON.stringify({ groups }) }),
  uploadImage: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return request<UploadDTO>('/api/files/upload', { method: 'POST', body: form });
  }
};
