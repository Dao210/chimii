# LDraw attribution

CHIMII Build Studio exports model files that reference part identifiers from the
[LDraw Parts Library](https://library.ldraw.org/). LDraw is an unofficial,
community-maintained CAD system and is not sponsored, endorsed, or authorized
by the LEGO Group.

The application does not bundle or embed LDraw `.dat` source files in exported
MPD files. A compatible LDraw viewer resolves the referenced files from its own
installed Parts Library.

The Starter Kit v1 allowlist references only parts marked **Official** by the
LDraw Parts Library. The referenced current files declare CC BY 4.0 licensing;
consult each installed file's `!LICENSE` header and the library's
`CAreadme.txt` as the authoritative license record.

LDraw™ is a trademark owned and licensed by the Estate of James Jessiman.
LEGO® is a registered trademark of the LEGO Group.

Starter Kit v1 references:

- `3001.dat` — Brick 2 x 4
- `3003.dat` — Brick 2 x 2
- `3004.dat` — Brick 1 x 2
- `3005.dat` — Brick 1 x 1
- `3020.dat` — Plate 2 x 4
- `3022.dat` — Plate 2 x 2
- `3023.dat` — Plate 1 x 2
- `3039.dat` — Roof Slope 2 x 2
- `4600.dat` — Plate 2 x 2 with 2 Wheel Pins
- `4624c04.dat` — Wheel Rim 6.4 x 8 with Tyre 8/75 x 8

# BrickGPT algorithm attribution

`shape-seams.go`, `shape-solver.go` and `shape-repair.go` adapt the cumulative
gap priority, component priority and critical-neighborhood retiling ideas from
[Ava Pun et al., BrickGPT](https://github.com/AvaLovelace1/BrickGPT), specifically
`src/mesh2brick/src/mesh2brick/voxel2brick.py` at commit
`da01aab83f646a700e2270b66b4c00180523448b`. Source SHA-256:
`e3fe42c2ae9b91aa5cddfba0c52108b5dc7324c4c1fde9cd502fe1ce7b256b7a`.

Chimii uses deterministic, bounded Go searches over its own certified catalog,
plate coordinates, target features and reusable inventory. It does not port
the upstream model, implicit baseplate, force solver, random merge operations
or inventory-skipping instruction exporter into the production compiler.
These heuristics are not physical validation.

The upstream license is reproduced below:

MIT License

Copyright (c) 2025 Ava Pun

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
