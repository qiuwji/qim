import type { UserDTO } from '../api/types';
import { Avatar } from './Avatar';

export function ProfilePanel({ user, onUploadAvatar, onUpdateProfile, onChangePassword }: {
  user: UserDTO; onUploadAvatar: (f: File) => void; onUpdateProfile: () => void; onChangePassword: () => void;
}) {
  return (
    <div className="profile-panel">
      <Avatar user={user} large />
      <strong>{user.nickname || user.username}</strong>
      <span>@{user.username}</span>
      <p>{user.sign || '这个人还没有写个性签名'}</p>
      <label className="upload-btn">上传头像<input type="file" accept="image/*" onChange={(e) => e.target.files?.[0] && onUploadAvatar(e.target.files[0])} /></label>
      <button className="wide-btn" onClick={onUpdateProfile}>修改昵称/签名</button>
      <button className="wide-btn" onClick={onChangePassword}>修改密码</button>
    </div>
  );
}
