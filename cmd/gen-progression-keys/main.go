package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Node struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type ProgressionTree struct {
	Nodes []Node `json:"nodes"`
}

func main() {
	configPath := flag.String("config", "configs/progression_tree.json", "Path to progression tree config")
	outputPath := flag.String("output", "internal/progression/keys.go", "Path to output keys.go file")
	flag.Parse()

	// Read config
	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var tree ProgressionTree
	if err := json.Unmarshal(data, &tree); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Generate code
	code := generateKeysFile(tree)

	// Ensure output directory exists
	dir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Write file
	if err := os.WriteFile(*outputPath, []byte(code), 0644); err != nil {
		log.Fatalf("Failed to write output file: %v", err)
	}

	fmt.Printf("✓ Generated %s with %d keys\n", *outputPath, len(tree.Nodes))
}

func generateKeysFile(tree ProgressionTree) string {
	// Group keys by type
	groups := make(map[string][]Node)
	for _, node := range tree.Nodes {
		groups[node.Type] = append(groups[node.Type], node)
	}

	// Sort each group by key
	for _, nodes := range groups {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Key < nodes[j].Key
		})
	}

	// Generate code
	var sb strings.Builder
	sb.WriteString(`package progression

// Feature and item keys used throughout the progression system
// This file is auto-generated from configs/progression_tree.json
// Do NOT edit manually - run: make generate
`)

	sb.WriteString("\nconst (\n")

	// System first
	if nodes, ok := groups["feature"]; ok {
		// Find progression_system
		for _, node := range nodes {
			if node.Key == "progression_system" {
				sb.WriteString("\t// System\n")
				sb.WriteString(fmt.Sprintf("\tFeature%s = %q\n", pascalCase(node.Key), node.Key))
				break
			}
		}
	}

	// Items
	if nodes, ok := groups["item"]; ok {
		sb.WriteString("\n\t// Items\n")
		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("\tItem%s = %q\n", pascalCase(stripPrefix(node.Key, "item_")), node.Key))
		}
	}

	// Features (excluding progression_system)
	if nodes, ok := groups["feature"]; ok {
		featureCount := 0
		for _, node := range nodes {
			if node.Key != "progression_system" {
				if featureCount == 0 {
					sb.WriteString("\n\t// Features\n")
				}
				sb.WriteString(fmt.Sprintf("\tFeature%s = %q\n", pascalCase(stripPrefix(node.Key, "feature_")), node.Key))
				featureCount++
			}
		}
	}

	// Upgrades
	if nodes, ok := groups["upgrade"]; ok {
		sb.WriteString("\n\t// Upgrades\n")
		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("\tUpgrade%s = %q\n", pascalCase(stripPrefix(node.Key, "upgrade_")), node.Key))
		}
	}

	// Jobs
	if nodes, ok := groups["job"]; ok {
		sb.WriteString("\n\t// Jobs\n")
		for _, node := range nodes {
			sb.WriteString(fmt.Sprintf("\tJob%s = %q\n", pascalCase(stripPrefix(node.Key, "job_")), node.Key))
		}
	}

	sb.WriteString("\n\t// Session Status\n")
	sb.WriteString("\tSessionStatusVoting = \"voting\"\n")
	sb.WriteString("\tSessionStatusFrozen = \"frozen\"\n")
	sb.WriteString("\tSessionStatusCompleted = \"completed\"\n")

	sb.WriteString(")\n")

	return sb.String()
}

// stripPrefix removes a prefix from a string if present
func stripPrefix(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

// pascalCase converts progression_key to ProgressionKey
func pascalCase(s string) string {
	parts := strings.Split(s, "_")
	var result strings.Builder
	for _, part := range parts {
		if part != "" {
			result.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return result.String()
}
