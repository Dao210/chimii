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

const shape = (id, label, kind, x, y, z, sx, sy, sz, color) => ({ id, label, kind, operation: "add", position: { x, y, z }, size: { x: sx, y: sy, z: sz }, color });
const clockRecipe = {
  ...recipe, version: 3, subject: "clock", title: "积木时钟", summary: "用圆形钟面、静态指针和刻度组成的桌面造型", modules: [],
  constraints: { exact_colors: true, no_wheels: true, part_count: 0, required_modules: [] },
  design: { version: 1, mode: "static", shapes: [
    shape("dial", "钟面", "ellipse", -5, 0, -5, 10, 6, 10, 15),
    shape("minute", "分针", "box", -1, 6, -3, 1, 3, 4, 1),
    shape("hour", "时针", "box", 0, 6, 0, 3, 3, 1, 4),
    shape("north", "上方刻度", "box", 0, 6, -4, 1, 3, 1, 14),
    shape("south", "下方刻度", "box", 0, 6, 3, 1, 3, 1, 14),
    shape("west", "左侧刻度", "box", -4, 6, 0, 1, 3, 1, 14),
    shape("east", "右侧刻度", "box", 3, 6, 0, 1, 3, 1, 14),
  ] },
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
    if (input.request && input.projects) {
      const idea = input.request.idea;
      if (/蓝牙|Bluetooth|定时|timer/i.test(idea)) {
        decision = { outcome: "unsupported", message: "This digital kit combination does not support Bluetooth or timers." };
      } else {
        const previous = input.request.previous_composition;
        let composition;
        if (/改成灯|change to a light/i.test(idea) && previous) {
          composition = { ...previous, inputs: [...previous.inputs], output: "BOS0017-R" };
        } else if (/同时|both|且/i.test(idea)) {
          composition = { inputs: ["BOS0002-R", "BOS0013"], operation: "and", output: "BOS0021" };
        } else if (/松开|反转|invert/i.test(idea)) {
          composition = { inputs: ["BOS0002-R"], operation: "not", output: "BOS0017-R" };
        } else if (/运动|motion/i.test(idea)) {
          composition = { inputs: ["BOS0013"], operation: "direct", output: "BOS0017-R" };
        } else {
          composition = { inputs: ["BOS0002-R"], operation: "direct", output: "BOS0021" };
        }
        decision = { outcome: "ready", title: composition.output === "BOS0021" ? "My signal fan" : "My signal light", composition };
      }
    } else if (input.idea.includes("真的飞起来")) {
      decision = { outcome: "unsupported", message: "当前还不能制作真正飞起来的机构。" };
    } else if (input.idea.includes("换一个颜色") && Object.keys(input.history).length === 0) {
      const next = structuredClone(input.draft);
      delete next.design;
      decision = { outcome: "clarify", recipe: next, question: { prompt: "钟面想换成什么颜色？", choices: [{ id: "red", label: "红色" }], allow_free_text: true } };
    } else if (input.idea.includes("钟") || input.draft?.design) {
      if (input.idea.includes("换一个颜色") && !input.draft?.design) throw new Error("Edit clarification lost the source design");
      const next = structuredClone(input.draft?.design ? input.draft : clockRecipe);
      if (input.idea.includes("红色") || Object.values(input.history).includes("红色") || Object.values(input.history).includes("red")) next.design.shapes[0].color = 4;
      decision = { outcome: "ready", recipe: next };
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
