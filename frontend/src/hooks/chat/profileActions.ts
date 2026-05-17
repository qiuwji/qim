import { api, getToken, saveSession } from '../../api/http';
import { validatePassword } from '../../utils';
import type { ChatStoreDeps } from './types';

export function createProfileActions(d: ChatStoreDeps) {
  async function uploadAvatar(file: File) {
    try {
      const u = await api.uploadImage(file);
      await api.updateProfile({ avatar: u.url });
      const next = { ...d.user, avatar: u.url };
      saveSession(getToken(), next);
      d.onUserChange(next);
      d.setUserCache((p) => ({ ...p, [d.user.id]: next }));
      d.setNotice({ kind: 'ok', text: '头像已更新' });
    } catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '上传失败' }); }
  }

  function updateProfile() {
    d.setModal({ type: 'prompt', title: '修改资料', fields: [{ key: 'nickname', label: '昵称', defaultValue: d.user.nickname }, { key: 'sign', label: '个性签名', defaultValue: d.user.sign }], onConfirm: async (v) => {
      try {
        await api.updateProfile({ nickname: v.nickname, sign: v.sign });
        const next = { ...d.user, nickname: v.nickname ?? d.user.nickname, sign: v.sign ?? d.user.sign };
        saveSession(getToken(), next);
        d.onUserChange(next);
        d.setUserCache((p) => ({ ...p, [d.user.id]: next }));
      } catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改资料失败' }); }
    } });
  }

  function changePassword() {
    d.setModal({ type: 'prompt', title: '修改密码', fields: [{ key: 'old', label: '旧密码', placeholder: '输入旧密码' }, { key: 'new', label: '新密码', placeholder: '8-20位，含大小写字母和特殊字符' }], onConfirm: async (v) => {
      const pwdErr = validatePassword(v.new ?? '');
      if (pwdErr) { d.setNotice({ kind: 'error', text: pwdErr }); return; }
      try { await api.changePassword({ old_password: v.old ?? '', new_password: v.new ?? '' }); d.setNotice({ kind: 'ok', text: '密码已修改' }); }
      catch (err) { d.setNotice({ kind: 'error', text: err instanceof Error ? err.message : '修改密码失败' }); }
    } });
  }

  return { uploadAvatar, updateProfile, changePassword };
}
