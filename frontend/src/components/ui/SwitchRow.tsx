export function SwitchRow({ label, on, onClick, ariaLabel }: { label: string; on: boolean; onClick: () => void; ariaLabel?: string }) {
  return (
    <div className="switch-row">
      <span>{label}</span>
      <button type="button" className={`switch-btn ${on ? 'on' : ''}`} onClick={onClick} aria-label={ariaLabel ?? label} />
    </div>
  );
}
