package impeller

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFragmentProgram(t *testing.T) {
	if NewFragmentProgram(nil) != nil || NewFragmentProgram([]byte("not a runtime stage")) != nil {
		t.Fatal("invalid data accepted")
	}
	impellerc := filepath.Join("..", "third_party", "impeller_sdk", "bin", "impellerc")
	if _, err := os.Stat(impellerc); err != nil {
		t.Skip("no impellerc")
	}
	dir := t.TempDir()
	src := filepath.Join("..", "..", "engine", "src", "flutter", "lib", "ui", "fixtures", "shaders", "general_shaders", "filter_shader.frag")
	out := filepath.Join(dir, "f.iplr")
	cmd := exec.Command(impellerc, "--runtime-stage-metal", "--iplr", "--sl="+out, "--spirv="+out+".spirv",
		"--input="+src, "--input-type=frag", "--include="+filepath.Join("..", "..", "engine", "src", "flutter", "impeller", "compiler", "shader_lib"))
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, msg)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	p := NewFragmentProgram(data)
	if p == nil {
		t.Fatal("program refused")
	}
	defer p.Release()
	ctx, err := NewMetalContext()
	if err != nil {
		t.Skip(err)
	}
	defer ctx.Release()
	tex := NewTextureRGBA8(ctx, 1, 1, []byte{1, 2, 3, 4})
	defer tex.Release()
	uniforms := make([]byte, 16)
	if s := NewFragmentProgramColorSource(ctx, p, []*Texture{tex}, uniforms); s == nil {
		t.Error("no color source")
	} else {
		s.Release()
	}
	if f := NewFragmentProgramImageFilter(ctx, p, []*Texture{tex}, uniforms); f == nil {
		t.Error("no image filter")
	} else {
		f.Release()
	}
	// A bundle without the Metal stage: no color source.
	vk := filepath.Join(dir, "vk.iplr")
	cmd = exec.Command(impellerc, "--runtime-stage-vulkan", "--iplr", "--sl="+vk, "--spirv="+vk+".spirv",
		"--input="+src, "--input-type=frag", "--include="+filepath.Join("..", "..", "engine", "src", "flutter", "impeller", "compiler", "shader_lib"))
	if msg, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, msg)
	}
	vdata, _ := os.ReadFile(vk)
	vp := NewFragmentProgram(vdata)
	if vp == nil {
		t.Fatal("vulkan-only program refused")
	}
	defer vp.Release()
	if s := NewFragmentProgramColorSource(ctx, vp, []*Texture{tex}, uniforms); s != nil {
		s.Release()
		t.Error("color source without a Metal stage")
	}
}
