import { Image } from "expo-image";
import { Ionicons } from "@expo/vector-icons";
import { View } from "react-native";
import {
  makerThumbnails,
  thumbnailVersion,
} from "@/lib/maker-thumbnails.generated";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";
export function PartThumbnail({
  version,
  ldraw,
  color,
}: {
  version: string;
  ldraw: string;
  color: number;
}) {
  const source =
    version === thumbnailVersion
      ? makerThumbnails[`${ldraw.toLowerCase()}:${color}`]
      : undefined;
  const { colorScheme } = useColorScheme();
  return (
    <View
      className="rounded-lg bg-secondary items-center justify-center"
      style={{ width: 76, height: 58 }}
    >
      {source ? (
        <Image
          source={source}
          contentFit="contain"
          style={{ width: 76, height: 58 }}
        />
      ) : (
        <Ionicons
          name="cube-outline"
          size={26}
          color={THEME[colorScheme].mutedForeground}
        />
      )}
    </View>
  );
}
