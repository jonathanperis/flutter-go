package ui

// Test assets for the fragment shader tests: the engine's shader fixtures
// (lib/ui/fixtures/shaders), compiled with the Impeller SDK's impellerc as
// the engine's build compiles them (impellerc --iplr with the SkSL, Metal,
// GLES, GLES3 and Vulkan runtime stages, retried without SkSL as the flutter
// tool does), and served through EngineAssetLoader under the names
// fragment_shader_test.dart loads them by.

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// repoRoot is the flutter-go checkout (this package is
// engine/src/flutter/lib/ui/go).
var repoRoot = filepath.Join("..", "..", "..", "..", "..", "..")

var (
	fragmentAssetsOnce sync.Once
	fragmentAssets     map[string][]byte
	fragmentAssetsErr  error
)

// impellercTargets are the runtime stages the engine's fixtures are
// compiled with (lib/ui/fixtures/shaders/*/BUILD.gn).
var impellercTargets = []string{"--sksl", "--runtime-stage-metal", "--runtime-stage-gles", "--runtime-stage-gles3", "--runtime-stage-vulkan"}

// compileShader runs the SDK's impellerc on a .frag file, as
// flutter_tools' ShaderCompiler does, and returns the iplr bundle.
func compileShader(src, outDir string, targets []string) ([]byte, error) {
	impellerc := filepath.Join(repoRoot, "src", "third_party", "impeller_sdk", "bin", "impellerc")
	out := filepath.Join(outDir, filepath.Base(src)+".iplr")
	run := func(targets []string) ([]byte, error) {
		args := append(append([]string{}, targets...),
			"--iplr",
			"--sl="+out,
			"--spirv="+out+".spirv",
			"--input="+src,
			"--input-type=frag",
			"--include="+filepath.Dir(src),
			"--include="+filepath.Join(repoRoot, "engine", "src", "flutter", "impeller", "compiler", "shader_lib"))
		return exec.Command(impellerc, args...).CombinedOutput()
	}
	msg, err := run(targets)
	if err != nil && len(targets) > 1 && targets[0] == "--sksl" {
		msg, err = run(targets[1:])
	}
	if err != nil {
		return nil, errors.New(src + ": " + err.Error() + "\n" + string(msg))
	}
	return os.ReadFile(out)
}

// loadFragmentAssets compiles the fixtures once per process and installs
// an EngineAssetLoader serving them (and DashInNooglerHat.jpg).
func loadFragmentAssets(t testing.TB) map[string][]byte {
	t.Helper()
	fragmentAssetsOnce.Do(func() {
		if _, err := os.Stat(filepath.Join(repoRoot, "src", "third_party", "impeller_sdk", "bin", "impellerc")); err != nil {
			fragmentAssetsErr = err
			return
		}
		dir, err := os.MkdirTemp("", "fragment_shaders")
		if err != nil {
			fragmentAssetsErr = err
			return
		}
		fixtures := filepath.Join(repoRoot, "engine", "src", "flutter", "lib", "ui", "fixtures")
		var srcs []string
		for _, sub := range []string{"general_shaders", "uniforms"} {
			m, _ := filepath.Glob(filepath.Join(fixtures, "shaders", sub, "*.frag"))
			srcs = append(srcs, m...)
		}
		srcs = append(srcs, filepath.Join(repoRoot, "engine", "src", "flutter", "impeller", "fixtures", "ink_sparkle.frag"))
		assets := map[string][]byte{}
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for _, src := range srcs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				data, err := compileShader(src, dir, impellercTargets)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					fragmentAssetsErr = errors.Join(fragmentAssetsErr, err)
					return
				}
				assets[filepath.Base(src)+".iplr"] = data
			}()
		}
		wg.Wait()
		if fragmentAssetsErr != nil {
			return
		}
		// no_uniforms_alt.frag.iplr is a byte-identical copy (BUILD.gn).
		assets["no_uniforms_alt.frag.iplr"] = assets["no_uniforms.frag.iplr"]
		jpg, err := os.ReadFile(filepath.Join(fixtures, "DashInNooglerHat.jpg"))
		if err != nil {
			fragmentAssetsErr = err
			return
		}
		assets["DashInNooglerHat.jpg"] = jpg
		fragmentAssets = assets
		previous := EngineAssetLoader
		EngineAssetLoader = func(key string) ([]byte, error) {
			if data, ok := fragmentAssets[key]; ok {
				return data, nil
			}
			if previous != nil {
				return previous(key)
			}
			return nil, errors.New("no asset " + key)
		}
	})
	if fragmentAssetsErr != nil {
		if strings.Contains(fragmentAssetsErr.Error(), "no such file") && fragmentAssets == nil {
			t.Skipf("impellerc unavailable (run src/scripts/fetch_impeller_sdk.sh): %v", fragmentAssetsErr)
		}
		t.Fatal(fragmentAssetsErr)
	}
	return fragmentAssets
}
