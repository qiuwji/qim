import type { ReactNode } from 'react';

export function EmptyState({ title, text, icon }: { title: string; text: string; icon?: ReactNode }) {
  return (
    <div className="m-auto grid place-items-center gap-2 p-8 text-center text-[#858c98]">
      {icon}
      <strong className="text-base font-semibold text-[#555f6d]">{title}</strong>
      <span className="text-sm">{text}</span>
    </div>
  );
}
