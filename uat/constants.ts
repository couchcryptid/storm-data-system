// Expected values from mock NOAA SPC dataset (2024-04-26).
// Must match storm-data-system/e2e/e2e_test.go constants.

export const EXPECTED_TOTAL = 271;
export const EXPECTED_HAIL = 79;
export const EXPECTED_TORNADO = 149;
export const EXPECTED_WIND = 43;

export const EXPECTED_STATE_COUNT = 11;

export const EXPECTED_TOP_STATES: Record<string, number> = {
  NE: 100,
  IA: 69,
  TX: 39,
};

export const DATA_READY_TEXT = `${EXPECTED_TOTAL} reports loaded`;
export const DATA_READY_TIMEOUT = 120_000;

export const TOOLBAR_LINKS = [
  { label: 'GraphQL Playground', href: 'http://localhost:8080' },
  { label: 'Prometheus', href: 'http://localhost:9090' },
  { label: 'Kafka UI', href: 'http://localhost:8082' },
];
