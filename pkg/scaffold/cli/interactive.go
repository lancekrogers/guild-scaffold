// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// runInteractiveConfiguration runs interactive mode for template configuration
func runInteractiveConfiguration(ctx context.Context, options *InitOptions) error {
	fmt.Println("🎯 Interactive Configuration Mode")
	fmt.Println("=====================================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Template selection if not specified
	if options.TemplateName == "" {
		templateName, err := promptTemplateSelection(reader)
		if err != nil {
			return fmt.Errorf("failed to select template: %w", err)
		}
		options.TemplateName = templateName
	}

	// Project basic information
	if err := promptBasicInfo(reader, options); err != nil {
		return fmt.Errorf("failed to collect basic information: %w", err)
	}

	// Template-specific configuration
	switch options.TemplateName {
	case "campaign":
		if err := promptCampaignConfig(reader, options); err != nil {
			return fmt.Errorf("failed to collect campaign configuration: %w", err)
		}

	case "guild_core_extension":
		if err := promptExtensionConfig(reader, options); err != nil {
			return fmt.Errorf("failed to collect extension configuration: %w", err)
		}

	case "single_agent":
		if err := promptAgentConfig(reader, options); err != nil {
			return fmt.Errorf("failed to collect agent configuration: %w", err)
		}
	}

	// Provider configuration
	if err := promptProviderConfig(reader, options); err != nil {
		return fmt.Errorf("failed to collect provider configuration: %w", err)
	}

	fmt.Println("✅ Configuration complete!")
	fmt.Println()

	return nil
}

// promptTemplateSelection prompts user to select a template
func promptTemplateSelection(reader *bufio.Reader) (string, error) {
	fmt.Println("📋 Available Templates:")
	templates := []struct {
		name        string
		description string
	}{
		{"campaign", "Complete campaign workspace with guild configuration"},
		{"guild_core_extension", "Extension to existing guild-core repository"},
		{"single_agent", "Simple single-agent project"},
		{"multi_guild", "Multiple coordinated guilds"},
		{"research_project", "Research and experimentation workspace"},
	}

	for i, tmpl := range templates {
		fmt.Printf("   %d. %s - %s\n", i+1, tmpl.name, tmpl.description)
	}
	fmt.Println()

	for {
		fmt.Print("Select template (1-5) [1]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			return templates[0].name, nil
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(templates) {
			fmt.Println("Invalid selection. Please enter a number between 1 and", len(templates))
			continue
		}

		return templates[choice-1].name, nil
	}
}

// promptBasicInfo collects basic project information
func promptBasicInfo(reader *bufio.Reader, options *InitOptions) error {
	fmt.Println("📝 Project Information")
	fmt.Println("---------------------")

	// Project description
	description := promptString(reader, "Project description", "")
	if description != "" {
		if options.Variables == nil {
			options.Variables = make(map[string]interface{})
		}
		options.Variables["project_description"] = description
	}

	// Author information
	author := promptString(reader, "Author name", "")
	if author != "" {
		options.Variables["author_name"] = author
	}

	email := promptString(reader, "Author email", "")
	if email != "" {
		options.Variables["author_email"] = email
	}

	fmt.Println()
	return nil
}

// promptCampaignConfig collects campaign-specific configuration
func promptCampaignConfig(reader *bufio.Reader, options *InitOptions) error {
	fmt.Println("👥 Campaign Configuration")
	fmt.Println("------------------------")

	// Coordination style
	coordStyles := []string{"collaborative", "hierarchical", "autonomous"}
	coordStyle := promptChoice(reader, "Coordination style", coordStyles, "collaborative")
	options.Variables["coordination_style"] = coordStyle

	// Communication pattern
	commPatterns := []string{"broadcast", "hub-and-spoke", "mesh"}
	commPattern := promptChoice(reader, "Communication pattern", commPatterns, "broadcast")
	options.Variables["communication_pattern"] = commPattern

	// Team size
	teamSize := promptInt(reader, "Expected team size", 3, 1, 20)
	options.Variables["team_size"] = teamSize

	// Quality tracking
	qualityFocus := promptBool(reader, "Enable comprehensive quality tracking", true)
	options.Variables["track_quality"] = qualityFocus

	fmt.Println()
	return nil
}

// promptExtensionConfig collects guild-core extension configuration
func promptExtensionConfig(reader *bufio.Reader, options *InitOptions) error {
	fmt.Println("🔧 Extension Configuration")
	fmt.Println("-------------------------")

	// Extension type
	extTypes := []string{"command", "agent", "provider", "tool", "full-feature"}
	extType := promptChoice(reader, "Extension type", extTypes, "full-feature")
	options.Variables["extension_type"] = extType

	// Package name
	packageName := promptString(reader, "Go package name", "myextension")
	options.Variables["package_name"] = packageName

	// Include tests
	includeTests := promptBool(reader, "Include comprehensive tests", true)
	options.Variables["include_tests"] = includeTests

	fmt.Println()
	return nil
}

// promptAgentConfig collects single-agent configuration
func promptAgentConfig(reader *bufio.Reader, options *InitOptions) error {
	fmt.Println("🤖 Agent Configuration")
	fmt.Println("---------------------")

	// Agent role
	agentRole := promptString(reader, "Agent role/specialty", "assistant")
	options.Variables["agent_role"] = agentRole

	// Agent capabilities
	capabilities := promptString(reader, "Agent capabilities (comma-separated)", "")
	if capabilities != "" {
		capList := strings.Split(capabilities, ",")
		for i, cap := range capList {
			capList[i] = strings.TrimSpace(cap)
		}
		options.Variables["agent_capabilities"] = capList
	}

	fmt.Println()
	return nil
}

// promptProviderConfig collects LLM provider configuration
func promptProviderConfig(reader *bufio.Reader, options *InitOptions) error {
	fmt.Println("🤖 LLM Provider Configuration")
	fmt.Println("-----------------------------")

	// Primary provider
	providers := []string{"anthropic", "openai", "ollama", "mixed"}
	if options.Provider == "" {
		provider := promptChoice(reader, "Primary LLM provider", providers, "anthropic")
		options.Provider = provider
	}

	if options.Provider != "mixed" && options.Model == "" {
		// Model selection based on provider
		models := getModelsForProvider(options.Provider)
		if len(models) > 0 {
			model := promptChoice(reader, "Default model", models, models[0])
			options.Model = model
		}
	}

	// Advanced settings
	advanced := promptBool(reader, "Configure advanced LLM settings", false)
	if advanced {
		temperature := promptFloat(reader, "Default temperature", 0.3, 0.0, 2.0)
		options.Variables["default_temperature"] = temperature

		maxTokens := promptInt(reader, "Default max tokens", 4000, 1000, 32000)
		options.Variables["default_max_tokens"] = maxTokens
	}

	fmt.Println()
	return nil
}

// Helper functions for interactive prompts

// promptString prompts for a string value
func promptString(reader *bufio.Reader, prompt, defaultValue string) string {
	fmt.Printf("%s", prompt)
	if defaultValue != "" {
		fmt.Printf(" [%s]", defaultValue)
	}
	fmt.Print(": ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

// promptChoice prompts for a choice from a list
func promptChoice(reader *bufio.Reader, prompt string, choices []string, defaultChoice string) string {
	fmt.Printf("%s (%s)", prompt, strings.Join(choices, "/"))
	if defaultChoice != "" {
		fmt.Printf(" [%s]", defaultChoice)
	}
	fmt.Print(": ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultChoice
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultChoice
	}

	// Validate choice
	for _, choice := range choices {
		if strings.EqualFold(input, choice) {
			return choice
		}
	}

	fmt.Printf("Invalid choice. Using default: %s\n", defaultChoice)
	return defaultChoice
}

// promptBool prompts for a boolean value
func promptBool(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	defaultStr := "n"
	if defaultValue {
		defaultStr = "y"
	}

	response := promptString(reader, fmt.Sprintf("%s (y/n)", prompt), defaultStr)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// promptInt prompts for an integer value within a range
func promptInt(reader *bufio.Reader, prompt string, defaultValue, min, max int) int {
	for {
		input := promptString(reader, fmt.Sprintf("%s (%d-%d)", prompt, min, max),
			fmt.Sprintf("%d", defaultValue))

		value, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("Invalid number. Please try again.\n")
			continue
		}

		if value < min || value > max {
			fmt.Printf("Value must be between %d and %d. Please try again.\n", min, max)
			continue
		}

		return value
	}
}

// promptFloat prompts for a float value within a range
func promptFloat(reader *bufio.Reader, prompt string, defaultValue, min, max float64) float64 {
	for {
		input := promptString(reader, fmt.Sprintf("%s (%.1f-%.1f)", prompt, min, max),
			fmt.Sprintf("%.1f", defaultValue))

		value, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Printf("Invalid number. Please try again.\n")
			continue
		}

		if value < min || value > max {
			fmt.Printf("Value must be between %.1f and %.1f. Please try again.\n", min, max)
			continue
		}

		return value
	}
}

// getModelsForProvider returns available models for a given provider
func getModelsForProvider(provider string) []string {
	models := map[string][]string{
		"anthropic": {
			"claude-3-5-sonnet-20241022",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
			"claude-3-opus-20240229",
		},
		"openai": {
			"gpt-4-turbo-preview",
			"gpt-4",
			"gpt-3.5-turbo",
		},
		"ollama": {
			"llama2",
			"codellama",
			"mistral",
		},
	}

	return models[provider]
}
