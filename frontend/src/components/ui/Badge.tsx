export function Badge({ count, muted = false }: { count: number; muted?: boolean }) {
  if (count <= 0) return null;
  return <b className={`grid h-5 min-w-5 place-items-center rounded-full px-1.5 text-[11px] font-semibold text-white ${muted ? 'bg-[#c9ced6]' : 'bg-[#f04b45]'}`}>{count > 99 ? '99+' : count}</b>;
}
