const EMOJI_LIST = [
  '😀','😁','😂','🤣','😃','😄','😅','😆','😉','😊',
  '😋','😎','😍','🥰','😘','😗','😙','😚','🙂','🤗',
  '🤔','😐','😑','😶','🙄','😏','😣','😥','😮','🤐',
  '😯','😪','😫','😴','😌','😛','😜','😝','🤤','😒',
  '😓','😔','😕','🙃','🤑','😲','🙁','😖','😞','😟',
  '😤','😢','😭','😦','😧','😨','😩','🤯','😬','😰',
  '😱','🥵','🥶','😳','🤪','😵','😡','😠','🤬','😈',
  '👍','👎','👏','🙌','🤝','💪','✌️','🤞','🤟','🤘',
  '❤️','🧡','💛','💚','💙','💜','🖤','💔','💕','💖',
  '🔥','⭐','🎉','🎊','💯','✅','❌','⚡','💡','🎵',
];

export function EmojiPicker({ onSelect, onClose }: { onSelect: (emoji: string) => void; onClose: () => void }) {
  return (
    <div className="emoji-overlay" onClick={onClose}>
      <div className="emoji-picker" onClick={(e) => e.stopPropagation()}>
        {EMOJI_LIST.map((emoji) => (
          <button key={emoji} type="button" className="emoji-item" onClick={() => onSelect(emoji)}>{emoji}</button>
        ))}
      </div>
    </div>
  );
}
