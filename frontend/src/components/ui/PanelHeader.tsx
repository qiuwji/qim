import type { ReactNode } from 'react';

export function PanelHeader({ title, subtitle, onBack, right }: {
  title: string;
  subtitle?: string;
  onBack?: () => void;
  right?: ReactNode;
}) {
  return (
    <header className="flex shrink-0 items-center gap-3 border-b border-[#dfe3e8] bg-[#f9fafb] px-4 py-3">
      {onBack && (
        <button className="back-btn" onClick={onBack}>
          <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="15 18 9 12 15 6" /></svg>
        </button>
      )}
      <div className="min-w-0 flex-1">
        <strong className="block truncate text-base font-semibold text-[#1a1a1a]">{title}</strong>
        {subtitle && <div className="truncate text-xs text-[#999]">{subtitle}</div>}
      </div>
      {right}
    </header>
  );
}
