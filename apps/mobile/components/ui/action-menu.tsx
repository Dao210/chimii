import { createContext, useEffect, useCallback, useContext, useRef, useState } from "react";
import { ActionSheetIOS, Platform, View, type ActionSheetIOSOptions } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from "./dropdown-menu";
import { Text } from "./text";

type ShowMenu = (options: ActionSheetIOSOptions, onSelect: (index: number) => void) => void;
const Context = createContext<ShowMenu | null>(null);

/** Same actions/permissions as web menus. iOS uses its native action sheet;
 * Android uses the existing RNR menu so every option stays reachable. */
export function ActionMenuProvider({ children }: { children: React.ReactNode }) {
  const [options, setOptions] = useState<ActionSheetIOSOptions | null>(null);
  const trigger = useRef<React.ComponentRef<typeof DropdownMenuTrigger>>(null);
  useEffect(() => { if (options) trigger.current?.open(); }, [options]);
  const callback = useRef<((index: number) => void) | null>(null);
  const insets = useSafeAreaInsets();
  const finish = useCallback((index?: number) => {
    const notify = callback.current;
    if (!notify) return;
    callback.current = null;
    setOptions(null);
    trigger.current?.close();
    if (index !== undefined) notify?.(index);
  }, []);
  const show = useCallback<ShowMenu>((next, onSelect) => {
    if (Platform.OS === "ios") {
      ActionSheetIOS.showActionSheetWithOptions(next, onSelect);
      return;
    }
    callback.current = onSelect;
    setOptions(next);
  }, []);
  const destructive = options?.destructiveButtonIndex;
  return (
    <Context.Provider value={show}>
      {children}
      {Platform.OS !== "ios" && options && (
        <View pointerEvents="box-none" style={{ position: "absolute", top: insets.top + 48, right: 16 }}>
          <DropdownMenu onOpenChange={(open) => {
            if (!open) finish(options?.cancelButtonIndex);
          }}>
            <DropdownMenuTrigger
              ref={trigger}
              accessible={false}
              accessibilityElementsHidden
              importantForAccessibility="no-hide-descendants"
              style={{ width: 1, height: 1 }}
            />
            <DropdownMenuContent>
              {options?.title ? <DropdownMenuLabel>{options.title}</DropdownMenuLabel> : null}
              {options?.options.map((label, index) => (
                <DropdownMenuItem closeOnPress={false} key={index} disabled={options.disabledButtonIndices?.includes(index)}
                  variant={(Array.isArray(destructive) ? destructive.includes(index) : destructive === index) ? "destructive" : "default"}
                  onPress={() => finish(index)}>
                  <Text>{label}</Text>
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </View>
      )}
    </Context.Provider>
  );
}
export function useActionMenu(): ShowMenu {
  const show = useContext(Context);
  if (!show) throw new Error("ActionMenuProvider is missing");
  return show;
}
