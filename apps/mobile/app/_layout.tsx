import { ActivityIndicator, View } from "react-native";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import "../global.css";
import { ActionMenuProvider } from "@/components/ui/action-menu";

import { useEffect, useRef } from "react";
import { Stack } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { GestureHandlerRootView } from "react-native-gesture-handler";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { KeyboardProvider } from "react-native-keyboard-controller";
import { QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@react-navigation/native";
import { PortalHost } from "@rn-primitives/portal";
import { api } from "@/data/api";
import { queryClient } from "@/data/query-client";
import { useAuthStore } from "@/data/auth-store";
import { LightboxProvider } from "@/lib/markdown";
import { NAV_THEME } from "@/lib/theme";
import { useColorScheme } from "@/lib/use-color-scheme";

function AuthInitializer({ children }: { children: React.ReactNode }) {
  const initialize = useAuthStore((s) => s.initialize);
  const isLoading = useAuthStore((s) => s.isLoading);
  const error = useAuthStore((s) => s.initializationError);
  // Idempotent guard: 401 on multiple in-flight requests would otherwise
  // logout/navigate repeatedly during the same session-expire moment.
  const signingOutRef = useRef(false);

  useEffect(() => {
    // Wire 401 handling onto the shared ApiClient singleton. Must be set
    // before any request fires — initialize() below kicks off the first
    // getMe() call, so do this synchronously first.
    api.setOptions({
      onUnauthorized: () => {
        if (signingOutRef.current) return;
        signingOutRef.current = true;
        void (async () => {
          try { await useAuthStore.getState().logout(); }
          catch { /* Auth/cache are already cleared even if storage fails. */ }
          // Reset on next tick so a fresh session can hit 401 again later
          // without being silently swallowed.
          setTimeout(() => {
            signingOutRef.current = false;
          }, 0);
        })();
      },
    });
    initialize();
  }, [initialize]);

  if (isLoading || error) return (
    <View className="flex-1 items-center justify-center gap-4 bg-background px-6">
      {isLoading ? <ActivityIndicator /> : <>
        <Text className="text-center text-foreground">{error}</Text>
        <Button onPress={() => void initialize()}><Text>Retry</Text></Button>
      </>}
    </View>
  );
  return <>{children}</>;
}

export default function RootLayout() {
  const { colorScheme, isDarkColorScheme } = useColorScheme();
  const userId = useAuthStore((s) => s.user?.id ?? "signed-out");
  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <SafeAreaProvider>
        <KeyboardProvider>
          <QueryClientProvider client={queryClient}>
            <ThemeProvider value={NAV_THEME[colorScheme]}>
              <AuthInitializer>
                <LightboxProvider key={userId}>
                  <ActionMenuProvider>
                  <StatusBar style={isDarkColorScheme ? "light" : "dark"} />
                  <Stack screenOptions={{ headerShown: false }}>
                    <Stack.Screen name="index" />
                    <Stack.Screen name="(auth)" />
                    <Stack.Screen name="(app)" />
                  </Stack>
                  <PortalHost />
                </ActionMenuProvider>
                </LightboxProvider>
              </AuthInitializer>
            </ThemeProvider>
          </QueryClientProvider>
        </KeyboardProvider>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}
