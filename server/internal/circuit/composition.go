package circuit

import (
	"errors"
	"fmt"
	"strings"
)

// CompositionSpec describes requested behavior, never model-authored wiring.
// Version one models digital signal states, not sensor timing or motor physics.
type CompositionSpec struct {
	Inputs    []string `json:"inputs"`
	Operation string   `json:"operation"`
	Output    string   `json:"output"`
}

type BehaviorCase struct {
	Inputs   map[string]bool `json:"inputs"`
	Powered  bool            `json:"powered"`
	Expected bool            `json:"expected"`
	Actual   bool            `json:"actual"`
}

type BehaviorReport struct {
	Model  string         `json:"model"`
	Passed bool           `json:"passed"`
	Cases  []BehaviorCase `json:"cases"`
}

func ValidateComposition(c Catalog, spec CompositionSpec) error {
	if c.KitID != "dfrobot-edu0080-en" || c.ConnectionSystem != "boson" || len(spec.Inputs) < 1 || len(spec.Inputs) > 2 {
		return errors.New("unsupported_composition")
	}
	if (spec.Operation == "and" && len(spec.Inputs) != 2) || (spec.Operation != "and" && (len(spec.Inputs) != 1 || (spec.Operation != "direct" && spec.Operation != "not"))) {
		return errors.New("unsupported_operation")
	}
	seen := map[string]bool{}
	for _, id := range spec.Inputs {
		if (id != "BOS0002-R" && id != "BOS0013") || seen[id] {
			return errors.New("unsupported_input")
		}
		seen[id] = true
	}
	if spec.Output != "BOS0017-R" && spec.Output != "BOS0021" {
		return errors.New("unsupported_output")
	}
	for _, id := range append(append([]string{}, spec.Inputs...), spec.Output, "BOS0036", "FIT0529", "BOSON-CABLE-10CM") {
		if _, ok := c.Part(id); !ok {
			return errors.New("unknown_part")
		}
	}
	return nil
}

