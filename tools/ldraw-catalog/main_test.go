package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseSourceAndComposeTransform(t *testing.T) {
	lines, err := parseSource([]byte("1 16 10 20 30 0 -1 0 1 0 0 0 0 1 s\\child.dat\n3 4 0 0 0 2 0 0 0 2 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].Reference != "s/child.dat" || lines[1].Color != 4 {
		t.Fatalf("unexpected parsed lines: %#v", lines)
	}
	got := compose(transform{M: [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}, T: vec3{X: 5}}, lines[0].Transform).point(vec3{X: 2})
	want := vec3{X: 15, Y: 22, Z: 30}
	if got != want {
		t.Fatalf("transformed point = %#v, want %#v", got, want)
	}
}

func TestCompilePartProducesDeterministicGLBAndDependencyClosure(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "tiny.zip")
	files := map[string]string{
		"ldraw/LDConfig.ldr":   "0 LDraw config\n0 !COLOUR Red CODE 4 VALUE #C91A09 EDGE #333333\n",
		"ldraw/parts/test.dat": "0 Test Part\n0 !LICENSE Licensed under CC BY 4.0\n1 16 0 0 0 1 0 0 0 1 0 0 0 1 tri.dat\n",
		"ldraw/p/tri.dat":      "3 16 0 0 0 20 0 0 0 0 20\n4 16 0 0 0 0 8 0 20 8 0 20 0 0\n",
	}
	writeZip(t, archivePath, files)
	library, err := openLibrary(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer library.close()
	colors, err := library.colors()
	if err != nil {
		t.Fatal(err)
	}
	first, err := library.compilePart("test.dat", colors)
	if err != nil {
		t.Fatal(err)
	}
	second, err := library.compilePart("test.dat", colors)
	if err != nil {
		t.Fatal(err)
	}
	if first.Hash != second.Hash || first.Data != second.Data {
		t.Fatal("catalog output is not deterministic")
	}
	if first.TriangleCount != 3 || !reflect.DeepEqual(first.Dependencies, []string{"p/tri.dat", "parts/test.dat"}) {
		t.Fatalf("unexpected asset metadata: %#v", first)
	}
	glb, err := base64.StdEncoding.DecodeString(first.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(glb) < 20 || binary.LittleEndian.Uint32(glb[:4]) != 0x46546c67 || binary.LittleEndian.Uint32(glb[4:8]) != 2 {
		t.Fatalf("invalid GLB header: %x", glb[:min(len(glb), 12)])
	}
}

func TestVerifyArchiveFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(path, []byte("not the pinned archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyArchive(path, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("expected SHA mismatch")
	}
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	var buffer bytes.Buffer
	w := zip.NewWriter(&buffer)
	for name, body := range files {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}
