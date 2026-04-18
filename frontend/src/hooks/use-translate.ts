import { useState, useEffect } from "react";
import { Translate as TranslateFn } from "../../wailsjs/go/main/App";

const translationCache: Record<string, string> = {};

export function useTranslate(key: string): string {
  const [translated, setTranslated] = useState<string>(key);

  useEffect(() => {
    // Check cache first
    if (translationCache[key]) {
      setTranslated(translationCache[key]);
      return;
    }

    // Fetch translation
    TranslateFn(key)
      .then((result) => {
        translationCache[key] = result;
        setTranslated(result);
      })
      .catch(() => {
        // Fallback to key if translation fails
        setTranslated(key);
      });
  }, [key]);

  return translated;
}

// Helper function for synchronous use (uses cache)
export function translate(key: string): string {
  return translationCache[key] || key;
}

// Preload translations
export async function preloadTranslations(keys: string[]): Promise<void> {
  const promises = keys.map(async (key) => {
    if (!translationCache[key]) {
      try {
        const result = await TranslateFn(key);
        translationCache[key] = result;
      } catch {
        // Ignore errors
      }
    }
  });
  await Promise.all(promises);
}
