import { ThemeChoice } from "./theme";
import { Toggles } from "./toggles";
import styles from "./SettingsPane.module.css";

type Props = {
  open: boolean;
  onClose: () => void;
  theme: ThemeChoice;
  setTheme: (c: ThemeChoice) => void;
  toggles: Toggles;
  setToggles: (t: Toggles) => void;
};

const THEME_OPTIONS: { value: ThemeChoice; label: string }[] = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

const SHOW_OPTIONS: { key: keyof Toggles; label: string }[] = [
  { key: "include_headings", label: "Section headings" },
  { key: "include_footnotes", label: "Footnotes" },
  { key: "include_verse_numbers", label: "Verse numbers" },
  { key: "include_passage_references", label: "Passage reference" },
  { key: "include_word_of_christ", label: "Words of Christ" },
];

export function SettingsPane({
  open,
  onClose,
  theme,
  setTheme,
  toggles,
  setToggles,
}: Props) {
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

        <div className={styles.showGroup} aria-label="Show">
          <span className={styles.showLabel}>Show</span>
          {SHOW_OPTIONS.map(({ key, label }) => (
            <label key={key} className={styles.toggle}>
              <input
                type="checkbox"
                checked={toggles[key]}
                onChange={(e) =>
                  setToggles({ ...toggles, [key]: e.target.checked })
                }
              />
              {label}
            </label>
          ))}
        </div>
      </div>
    </div>
  );
}
