export function SearchBar({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div className="flex items-center gap-2 rounded-lg bg-white px-3 py-2">
      <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="#b0b5be" strokeWidth="2" strokeLinecap="round"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
      <input
        className="min-w-0 flex-1 bg-transparent text-[13px] text-[#1a1a1a] outline-none placeholder:text-[#b0b5be]"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
      />
      {value && (
        <button className="grid h-5 w-5 place-items-center rounded-full text-[#b0b5be] hover:bg-[#f0f0f0]" onClick={() => onChange('')}>
          <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
        </button>
      )}
    </div>
  );
}
