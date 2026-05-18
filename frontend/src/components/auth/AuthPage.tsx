import { useState } from 'react';
import type { UserDTO } from '@/api/types';
import type { Notice } from '@/types';
import { api } from '@/api/http';
import { validatePassword } from '@/utils';

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
      if (!/^\d+$/.test(username.trim())) {
        setNotice({ kind: 'error', text: '账号只能使用纯数字' });
        return;
      }
      if (mode === 'register') {
        const pwdErr = validatePassword(password);
        if (pwdErr) { setNotice({ kind: 'error', text: pwdErr }); return; }
        await api.register({ username, password, nickname: nickname || username });
      }
      const data = await api.login({ username, password });
      onLoggedIn(data.token, data.user);
    } catch (err) { setNotice({ kind: 'error', text: err instanceof Error ? err.message : '登录失败' }); }
    finally { setLoading(false); }
  }

  return (
    <main className="grid min-h-full place-items-center bg-[#e9edf2] p-6">
      <section className="w-[min(400px,100%)] rounded-[18px] border border-[#dfe3e8] bg-white p-9 text-center shadow-[0_12px_36px_rgba(31,35,41,0.08)] max-[760px]:rounded-2xl max-[760px]:px-5 max-[760px]:py-7">
        <div className="mx-auto mb-4 grid h-[68px] w-[68px] place-items-center rounded-[18px] bg-[#12b35f] text-[32px] font-extrabold text-white">Q</div>
        <h1 className="m-0 text-3xl tracking-[0.5px]">QIM</h1>
        <p className="mt-2 mb-[26px] text-[#7b8491]">像 QQ / 微信一样轻量的即时通讯 Demo</p>
        {notice && <div className={`mb-3 rounded-lg px-3 py-2.5 text-sm ${notice.kind === 'ok' ? 'bg-[#e8f7ef] text-[#0f7a43]' : notice.kind === 'error' ? 'bg-[#fff1f0] text-[#a61d24]' : 'bg-[#edf6ff] text-[#1d5f99]'}`}>{notice.text}</div>}
        <form onSubmit={submit} className="grid gap-3.5 text-left">
          <label className="grid gap-2 text-sm text-[#555f6d]">账号<input className="w-full rounded-lg border border-[#d8dde4] bg-white px-3 py-2.5 outline-none focus:border-[#12b35f] focus:shadow-[0_0_0_3px_rgba(18,179,95,0.1)]" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="请输入纯数字账号" inputMode="numeric" pattern="[0-9]*" required /></label>
          {mode === 'register' && <label className="grid gap-2 text-sm text-[#555f6d]">昵称<input className="w-full rounded-lg border border-[#d8dde4] bg-white px-3 py-2.5 outline-none focus:border-[#12b35f] focus:shadow-[0_0_0_3px_rgba(18,179,95,0.1)]" value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="Alice" /></label>}
          <label className="grid gap-2 text-sm text-[#555f6d]">密码<input className="w-full rounded-lg border border-[#d8dde4] bg-white px-3 py-2.5 outline-none focus:border-[#12b35f] focus:shadow-[0_0_0_3px_rgba(18,179,95,0.1)]" value={password} onChange={(e) => setPassword(e.target.value)} type="password" placeholder="至少输入一个密码" required /></label>
          <button className="rounded-lg bg-[#12b35f] px-[18px] py-2.5 font-semibold text-white hover:bg-[#0ea254]" disabled={loading}>{loading ? '处理中...' : mode === 'login' ? '登录' : '注册并登录'}</button>
        </form>
        <button className="mt-[18px] bg-transparent text-[#1677c7]" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>{mode === 'login' ? '没有账号？去注册' : '已有账号？去登录'}</button>
      </section>
    </main>
  );
}
