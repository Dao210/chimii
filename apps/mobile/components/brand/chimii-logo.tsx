/**
 * Chimii mascot mark. Shares artwork with the web favicon and the app icons;
 * the cream tile keeps it legible on both light and dark surfaces.
 */
import { Image } from "react-native";

interface ChimiiLogoProps {
  size?: number;
}

export function ChimiiLogo({ size = 48 }: ChimiiLogoProps) {
  return (
    <Image
      source={require("../../assets/chimii-brand.png")}
      style={{ width: size, height: size }}
      resizeMode="contain"
      accessible={false}
      accessibilityIgnoresInvertColors
    />
  );
}
