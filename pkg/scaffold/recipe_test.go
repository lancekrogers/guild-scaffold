package scaffold

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipe_DefaultValues(t *testing.T) {
	recipe := Recipe{
		ScaffoldVersion: "1.0.0",
		TemplatesDir:    "templates",
		Files: []FileEntry{
			{Path: "test.txt", Template: "test.tmpl"},
		},
	}

	assert.Equal(t, "1.0.0", recipe.ScaffoldVersion)
	assert.Equal(t, "templates", recipe.TemplatesDir)
	assert.Len(t, recipe.Files, 1)
	assert.Equal(t, "test.txt", recipe.Files[0].Path)
	assert.Equal(t, "test.tmpl", recipe.Files[0].Template)
}

func TestFileEntry_WithVariables(t *testing.T) {
	file := FileEntry{
		Path:     "config/app.yaml",
		Template: "app.yaml.tmpl",
		With: map[string]any{
			"app_name": "my-app",
			"port":     8080,
		},
	}

	assert.Equal(t, "config/app.yaml", file.Path)
	assert.Equal(t, "app.yaml.tmpl", file.Template)
	assert.Equal(t, "my-app", file.With["app_name"])
	assert.Equal(t, 8080, file.With["port"])
}

func TestOptions_DefaultBehavior(t *testing.T) {
	options := Options{
		Dest: "/tmp/test",
		Dry:  false,
	}

	assert.Equal(t, "/tmp/test", options.Dest)
	assert.False(t, options.Dry)
	assert.False(t, options.Overwrite)
	assert.Nil(t, options.Vars)
}

func TestParseOptions_Defaults(t *testing.T) {
	assert.True(t, DefaultParseOptions.StrictMode)
	assert.Equal(t, int64(1024*1024), DefaultParseOptions.MaxFileSize)
	assert.Equal(t, time.Second, DefaultParseOptions.MaxParseTime)
	assert.True(t, DefaultParseOptions.ValidateSchema)
	assert.True(t, DefaultParseOptions.ValidateSemantics)
	assert.False(t, DefaultParseOptions.AllowExtensions)
}

func TestRenderContext_VariableMerging(t *testing.T) {
	recipe := &Recipe{
		Vars: map[string]any{
			"global_var": "global_value",
			"shared":     "from_global",
		},
	}

	file := FileEntry{
		Path:     "test.txt",
		Template: "test.tmpl",
		With: map[string]any{
			"file_var": "file_value",
			"shared":   "from_file",
		},
	}

	context := RenderContext{
		Vars: map[string]any{
			"global_var": "global_value",
			"file_var":   "file_value",
			"shared":     "from_file", // File vars should override global
		},
		File:   file,
		Recipe: recipe,
	}

	assert.Equal(t, "global_value", context.Vars["global_var"])
	assert.Equal(t, "file_value", context.Vars["file_var"])
	assert.Equal(t, "from_file", context.Vars["shared"]) // File override
}

func TestScaffoldStats_Initialization(t *testing.T) {
	stats := &ScaffoldStats{
		TotalFiles: 5,
	}

	assert.Equal(t, 5, stats.TotalFiles)
	assert.Equal(t, 0, stats.FilesGenerated)
	assert.Equal(t, 0, stats.FilesSkipped)
	assert.Equal(t, 0, stats.FilesFailed)
	assert.Equal(t, time.Duration(0), stats.Duration)
}

func TestRecipe_ComplexVariables(t *testing.T) {
	recipe := Recipe{
		ScaffoldVersion: "1.0.0",
		TemplatesDir:    "templates",
		Vars: map[string]any{
			"simple_string": "hello",
			"number":        42,
			"boolean":       true,
			"nested_map": map[string]any{
				"inner_key": "inner_value",
				"inner_num": 123,
			},
			"array": []any{"item1", "item2", "item3"},
		},
		Files: []FileEntry{
			{
				Path:     "test.txt",
				Template: "test.tmpl",
				With: map[string]any{
					"override": "file_specific",
				},
			},
		},
	}

	// Test simple variables
	assert.Equal(t, "hello", recipe.Vars["simple_string"])
	assert.Equal(t, 42, recipe.Vars["number"])
	assert.Equal(t, true, recipe.Vars["boolean"])

	// Test nested map
	nestedMap, ok := recipe.Vars["nested_map"].(map[string]any)
	require.True(t, ok, "nested_map should be a map[string]any")
	assert.Equal(t, "inner_value", nestedMap["inner_key"])
	assert.Equal(t, 123, nestedMap["inner_num"])

	// Test array
	array, ok := recipe.Vars["array"].([]any)
	require.True(t, ok, "array should be a []any")
	assert.Len(t, array, 3)
	assert.Equal(t, "item1", array[0])
	assert.Equal(t, "item2", array[1])
	assert.Equal(t, "item3", array[2])

	// Test file-specific variables
	assert.Equal(t, "file_specific", recipe.Files[0].With["override"])
}

func TestRecipe_EmptyInitialization(t *testing.T) {
	recipe := Recipe{}

	assert.Empty(t, recipe.ScaffoldVersion)
	assert.Empty(t, recipe.TemplatesDir)
	assert.Nil(t, recipe.Vars)
	assert.Nil(t, recipe.Files)
}

func TestFileEntry_EmptyWith(t *testing.T) {
	file := FileEntry{
		Path:     "test.txt",
		Template: "test.tmpl",
		// With is not initialized
	}

	assert.Equal(t, "test.txt", file.Path)
	assert.Equal(t, "test.tmpl", file.Template)
	assert.Nil(t, file.With)
}

func BenchmarkRecipe_VariableAccess(b *testing.B) {
	recipe := Recipe{
		Vars: map[string]any{
			"var1": "value1",
			"var2": "value2",
			"var3": "value3",
			"var4": "value4",
			"var5": "value5",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = recipe.Vars["var3"]
	}
}

func BenchmarkFileEntry_WithAccess(b *testing.B) {
	file := FileEntry{
		With: map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = file.With["key2"]
	}
}
