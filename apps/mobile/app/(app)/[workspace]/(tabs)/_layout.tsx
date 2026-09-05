import { Tabs } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useColorScheme } from "@/lib/use-color-scheme";
import { THEME } from "@/lib/theme";
export const unstable_settings = { initialRouteName: "build" };
export default function TabsLayout() {
  const { colorScheme } = useColorScheme(); const theme = THEME[colorScheme];
  return <Tabs initialRouteName="build" screenOptions={{ headerShown: false, lazy: true, tabBarHideOnKeyboard: true, tabBarActiveTintColor: theme.primary, tabBarInactiveTintColor: theme.mutedForeground, tabBarStyle: { backgroundColor: theme.background }, tabBarLabelStyle: { fontSize: 11 } }}>
    <Tabs.Screen name="build" options={{ title: "Build", tabBarIcon: ({ color, size }) => <Ionicons name="hammer-outline" color={color} size={size} /> }} />
    <Tabs.Screen name="circuit" options={{ title: "Circuit", tabBarIcon: ({ color, size }) => <Ionicons name="flash-outline" color={color} size={size} /> }} />
    <Tabs.Screen name="block" options={{ title: "Block", tabBarIcon: ({ color, size }) => <Ionicons name="cube-outline" color={color} size={size} /> }} />
    <Tabs.Screen name="creations" options={{ title: "Creations", tabBarIcon: ({ color, size }) => <Ionicons name="grid-outline" color={color} size={size} /> }} />
    <Tabs.Screen name="inbox" options={{ href: null }} />
    <Tabs.Screen name="my-issues" options={{ href: null }} />
    <Tabs.Screen name="chat" options={{ href: null }} />
    <Tabs.Screen name="more" options={{ href: null }} />
  </Tabs>;
}
