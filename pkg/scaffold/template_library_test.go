package scaffold

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateLibrary_Basic(t *testing.T) {
	ctx := context.Background()
	library := NewTemplateLibrary()

	t.Run("GetSpecification", func(t *testing.T) {
		spec, err := library.GetSpecification(ctx)
		require.NoError(t, err)
		assert.NotNil(t, spec)
		assert.NotEmpty(t, spec.Scaffold.Name)
		assert.NotEmpty(t, spec.Categories)
	})

	t.Run("ListTemplates", func(t *testing.T) {
		templates, err := library.ListTemplates(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, templates)
		
		// Check for our expected templates
		expectedTemplates := []string{
			"campaign.yml.tmpl",
			"guild.yml.tmpl", 
			"agents/guild-master.yml.tmpl",
			"agents/developer.yml.tmpl",
			"agents/tester.yml.tmpl",
			"README.md.tmpl",
			"socket-registry.yml.tmpl",
			"database-init.sql.tmpl",
		}
		
		for _, expected := range expectedTemplates {
			assert.Contains(t, templates, expected, "Template %s should be found", expected)
		}
	})

	t.Run("GetTemplate", func(t *testing.T) {
		content, err := library.GetTemplate(ctx, "campaign.yml.tmpl")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, string(content), "Guild Campaign Configuration")
		assert.Contains(t, string(content), "{{ .vars.campaign_name")
	})

	t.Run("ListCategories", func(t *testing.T) {
		categories, err := library.ListCategories(ctx)
		require.NoError(t, err)
		assert.Contains(t, categories, "campaign")
		assert.Contains(t, categories, "guild")
		assert.Contains(t, categories, "agents")
		assert.Contains(t, categories, "documentation")
	})

	t.Run("ListPresets", func(t *testing.T) {
		presets, err := library.ListPresets(ctx)
		require.NoError(t, err)
		assert.Contains(t, presets, "basic-development")
		assert.Contains(t, presets, "full-stack")
	})
}

func TestTemplateLibrary_TemplateRendering(t *testing.T) {
	ctx := context.Background()
	library := NewTemplateLibrary()
	
	// Create a renderer with our template library
	fs, err := NewOSFileSystem("/tmp/test-templates")
	require.NoError(t, err)
	renderer := NewRenderer(library.GetFileSystem(), fs)
	
	t.Run("RenderCampaignTemplate", func(t *testing.T) {
		// Create test data
		vars := map[string]any{
			"campaign_name": "test-campaign",
			"scaffold_version": "1.0.0",
			"description": "A test campaign",
			"daemon_enabled": true,
			"providers": map[string]any{
				"anthropic": map[string]any{
					"api_key_env": "ANTHROPIC_API_KEY",
					"default_model": "claude-3-sonnet-20240229",
				},
			},
		}
		
		computed := map[string]any{
			"project_type": "go",
		}
		
		runtime := map[string]any{
			"user": "testuser",
			"os": "darwin",
		}
		
		// Add the computed and runtime data to vars so they're accessible in templates
		vars["computed"] = computed
		vars["runtime"] = runtime
		
		renderCtx := RenderContext{
			Vars: vars,
			Recipe: &Recipe{
				TemplatesDir: "templates",
			},
		}
		
		content, err := renderer.RenderTemplate(ctx, "campaign.yml.tmpl", renderCtx)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		
		contentStr := string(content)
		assert.Contains(t, contentStr, "test-campaign")
		assert.Contains(t, contentStr, "daemon:")
		assert.Contains(t, contentStr, "enabled: true")
	})
}

func TestTemplateLibrary_VariableValidation(t *testing.T) {
	ctx := context.Background()
	library := NewTemplateLibrary()

	t.Run("ValidateRequiredVariables", func(t *testing.T) {
		// Missing required variable
		vars := map[string]interface{}{
			"description": "test",
		}
		
		err := library.ValidateVariables(ctx, vars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required variable missing")
	})

	t.Run("ValidateVariableTypes", func(t *testing.T) {
		vars := map[string]interface{}{
			"campaign_name": "test-campaign",
			"max_agents": "not-a-number", // Should be integer
		}
		
		err := library.ValidateVariables(ctx, vars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "variable must be an integer")
	})

	t.Run("ValidateVariablePattern", func(t *testing.T) {
		vars := map[string]interface{}{
			"campaign_name": "invalid name with spaces", // Should match identifier pattern
		}
		
		err := library.ValidateVariables(ctx, vars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not match required pattern")
	})

	t.Run("ValidateEnum", func(t *testing.T) {
		vars := map[string]interface{}{
			"campaign_name": "test-campaign",
			"guild_type": "invalid-type", // Should be one of the enum values
		}
		
		err := library.ValidateVariables(ctx, vars)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in allowed enum")
	})

	t.Run("ValidateValidVariables", func(t *testing.T) {
		vars := map[string]interface{}{
			"campaign_name": "test-campaign",
			"guild_type": "development",
			"max_agents": 5,
		}
		
		err := library.ValidateVariables(ctx, vars)
		assert.NoError(t, err)
	})
}

func TestTemplateLibrary_Stats(t *testing.T) {
	ctx := context.Background()
	library := NewTemplateLibrary()

	stats, err := library.GetStats(ctx)
	require.NoError(t, err)
	assert.Greater(t, stats.TotalTemplates, 0)
	assert.Greater(t, stats.CategoriesCount, 0)
	assert.Greater(t, stats.PresetsCount, 0)
}