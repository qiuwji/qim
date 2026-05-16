import type { UserDTO } from '../api/types';
import { Avatar } from './Avatar';

export function UserProfilePage({ user, isFriend, isSelf, onBack, onStartPrivate, onAddFriend }: {
  user: UserDTO; isFriend?: boolean; isSelf?: boolean; onBack: () => void; onStartPrivate: (uid: number) => void; onAddFriend?: (uid: number) => void;
}) {
  return (
    <div className="user-profile-page">
      <header className="chat-header">
        <button className="back-btn" onClick={onBack}>返回</button>
        <div className="chat-title"><strong>个人信息</strong></div>
      </header>
      <div className="user-profile-content">
        <div className="user-profile-hero">
          <Avatar user={user} large />
          <strong>{user.nickname || user.username}</strong>
          <span>@{user.username}</span>
          <span className="user-card-id">ID: {user.id}</span>
        </div>
        <div className="user-profile-fields">
          <div className="profile-field">
            <label>昵称</label>
            <span>{user.nickname || '-'}</span>
          </div>
          <div className="profile-field">
            <label>用户名</label>
            <span>{user.username}</span>
          </div>
          <div className="profile-field">
            <label>UID</label>
            <span>{user.id}</span>
          </div>
          <div className="profile-field">
            <label>个性签名</label>
            <span>{user.sign || '这个人还没有写个性签名'}</span>
          </div>
          <div className="profile-field">
            <label>注册时间</label>
            <span>{user.created_at ? new Date(user.created_at * 1000).toLocaleString('zh-CN') : '-'}</span>
          </div>
        </div>
        <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
          {!isSelf && isFriend && <button className="primary-btn" onClick={() => onStartPrivate(user.id)}>聊天记录</button>}
          {!isSelf && !isFriend && onAddFriend && <button className="primary-btn" onClick={() => onAddFriend(user.id)}>加好友</button>}
        </div>
      </div>
    </div>
  );
}
