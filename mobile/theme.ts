export const Colors = {
  // Brand colors
  primary: "#B6FF35", // Lime green - primary accent/CTA
  background: "#F7F8FA", // App background
  surface: "#FFFFFF", // Cards and surfaces

  // Text colors
  primaryText: "#202124", // Primary text
  secondaryText: "#777B83", // Secondary text

  // Borders & dividers
  border: "#ECEEF1",

  // States
  error: "#D32F2F",

  // Map-specific (we'll add these later)
  terminalNeutral: "#F0F0F0", // For terminals with no wait data
  terminalGood: "#4CAF50", // Green for short waits
  terminalWarning: "#FF9800", // Orange for medium waits
  terminalBad: "#F44336", // Red for long waits
};

export const Spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
};

export const Radius = {
  sm: 4,
  md: 8,
  lg: 12,
  pill: 999,
};

export const Typography = {
  // We'll use system fonts but can customize weights/sizes
  h1: { fontSize: 24, fontWeight: "bold" as const },
  h2: { fontSize: 20, fontWeight: "bold" as const },
  body: { fontSize: 16, fontWeight: "400" as const },
  caption: { fontSize: 12, fontWeight: "400" as const },
};
