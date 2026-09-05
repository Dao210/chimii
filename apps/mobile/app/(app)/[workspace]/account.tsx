import { router } from "expo-router";
import { Text } from "@/components/ui/text";
import {
  MakerScroll,
  Panel,
  RowLink,
  MakerDetailHeader,
  useMaker,
} from "@/components/maker/shared";
import { SafeAreaView } from "react-native-safe-area-context";
export default function Account() {
  const { slug, user } = useMaker();
  return (
    <SafeAreaView edges={["bottom"]} className="flex-1 bg-background">
      <MakerDetailHeader title="家庭与账户" />
      <MakerScroll>
        <Panel>
          <Text className="text-xl font-bold">{user?.name || "我的账户"}</Text>
          <Text className="text-muted-foreground">{user?.email}</Text>
        </Panel>
        {[
          ["家庭 / 空间", "switch-workspace"],
          ["账户与设置", "more/settings"],
          ["消息", "inbox"],
          ["对话", "chat"],
          ["我的任务", "my-issues"],
          ["协作项目", "more/projects"],
          ["全部任务", "more/issues"],
          ["已固定", "more/pins"],
        ].map(([label, path]) => (
          <RowLink
            key={path}
            label={label!}
            onPress={() => router.push(`/${slug}/${path}`)}
          />
        ))}
      </MakerScroll>
    </SafeAreaView>
  );
}