// ComposeProject lays out a small typed signal graph. Module references supply
// port contracts; this newly composed project is not a manufacturer project.
func ComposeProject(c Catalog, spec CompositionSpec) (Project, error) {
	if err := ValidateComposition(c, spec); err != nil {
		return Project{}, err
	}
	ref, _ := c.Project("boson-button-light")
	p := Project{ID: "boson-composed-" + strings.Join(spec.Inputs, "-") + "-" + spec.Operation + "-" + spec.Output,
		Source: ref.Source, Placements: []Placement{}, Connections: []Connection{}, ExpectedNets: [][]string{}, Steps: []Step{},
		Troubleshooting: []Text{{EN: "Switch m2 off before changing any cable. Match each module marking and IN/OUT label.", ZH: "改线前关闭 m2，逐个核对模块编号及 IN/OUT 标记。"}, {EN: "A motion sensor can hold its signal after movement. Logic checks do not predict its delay or the fan's starting speed.", ZH: "人体感应信号可能延迟恢复；逻辑检查不预测这个延时或风扇起转速度。"}},
	}
	inputNames := []string{}
	zhNames := []string{}
	for _, id := range spec.Inputs {
		part, _ := c.Part(id)
		inputNames = append(inputNames, part.Name.EN)
		zhNames = append(zhNames, part.Name.ZH)
	}
	out, _ := c.Part(spec.Output)
	p.Title = Text{EN: strings.Join(inputNames, " + ") + " → " + out.Name.EN, ZH: strings.Join(zhNames, "＋") + "控制" + out.Name.ZH}
	if spec.Operation == "not" {
		p.Description = Text{EN: "The output is active when the input signal is low, and inactive when it is high.", ZH: "输入信号低时输出开启，输入信号高时输出关闭。"}
	} else if spec.Operation == "and" {
		p.Description = Text{EN: "The output is active only when both input signals are high.", ZH: "两个输入信号都为高时，输出才开启。"}
	} else {
		p.Description = Text{EN: "The output follows the input signal: high activates it, low deactivates it.", ZH: "输出跟随输入信号：高时开启，低时关闭。"}
	}
	p.Explanation = p.Description
	p.TestInstruction = Text{EN: "With an adult, check the connections and clear the fan blades, then switch m2 on. Try each input state shown in the logic check. Turn m2 off before adjusting the assembly.", ZH: "和成人一起检查接线并避开扇叶，再打开 m2。按逻辑检查表逐个尝试输入状态，改动前先关闭 m2。"}
	add := func(id, part string, x, y int) {
		p.Placements = append(p.Placements, Placement{ID: id, PartID: part, X: x, Y: y, Layer: 1})
	}
	add("power", "BOS0036", 13, 6)
	for i, id := range spec.Inputs {
		y := 6
		if len(spec.Inputs) == 2 {
			y = 2 + i*8
		}
		add(fmt.Sprintf("input%d", i+1), id, 1, y)
	}
	if spec.Operation == "and" {
		add("logic", "BOS0027", 7, 6)
	}
	if spec.Operation == "not" {
		add("logic", "BOS0029", 19, 6)
	}
	add("output", spec.Output, 25, 6)
	moduleIDs := []string{}
	for _, v := range p.Placements {
		moduleIDs = append(moduleIDs, v.ID)
	}
	p.Steps = append(p.Steps, Step{ID: "arrange", Title: Text{EN: "Find and arrange the modules", ZH: "找出模块，按图摆好"}, Instruction: Text{EN: "Match the markings on your parts. Keep m2 OFF and leave the battery disconnected. The drawing shows connection order, not mounting scale.", ZH: "核对实物编号，保持 m2 关闭，暂不接电池。图中表示连接顺序，不表示实际安装尺寸。"}, PlacementIDs: moduleIDs})
	wire := func(from, to string) {
		n := len(p.Connections) + 1
		cable := fmt.Sprintf("cable%d", n)
		add(cable, "BOSON-CABLE-10CM", 0, 0)
		p.Connections = append(p.Connections, Connection{ID: fmt.Sprintf("link%d", n), From: from, To: to, CableID: cable})
		endpoint := func(key string) string {
			tokens := strings.Split(key, ":")
			for _, v := range p.Placements {
				if v.ID == tokens[0] {
					part, _ := c.Part(v.PartID)
					return part.Marking + " " + tokens[1]
				}
			}
			return key
		}
		p.Steps = append(p.Steps, Step{ID: cable, Title: Text{EN: "Connect " + endpoint(from) + " → " + endpoint(to), ZH: "连接 " + endpoint(from) + " → " + endpoint(to)}, Instruction: Text{EN: "Take one matching 10 cm BOSON cable. Align the keyed plug with each socket; do not force it. Move the modules closer if needed.", ZH: "拿一根配套的 10 厘米 BOSON 连接线，对准两端插口方向，轻轻插入。如果够不到，先把模块靠近。"}, PlacementIDs: []string{cable}})
	}
	if spec.Operation == "and" {
		wire("input1:OUT", "logic:INA")
		wire("input2:OUT", "logic:INB")
		wire("logic:OUT", "power:IN")
	} else {
		wire("input1:OUT", "power:IN")
	}
	if spec.Operation == "not" {
		wire("power:OUT", "logic:IN")
		wire("logic:OUT", "output:IN")
	} else {
		wire("power:OUT", "output:IN")
	}
	add("battery", "FIT0529", 13, 12)
	p.Connections = append(p.Connections, Connection{ID: "battery-lead", From: "battery:USB", To: "power:USB"})
	p.Steps = append(p.Steps, Step{ID: "battery", Title: Text{EN: "Connect the battery last", ZH: "最后连接电池盒"}, Instruction: Text{EN: "With m2 still OFF, ask an adult to check the 3×AAA holder and battery direction, then connect its matching lead to m2 USB. Check the assembly before switching on.", ZH: "保持 m2 关闭，请成人核对三节 AAA 电池盒及电池方向，再把配套接头连接到 m2 的 USB 口。检查完整搭建后再通电。"}, PlacementIDs: []string{"battery"}})
	for _, w := range p.Connections {
		p.ExpectedNets = append(p.ExpectedNets, []string{w.From, w.To})
	}
	return p, nil
}

