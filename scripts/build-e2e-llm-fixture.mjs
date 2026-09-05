// Deterministic loopback fixture for e2e/build.spec.ts, never a deployed planner.
import http from "node:http";

const recipe = {
  version: 2,
  subject: "robot",
  summary: "一个站着的机器人",
  title: "积木机器人",
  requirements: [],
  constraints: { exact_colors: false, no_wheels: false, part_count: 0, required_modules: [] },
  modules: [
    { id: "base", kind: "robot-base", color: 14 },
    { id: "head", kind: "head", parent: "base", port: "head", color: 1 },
    { id: "eyes", kind: "eyes", parent: "head", port: "face", color: 15 },
  ],
};

http.createServer(async (req, res) => {
  try {
    let raw = "";
    for await (const chunk of req) {
      raw += chunk;
      if (raw.length > 100_000) throw new Error("Fixture request too large");
    }
    const body = JSON.parse(raw);
    const input = JSON.parse(body.messages[0].content[0].text);
    let decision;
    if (input.idea.includes("城堡")) {
      decision = { outcome: "unsupported", message: "当前模块还不能表达城堡，请换一个想法。" };
    } else if (input.idea.includes("朋友") && Object.keys(input.history).length === 0) {
      decision = {
        outcome: "clarify",
        recipe: { ...recipe, subject: "friend", title: "积木朋友", summary: "积木朋友，类型待确定", modules: [] },
        question: {
          prompt: "你想做怎样的积木朋友？",
          choices: [{ id: "robot", label: "站着的机器人" }],
          allow_free_text: true,
        },
      };
    } else {
      decision = { outcome: "ready", recipe };
    }
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({
      id: "stub", type: "message", role: "assistant", model: "test",
      content: [{ type: "text", text: JSON.stringify(decision) }],
      stop_reason: "end_turn", usage: { input_tokens: 1, output_tokens: 1 },
    }));
  } catch {
    res.writeHead(400);
    res.end("Invalid local fixture request");
  }
}).listen(55441, "127.0.0.1");
