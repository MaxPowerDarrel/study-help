type Props = {
  open: boolean;
  onClose: () => void;
  autoLoad: boolean;
  setAutoLoad: (v: boolean) => void;
};

export function SettingsPane({ open, onClose, autoLoad, setAutoLoad }: Props) {
  if (!open) return null;
  return (
    <div className="settings-overlay" role="dialog" aria-label="Settings">
      <div className="settings-pane">
        <header className="settings-header">
          <h2>Settings</h2>
          <button
            type="button"
            className="settings-close"
            onClick={onClose}
            aria-label="Close settings"
          >
            ×
          </button>
        </header>
        <label className="settings-toggle">
          <input
            type="checkbox"
            checked={autoLoad}
            onChange={(e) => setAutoLoad(e.target.checked)}
          />
          Auto-load daily reading
        </label>
      </div>
    </div>
  );
}
