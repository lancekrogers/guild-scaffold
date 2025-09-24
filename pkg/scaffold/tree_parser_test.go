package scaffold

import (
	"reflect"
	"testing"
)

func TestTreeParser_ParseTreeFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Recipe
		wantErr bool
	}{
		{
			name: "minimal tree structure",
			input: `
_scaffold_version: "1.0.0"
_templates_dir: "templates"

project/:
  _files:
    README.md: readme.tmpl
    main.go: main.tmpl
`,
			want: &Recipe{
				ScaffoldVersion: "1.0.0",
				TemplatesDir:    "templates",
				Files: []FileEntry{
					{Path: "project/README.md", Template: "readme.tmpl", With: make(map[string]any)},
					{Path: "project/main.go", Template: "main.tmpl", With: make(map[string]any)},
				},
				Vars: make(map[string]any),
			},
			wantErr: false,
		},
		{
			name: "nested directories",
			input: `
project/:
  src/:
    _files:
      main.go: src/main.tmpl
    utils/:
      _files:
        helper.go: utils/helper.tmpl
  docs/:
    _files:
      README.md: docs/readme.tmpl
`,
			want: &Recipe{
				ScaffoldVersion: "1.0.0",
				TemplatesDir:    "templates",
				Files: []FileEntry{
					{Path: "project/src/main.go", Template: "src/main.tmpl", With: make(map[string]any)},
					{Path: "project/src/utils/helper.go", Template: "utils/helper.tmpl", With: make(map[string]any)},
					{Path: "project/docs/README.md", Template: "docs/readme.tmpl", With: make(map[string]any)},
				},
				Vars: make(map[string]any),
			},
			wantErr: false,
		},
		{
			name: "empty directories with _empty marker",
			input: `
project/:
  logs/:
    _empty: true
  data/:
    _empty: true
`,
			want: &Recipe{
				ScaffoldVersion: "1.0.0",
				TemplatesDir:    "templates",
				Files: []FileEntry{
					{Path: "project/logs/.gitkeep", Template: "~", With: make(map[string]any)},
					{Path: "project/data/.gitkeep", Template: "~", With: make(map[string]any)},
				},
				Vars: make(map[string]any),
			},
			wantErr: false,
		},
		{
			name: "with global variables",
			input: `
_vars:
  project_name: TestProject
  version: "1.0.0"

project/:
  _files:
    config.yaml: config.tmpl
`,
			want: &Recipe{
				ScaffoldVersion: "1.0.0",
				TemplatesDir:    "templates",
				Files: []FileEntry{
					{Path: "project/config.yaml", Template: "config.tmpl", With: make(map[string]any)},
				},
				Vars: map[string]any{
					"project_name": "TestProject",
					"version":      "1.0.0",  // Using string to avoid float parsing issues
				},
			},
			wantErr: false,
		},
		{
			name: "mixed files and directories",
			input: `
app/:
  _files:
    README.md: readme.tmpl
    .gitignore: gitignore.tmpl
  cmd/:
    myapp/:
      _files:
        main.go: cmd/main.tmpl
  pkg/:
    _files:
      lib.go: pkg/lib.tmpl
`,
			want: &Recipe{
				ScaffoldVersion: "1.0.0",
				TemplatesDir:    "templates",
				Files: []FileEntry{
					{Path: "app/README.md", Template: "readme.tmpl", With: make(map[string]any)},
					{Path: "app/.gitignore", Template: "gitignore.tmpl", With: make(map[string]any)},
					{Path: "app/cmd/myapp/main.go", Template: "cmd/main.tmpl", With: make(map[string]any)},
					{Path: "app/pkg/lib.go", Template: "pkg/lib.tmpl", With: make(map[string]any)},
				},
				Vars: make(map[string]any),
			},
			wantErr: false,
		},
		{
			name: "special file markers",
			input: `
project/:
  _files:
    .env: "{ENV_FILE}"
    .uuid: "{UUID_FILE}"
    empty.txt: "~"
`,
			want: &Recipe{
				ScaffoldVersion: "1.0.0",
				TemplatesDir:    "templates",
				Files: []FileEntry{
					{Path: "project/.env", Template: "{ENV_FILE}", With: make(map[string]any)},
					{Path: "project/.uuid", Template: "{UUID_FILE}", With: make(map[string]any)},
					{Path: "project/empty.txt", Template: "~", With: make(map[string]any)},
				},
				Vars: make(map[string]any),
			},
			wantErr: false,
		},
		{
			name: "invalid yaml",
			input: `
this is not valid yaml
  - list item
    key: value
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := NewTreeParser()
			got, err := tp.ParseTreeFormat([]byte(tt.input))
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTreeFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if got.ScaffoldVersion != tt.want.ScaffoldVersion {
					t.Errorf("ScaffoldVersion = %v, want %v", got.ScaffoldVersion, tt.want.ScaffoldVersion)
				}
				
				if got.TemplatesDir != tt.want.TemplatesDir {
					t.Errorf("TemplatesDir = %v, want %v", got.TemplatesDir, tt.want.TemplatesDir)
				}
				
				if !reflect.DeepEqual(got.Vars, tt.want.Vars) {
					t.Errorf("Vars = %v, want %v", got.Vars, tt.want.Vars)
				}
				
				// Sort files for comparison
				if len(got.Files) != len(tt.want.Files) {
					t.Errorf("Files length = %v, want %v", len(got.Files), len(tt.want.Files))
				} else {
					// Create maps for easier comparison
					gotMap := make(map[string]string)
					wantMap := make(map[string]string)
					
					for _, f := range got.Files {
						gotMap[f.Path] = f.Template
					}
					
					for _, f := range tt.want.Files {
						wantMap[f.Path] = f.Template
					}
					
					if !reflect.DeepEqual(gotMap, wantMap) {
						t.Errorf("Files mapping = %v, want %v", gotMap, wantMap)
					}
				}
			}
		})
	}
}

func TestTreeParser_ConvertRecipeToTree(t *testing.T) {
	tp := NewTreeParser()
	
	recipe := &Recipe{
		ScaffoldVersion: "1.0.0",
		TemplatesDir:    "templates",
		Files: []FileEntry{
			{Path: "project/README.md", Template: "readme.tmpl", With: make(map[string]any)},
			{Path: "project/src/main.go", Template: "main.tmpl", With: make(map[string]any)},
			{Path: "project/src/utils/helper.go", Template: "helper.tmpl", With: make(map[string]any)},
		},
		Vars: map[string]any{
			"project_name": "TestProject",
		},
	}
	
	tree, err := tp.ConvertRecipeToTree(recipe)
	if err != nil {
		t.Fatalf("ConvertRecipeToTree() error = %v", err)
	}
	
	// Check metadata
	if tree["_scaffold_version"] != "1.0.0" {
		t.Errorf("_scaffold_version = %v, want %v", tree["_scaffold_version"], "1.0.0")
	}
	
	if tree["_templates_dir"] != "templates" {
		t.Errorf("_templates_dir = %v, want %v", tree["_templates_dir"], "templates")
	}
	
	// Check vars
	vars, ok := tree["_vars"].(map[string]any)
	if !ok {
		t.Fatal("_vars is not a map")
	}
	
	if vars["project_name"] != "TestProject" {
		t.Errorf("project_name = %v, want %v", vars["project_name"], "TestProject")
	}
	
	// Check project structure
	project, ok := tree["project/"].(map[string]interface{})
	if !ok {
		t.Fatal("project/ is not a map")
	}
	
	// Check root files
	files, ok := project["_files"].(map[string]interface{})
	if !ok {
		t.Fatal("project/_files is not a map")
	}
	
	if files["README.md"] != "readme.tmpl" {
		t.Errorf("README.md = %v, want %v", files["README.md"], "readme.tmpl")
	}
	
	// Check nested structure
	src, ok := project["src/"].(map[string]interface{})
	if !ok {
		t.Fatal("project/src/ is not a map")
	}
	
	srcFiles, ok := src["_files"].(map[string]interface{})
	if !ok {
		t.Fatal("project/src/_files is not a map")
	}
	
	if srcFiles["main.go"] != "main.tmpl" {
		t.Errorf("main.go = %v, want %v", srcFiles["main.go"], "main.tmpl")
	}
}

func TestTreeParser_RoundTrip(t *testing.T) {
	// Test that we can parse a tree, convert back, and get the same result
	tp := NewTreeParser()
	
	originalYAML := `
_scaffold_version: "1.0.0"
_templates_dir: "templates"
_vars:
  name: TestProject
  version: 1.0.0

myproject/:
  src/:
    _files:
      main.go: src/main.tmpl
    utils/:
      _files:
        helper.go: utils/helper.tmpl
  docs/:
    _files:
      README.md: docs/readme.tmpl
  _files:
    .gitignore: gitignore.tmpl
`
	
	// Parse the tree format
	recipe, err := tp.ParseTreeFormat([]byte(originalYAML))
	if err != nil {
		t.Fatalf("ParseTreeFormat() error = %v", err)
	}
	
	// Convert back to tree
	tree, err := tp.ConvertRecipeToTree(recipe)
	if err != nil {
		t.Fatalf("ConvertRecipeToTree() error = %v", err)
	}
	
	// Parse the tree again to get a second recipe
	// For this test, we'll just verify the tree structure
	if tree["_scaffold_version"] != "1.0.0" {
		t.Errorf("Round trip failed: scaffold_version = %v", tree["_scaffold_version"])
	}
	
	if tree["_templates_dir"] != "templates" {
		t.Errorf("Round trip failed: templates_dir = %v", tree["_templates_dir"])
	}
	
	vars := tree["_vars"].(map[string]any)
	if vars["name"] != "TestProject" {
		t.Errorf("Round trip failed: vars.name = %v", vars["name"])
	}
	
	// Verify file count matches
	if len(recipe.Files) != 4 {
		t.Errorf("Round trip failed: expected 4 files, got %d", len(recipe.Files))
	}
}