func CompileComposition(c Catalog, spec CompositionSpec, prompt, title string, inventory map[string]int) (Document, error) {
	p, err := ComposeProject(c, spec)
	if err != nil {
		return Document{}, err
	}
	if inventory == nil {
		return Document{}, errors.New("inventory_required")
	}
	r := validateModuleGraph(c, nil, p.Placements, p.Connections, inventory)
	b := CheckCompositionBehavior(c, spec, p)
	if !b.Passed {
		r.Passed = false
		r.Issues = append(r.Issues, Issue{Code: "behavior_mismatch"})
	}
	if !r.Passed {
		return Document{Validation: r, Behavior: &b}, errors.New("validation_failed")
	}
	if err := validateSteps(p); err != nil {
		return Document{}, err
	}
	parts := []Part{}
	for _, part := range c.Parts {
		if r.UsedParts[part.ID] > 0 {
			parts = append(parts, part)
		}
	}
	d := Document{Version: 2, ConnectionSystem: "boson", CatalogVersion: c.Version, KitID: c.KitID, Prompt: prompt, Title: title, Planner: "boson-composition-v1", Project: p, Parts: parts, Columns: c.Columns, Rows: c.Rows, Preparation: c.Preparation, Inventory: inventory, Validation: r, Composition: &spec, Behavior: &b}
	return SealDocument(d)
}

// Expected results come from the request; actual results are evaluated by
// traversing concrete module ports. Neither is supplied by the language model.
func CheckCompositionBehavior(c Catalog, spec CompositionSpec, p Project) BehaviorReport {
	r := BehaviorReport{Model: "boson-digital-v1", Cases: []BehaviorCase{}}
	if ValidateComposition(c, spec) != nil {
		return r
	}
	parts := map[string]string{}
	incoming := map[string]string{}
	for _, v := range p.Placements {
		parts[v.ID] = v.PartID
	}
	for _, w := range p.Connections {
		if _, ok := incoming[w.To]; ok {
			return r
		}
		incoming[w.To] = w.From
	}
	for mask := 0; mask < (1<<len(spec.Inputs))+1; mask++ {
		powered := mask < (1 << len(spec.Inputs))
		states := map[string]bool{}
		for i, id := range spec.Inputs {
			states[id] = mask&(1<<i) != 0 || !powered
		}
		expected := states[spec.Inputs[0]]
		if spec.Operation == "not" {
			expected = !expected
		}
		if spec.Operation == "and" {
			expected = expected && states[spec.Inputs[1]]
		}
		expected = expected && powered
		visiting := map[string]bool{}
		var eval func(string) (bool, error)
		eval = func(key string) (bool, error) {
			if visiting[key] {
				return false, errors.New("cycle")
			}
			visiting[key] = true
			defer delete(visiting, key)
			v := strings.Split(key, ":")
			if len(v) != 2 {
				return false, errors.New("missing signal")
			}
			id := parts[v[0]]
			if v[1] != "OUT" {
				return false, errors.New("not an output")
			}
			switch id {
			case "BOS0002-R", "BOS0013":
				state, ok := states[id]
				if !ok {
					return false, errors.New("unexpected input")
				}
				return state, nil
			case "BOS0036":
				battery := strings.Split(incoming[v[0]+":USB"], ":")
				if len(battery) != 2 || parts[battery[0]] != "FIT0529" || battery[1] != "USB" {
					return false, errors.New("missing power")
				}
				value, err := eval(incoming[v[0]+":IN"])
				return value, err
			case "BOS0029":
				value, err := eval(incoming[v[0]+":IN"])
				return !value, err
			case "BOS0027":
				a, err := eval(incoming[v[0]+":INA"])
				if err != nil {
					return false, err
				}
				b, err := eval(incoming[v[0]+":INB"])
				return a && b, err
			default:
				return false, errors.New("unmodeled module")
			}
		}
		if parts["output"] != spec.Output {
			return r
		}
		actual, err := eval(incoming["output:IN"])
		if err != nil {
			return r
		}
		actual = actual && powered
		r.Cases = append(r.Cases, BehaviorCase{Inputs: states, Powered: powered, Expected: expected, Actual: actual})
		if actual != expected {
			return r
		}
	}
	r.Passed = true
	return r
}
