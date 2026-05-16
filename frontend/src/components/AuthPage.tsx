import { useState } from 'react';
import type { UserDTO } from '../api/types';
import type { Notice } from '../types';
import { api, saveSession } from '../api/http';

export function AuthPage({ onLoggedIn, notice, setNotice }: {
  onLoggedIn: (token: string, user: UserDTO) => void; notice: Notice; setNotice: (n: Notice) => void;
}) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [nickname, setNickname] = useState('');
  const [loading, setLoading] = useState(false);

  async function submit(e: { preventDefault: () => void }) {
    e.preventDefault(); setLoading(true); setNotice(null);
    try {
      if (mode === 'register') await api.register({ username, password, nickname: nickname || username });
      const data = await api.login({ username, password });
      onLoggedIn(data.token, data.user);
    } catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '登录失败' }); }
    finally { setLoading(false); }
  }

  return (
    <main className="auth-screen">
      <section className="auth-card">
        <div className="brand-mark">Q</div>
        <h1>QIM</h1>
        <p>像 QQ / 微信一样轻量的即时通讯 Demo</p>
        {notice && <div className={`notice ${notice.kind}`}>{notice.text}</div>}
        <form onSubmit={submit} className="auth-form">
          <label>账号<input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="alice" required /></label>
          {mode === 'register' && <label>昵称<input value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="Alice" /></label>}
          <label>密码<input value={password} onChange={(e) => setPassword(e.target.value)} type="password" placeholder="至少输入一个密码" required /></label>
          <button className="primary-btn" disabled={loading}>{loading ? '处理中...' : mode === 'login' ? '登录' : '注册并登录'}</button>
        </form>
        <button className="link-btn" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>{mode === 'login' ? '没有账号？去注册' : '已有账号？去登录'}</button>
      </section>
    </main>
  );
}
