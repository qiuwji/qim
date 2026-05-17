export function SearchInput({ value, onChange, placeholder, onSearch }: { value: string; onChange: (v: string) => void; placeholder?: string; onSearch?: () => void }) {
  return (
    <div className="grid grid-cols-[1fr_auto] gap-2">
      <input className="min-w-0 border border-[#d8dde4] rounded-lg px-2.5 py-2 outline-none focus:border-[#12b35f]" value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} onKeyDown={(e) => e.key === 'Enter' && onSearch?.()} />
      {onSearch && <button type="button" className="px-3 py-2 rounded-lg text-white bg-[#12b35f] font-semibold" onClick={onSearch}>搜索</button>}
    </div>
  );
}
