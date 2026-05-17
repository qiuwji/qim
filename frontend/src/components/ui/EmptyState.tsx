export function EmptyState({ title, text }: { title: string; text: string }) {
  return (
    <div className="m-auto grid place-items-center gap-2 p-8 text-center text-[#858c98]">
      <strong className="text-base font-semibold text-[#555f6d]">{title}</strong>
      <span className="text-sm">{text}</span>
    </div>
  );
}
