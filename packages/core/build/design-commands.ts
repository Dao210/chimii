import { BuildDesignSpecSchema } from "./schemas";
import type { BuildDesignSpec, BuildShapeNode } from "./types";

export type BuildDesignCommand =
  | { type: "replace_shape"; shape: BuildShapeNode }
  | { type: "add_shape"; shape: BuildShapeNode }
  | { type: "remove_shape"; id: string };

// Commands alter target shapes. Only server compilation can turn the draft into
// physical placements and a validated creation revision.
export function applyBuildDesignCommand(design: BuildDesignSpec, command: BuildDesignCommand): BuildDesignSpec {
  let shapes = design.shapes;
  switch (command.type) {
    case "replace_shape":
      if (!shapes.some(shape => shape.id === command.shape.id)) throw new Error("Unknown shape");
      shapes = shapes.map(shape => shape.id === command.shape.id ? command.shape : shape);
      break;
    case "add_shape":
      if (shapes.some(shape => shape.id === command.shape.id)) throw new Error("Duplicate shape");
      shapes = [...shapes, command.shape];
      break;
    case "remove_shape":
      if (!shapes.some(shape => shape.id === command.id)) throw new Error("Unknown shape");
      shapes = shapes.filter(shape => shape.id !== command.id);
      break;
  }
  return BuildDesignSpecSchema.parse({ ...design, shapes });
}

export interface BuildDesignHistory { present: BuildDesignSpec; past: BuildDesignSpec[]; future: BuildDesignSpec[] }
export function reduceBuildDesignHistory(history: BuildDesignHistory, command: BuildDesignCommand | { type: "undo" } | { type: "redo" }): BuildDesignHistory {
  if (command.type === "undo") {
    const present = history.past.at(-1);
    return present ? { present, past: history.past.slice(0, -1), future: [history.present, ...history.future] } : history;
  }
  if (command.type === "redo") {
    const present = history.future[0];
    return present ? { present, past: [...history.past, history.present], future: history.future.slice(1) } : history;
  }
  return { present: applyBuildDesignCommand(history.present, command), past: [...history.past.slice(-49), history.present], future: [] };
}
