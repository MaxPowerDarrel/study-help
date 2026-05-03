// TimezoneProvider is the platform abstraction for resolving the
// client's IANA timezone (PROJECT_CONSTITUTION.md §4 — platform features
// behind an abstraction). A native shell can substitute its own source.
export interface TimezoneProvider {
  get(): string;
}

class IntlTimezoneProvider implements TimezoneProvider {
  get(): string {
    try {
      const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
      return tz || "UTC";
    } catch {
      return "UTC";
    }
  }
}

export const defaultTimezoneProvider: TimezoneProvider =
  new IntlTimezoneProvider();
