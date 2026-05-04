import { ThemeChoice } from "./theme";
import styles from "./SettingsPane.module.css";

type Props = {
  open: boolean;
  onClose: () => void;
  theme: ThemeChoice;
  setTheme: (c: ThemeChoice) => void;
};

const THEME_OPTIONS: { value: ThemeChoice; label: string }[] = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

export function SettingsPane({ open, onClose, theme, setTheme }: Props) {
  if (!open) return null;
  return (
    <div className={styles.overlay} role="dialog" aria-label="Settings">
      <div className={styles.pane}>
        <header className={styles.header}>
          <h2>Settings</h2>
          <button
            type="button"
            className={styles.close}
            onClick={onClose}
            aria-label="Close settings"
          >
            ×
          </button>
        </header>

        <div className={styles.themeGroup} role="radiogroup" aria-label="Theme">
          <span className={styles.themeLabel}>Theme</span>
          <div className={styles.themeChoices}>
            {THEME_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                type="button"
                role="radio"
                aria-checked={theme === opt.value}
                className={
                  theme === opt.value
                    ? `${styles.themeChoice} ${styles.themeChoiceActive}`
                    : styles.themeChoice
                }
                onClick={() => setTheme(opt.value)}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
