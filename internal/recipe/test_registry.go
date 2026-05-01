package recipe

func RegistryForTest(recipes map[string]Recipe) Registry {
	return Registry{recipes: recipes}
}
