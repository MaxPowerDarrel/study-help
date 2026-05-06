// Per-translation attribution slot. YouVersion's Platform ToS expects
// a visible "Powered by YouVersion" mark whenever NIV passages are
// rendered (final wording / link is locked during pre-merge ToS review).
// The component renders null for translations that don't require an
// attribution; it sits in the layout regardless so spacing is stable.

import type { TranslationID } from "./catalog";
import styles from "./Attribution.module.css";

type Props = { translation: TranslationID };

export function Attribution({ translation }: Props) {
  if (translation === "NIV") {
    return (
      <span className={styles.attribution}>
        Powered by{" "}
        <a
          href="https://www.youversion.com/"
          target="_blank"
          rel="noreferrer noopener"
        >
          YouVersion
        </a>
      </span>
    );
  }
  return null;
}
