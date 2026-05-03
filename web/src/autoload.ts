import { useEffect, useState } from "react";
import { defaultToggleStore, ToggleStore } from "./platform/ToggleStore";

const STORAGE_KEY = "reader.autoload";

export function useAutoLoad(store: ToggleStore = defaultToggleStore) {
  const [enabled, setEnabled] = useState<boolean>(() => {
    return store.get(STORAGE_KEY) === "true";
  });

  useEffect(() => {
    store.set(STORAGE_KEY, String(enabled));
  }, [enabled, store]);

  return [enabled, setEnabled] as const;
}
