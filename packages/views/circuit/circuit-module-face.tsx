import type { CircuitPart } from "@chimii/core/circuit";

// Original recognition drawings, not photographs or mounting-scale geometry.
export function CircuitModuleFace({ part }: { part: CircuitPart }) {
  const component = (() => {
    switch (part.id) {
      case "BOS0002-R":
        return (
          <>
            <rect
              x="-19"
              y="-19"
              width="38"
              height="38"
              rx="6"
              className="fill-foreground"
            />
            <circle r="16" className="circuit-component-red" />
            <path
              d="M-9 -8 Q0 -15 9 -8"
              fill="none"
              stroke="white"
              strokeOpacity=".5"
              strokeWidth="3"
            />
          </>
        );
      case "BOS0017-R":
        return (
          <>
            <path
              d="M-6 12 V23 M6 12 V23"
              className="stroke-foreground"
              strokeWidth="2"
            />
            <path
              d="M-11 10 V-6 A11 11 0 0 1 11 -6 V10 Z"
              className="circuit-component-red"
            />
            <path
              d="M-5 -8 V4"
              stroke="white"
              strokeOpacity=".7"
              strokeWidth="3"
            />
          </>
        );
      case "BOS0021":
        return (
          <>
            <circle r="13" className="fill-muted stroke-foreground" />
            <path
              d="M0 -4 C-30 -25 -29 1 -5 4 C18 30 33 11 5 0 C29 -22 8 -33 0 -4"
              className="circuit-component-red"
            />
            <circle r="5" className="fill-foreground" />
          </>
        );
      case "BOS0013":
        return (
          <>
            <circle r="20" className="fill-background stroke-foreground" />
            <path
              d="M-18 0 H18 M-14 -10 H14 M-14 10 H14 M0 -20 Q-16 0 0 20 M0 -20 Q16 0 0 20"
              className="stroke-muted-foreground"
              fill="none"
              strokeWidth="1"
            />
          </>
        );
      case "BOS0001":
        return (
          <>
            <circle r="20" className="fill-muted stroke-foreground" />
            <circle r="15" className="fill-foreground" />
            <path d="M0 -13 V-3" stroke="white" strokeWidth="3" />
          </>
        );
      case "BOS0009":
        return (
          <>
            <circle r="18" className="fill-muted stroke-foreground" />
            {[-9, 0, 9].flatMap((y) =>
              [-9, 0, 9].map((x) => (
                <circle
                  key={`${x}:${y}`}
                  cx={x}
                  cy={y}
                  r="2"
                  className="fill-muted-foreground"
                />
              )),
            )}
          </>
        );
      case "BOS0036":
        return (
          <>
            <rect
              x="-19"
              y="-17"
              width="38"
              height="34"
              rx="3"
              className="fill-muted stroke-foreground"
            />
            <rect
              x="-13"
              y="-11"
              width="14"
              height="15"
              className="fill-foreground"
            />
            <rect
              x="8"
              y="-12"
              width="7"
              height="25"
              className="fill-foreground"
            />
            <path
              d="M-7 10 H0 M-7 14 H0"
              className="stroke-muted-foreground"
              strokeWidth="2"
            />
          </>
        );
      case "FIT0529":
        return (
          <>
            {[-20, -6, 8].map((x) => (
              <g key={x}>
                <rect
                  x={x}
                  y="-22"
                  width="12"
                  height="42"
                  rx="2"
                  className="fill-foreground"
                />
                <rect
                  x={x + 1}
                  y="-21"
                  width="10"
                  height="10"
                  className="circuit-component-metal"
                />
                <text x={x + 6} y="-12" textAnchor="middle" fontSize="9">
                  +
                </text>
              </g>
            ))}
          </>
        );
      case "BOS0029":
      case "BOS0027":
        return (
          <>
            <rect
              x="-19"
              y="-17"
              width="38"
              height="34"
              rx="3"
              className="fill-foreground"
            />
            <text
              textAnchor="middle"
              dominantBaseline="middle"
              fill="white"
              fontSize="12"
              fontWeight="800"
            >
              {part.id === "BOS0029" ? "NOT" : "AND"}
            </text>
          </>
        );
      default:
        return (
          <rect
            x="-15"
            y="-12"
            width="30"
            height="24"
            rx="3"
            className="fill-muted stroke-foreground"
          />
        );
    }
  })();
  return <g transform="translate(52,31)">{component}</g>;
}
