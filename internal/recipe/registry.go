package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

type Registry struct {
	dir     string
	recipes map[string]Recipe
}

func LoadRegistry(dir string) (Registry, error) {
	if dir == "" {
		dir = FindRecipesDir()
	}
	list, err := LoadDir(dir)
	if err != nil {
		return Registry{}, err
	}
	reg := Registry{dir: dir, recipes: map[string]Recipe{}}
	for _, r := range list {
		if err := ValidateRecipe(r); err != nil {
			return Registry{}, fmt.Errorf("%s: %w", r.ID, err)
		}
		reg.recipes[r.ID] = r
	}
	return reg, nil
}

func (r Registry) Get(id string) (Recipe, bool) {
	recipe, ok := r.recipes[id]
	return recipe, ok
}

func (r Registry) Exists(id string) bool {
	_, ok := r.recipes[id]
	return ok
}

func (r Registry) List() []Recipe {
	out := make([]Recipe, 0, len(r.recipes))
	for _, rec := range r.recipes {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r Registry) SupportedList() []Recipe {
	var out []Recipe
	for _, rec := range r.List() {
		if SupportsOS(rec, runtime.GOOS) {
			out = append(out, rec)
		}
	}
	return out
}

func FindRecipesDir() string {
	candidates := []string{}
	// Deployment override: a binary placed away from its source tree
	// (e.g. an asset-cache/app-bundle install) cannot resolve recipes
	// via cwd/exe-relative paths. AGENTLINK_RECIPES_DIR makes the
	// recipe surface deterministic for embedders (V0300 product).
	if d := os.Getenv("AGENTLINK_RECIPES_DIR"); d != "" {
		candidates = append(candidates, d)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "recipes"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "recipes"), filepath.Join(dir, "..", "recipes"), filepath.Join(dir, "..", "..", "recipes"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c
		}
	}
	return "recipes"
}
