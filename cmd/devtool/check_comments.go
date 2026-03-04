package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CheckCommentsCommand struct{}

func (c *CheckCommentsCommand) Name() string {
	return "check-comments"
}

func (c *CheckCommentsCommand) Description() string {
	return "Check for duplicate, long, and commented-out code in Go files"
}

func (c *CheckCommentsCommand) Run(args []string) error {
	fs := flag.NewFlagSet("check-comments", flag.ContinueOnError)
	maxLen := fs.Int("max-len", 1000, "Maximum allowed length for a comment line")
	excludeDir := fs.String("exclude", "vendor,node_modules,.git,mocks,internal/database/generated", "Directories to exclude")

	if err := fs.Parse(args); err != nil {
		return err
	}

	targets := fs.Args()
	if len(targets) == 0 {
		targets = []string{"."}
	}

	PrintHeader("Checking comments...")
	PrintInfo("Max comment length: %d", *maxLen)

	var violations int
	walkErr := c.walkTargets(targets, strings.Split(*excludeDir, ","), &violations, *maxLen)

	if walkErr != nil {
		return walkErr
	}

	if violations > 0 {
		PrintError("Found %d comment violations", violations)
		return fmt.Errorf("comment check failed with %d violations", violations)
	}

	PrintSuccess("No comment violations found!")
	return nil
}

func (c *CheckCommentsCommand) walkTargets(targets []string, excludes []string, total *int, maxLen int) error {
	for _, target := range targets {
		err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				for _, ex := range excludes {
					if info.Name() == ex || path == ex || strings.HasPrefix(path, ex) {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if c.isCheckable(path) {
				v, err := c.checkFile(path, maxLen)
				if err != nil {
					return err
				}
				if v > 0 {
					*total += v
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CheckCommentsCommand) isCheckable(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".go" && !strings.HasSuffix(path, "_generated.go") && !strings.HasSuffix(path, ".pb.go")
}

type commentBlock struct {
	startLine int
	lines     []string
}

func (c *CheckCommentsCommand) checkFile(path string, maxLen int) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	violations := 0
	inBlockComment := false
	var currentBlock *commentBlock

	// Check if the file has any functions
	hasFuncs := c.hasFunctions(allLines)

	for i, line := range allLines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		isComment, commentContent, isPureComment := c.processLine(trimmedLine, &inBlockComment)

		if isComment {
			trimmedContent := strings.TrimSpace(commentContent)

			// 1. Long comment check - applies to ALL comments
			if len(line) > maxLen && !c.isSpecialComment(trimmedContent) {
				c.report(path, lineNum, "comment line too long (%d > %d characters)", len(line), maxLen)
				violations++
			}

			// Collect into block ONLY if it's a pure comment
			if isPureComment {
				if currentBlock == nil {
					currentBlock = &commentBlock{startLine: lineNum}
				}
				currentBlock.lines = append(currentBlock.lines, trimmedContent)
			} else if currentBlock != nil {
				// Pure comment followed by end-of-line comment ends the block
				violations += c.analyzeBlock(path, currentBlock, allLines, hasFuncs)
				currentBlock = nil
			}
		} else if currentBlock != nil {
			// End of comment block
			violations += c.analyzeBlock(path, currentBlock, allLines, hasFuncs)
			currentBlock = nil
		}
	}

	// Final block if file ends with comment
	if currentBlock != nil {
		violations += c.analyzeBlock(path, currentBlock, allLines, hasFuncs)
	}

	return violations, nil
}

func (c *CheckCommentsCommand) report(path string, line int, format string, args ...interface{}) {
	fmt.Printf("%s:%d: %s\n", path, line, fmt.Sprintf(format, args...))
}

func (c *CheckCommentsCommand) hasFunctions(lines []string) bool {
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "func ") || strings.Contains(trimmed, " func(") {
			return true
		}
	}
	return false
}

func (c *CheckCommentsCommand) isSpecialComment(content string) bool {
	t := strings.TrimSpace(content)
	return strings.Contains(content, "http://") ||
		strings.Contains(content, "https://") ||
		strings.Contains(content, "nolint") ||
		strings.Contains(content, "go:") ||
		strings.HasPrefix(t, "@") ||
		strings.HasPrefix(t, "swagger:")
}

func (c *CheckCommentsCommand) processLine(trimmedLine string, inBlockComment *bool) (bool, string, bool) {
	if *inBlockComment {
		return c.handleOngoingBlockComment(trimmedLine, inBlockComment)
	}

	if strings.HasPrefix(trimmedLine, "//") {
		return true, trimmedLine[2:], true
	}
	if strings.HasPrefix(trimmedLine, "/*") {
		return c.handleStartingBlockComment(trimmedLine, inBlockComment)
	}

	return c.findInlineComment(trimmedLine, inBlockComment)
}

func (c *CheckCommentsCommand) handleOngoingBlockComment(trimmedLine string, inBlockComment *bool) (bool, string, bool) {
	if idx := strings.Index(trimmedLine, "*/"); idx != -1 {
		*inBlockComment = false
		return true, trimmedLine[:idx], true
	}
	return true, trimmedLine, true
}

func (c *CheckCommentsCommand) handleStartingBlockComment(trimmedLine string, inBlockComment *bool) (bool, string, bool) {
	*inBlockComment = true
	content := trimmedLine[2:]
	if idxEnd := strings.Index(content, "*/"); idxEnd != -1 {
		*inBlockComment = false
		return true, content[:idxEnd], true
	}
	return true, content, true
}

func (c *CheckCommentsCommand) findInlineComment(trimmedLine string, inBlockComment *bool) (bool, string, bool) {
	inString := false
	for i := 0; i < len(trimmedLine)-1; i++ {
		char := trimmedLine[i]
		if char == '"' && (i == 0 || trimmedLine[i-1] != '\\') {
			inString = !inString
		}
		if inString {
			continue
		}

		if trimmedLine[i] == '/' && trimmedLine[i+1] == '/' {
			return true, trimmedLine[i+2:], false
		}
		if trimmedLine[i] == '/' && trimmedLine[i+1] == '*' {
			*inBlockComment = true
			content := trimmedLine[i+2:]
			if idxEnd := strings.Index(content, "*/"); idxEnd != -1 {
				*inBlockComment = false
				return true, content[:idxEnd], false
			}
			return true, content, false
		}
	}
	return false, "", false
}

func (c *CheckCommentsCommand) analyzeBlock(path string, block *commentBlock, allLines []string, hasFuncs bool) int {
	violations := 0

	// 1. Duplicate check
	for i := 1; i < len(block.lines); i++ {
		line := block.lines[i]
		if line != "" && len(line) > 3 && line == block.lines[i-1] {
			c.report(path, block.startLine+i, "duplicate consecutive comment: %q", line)
			violations++
		}
	}

	// 2. Multiline Style Check (>= 3 lines)
	if len(block.lines) >= 3 {
		docName := c.findDocName(block, allLines)
		if docName != "" {
			// It's a doc comment. Must start with docName.
			if !c.hasValidDocStart(block, docName) {
				c.report(path, block.startLine, "doc comment for %s must start with %q", docName, docName)
				violations++
			}
		} else if c.shouldFlagInternal(path, block, hasFuncs) {
			// Not a doc comment, and not in an excluded file/pattern
			c.report(path, block.startLine, "large internal comment (>= 3 lines) should be refactored into a function or nolinted")
			violations++
		}
	}

	return violations
}

func (c *CheckCommentsCommand) findDocName(block *commentBlock, allLines []string) string {
	nextIdx := block.startLine + len(block.lines) - 1
	if nextIdx < len(allLines) {
		nextTrimmed := strings.TrimSpace(allLines[nextIdx])
		if nextTrimmed == "" {
			return ""
		}
		return c.getDocName(nextTrimmed)
	}
	return ""
}

func (c *CheckCommentsCommand) hasValidDocStart(block *commentBlock, docName string) bool {
	for _, l := range block.lines {
		if t := strings.TrimSpace(l); t != "" {
			return strings.HasPrefix(t, docName)
		}
	}
	return true
}

func (c *CheckCommentsCommand) shouldFlagInternal(path string, block *commentBlock, hasFuncs bool) bool {
	if !hasFuncs {
		return false
	}
	if strings.HasSuffix(path, "constants.go") || strings.Contains(path, "/domain/") {
		return false
	}
	for _, l := range block.lines {
		if c.isDivider(l) || c.isSpecialComment(l) {
			return false
		}
	}
	return true
}

func (c *CheckCommentsCommand) isDivider(line string) bool {
	trimmed := strings.Trim(line, "=-*# ")
	return len(line) > 10 && trimmed == ""
}

func (c *CheckCommentsCommand) getDocName(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return ""
	}

	switch fields[0] {
	case "func":
		// Handle methods: func (r *Repo) Name()
		nameIdx := 1
		if strings.HasPrefix(fields[1], "(") {
			for i := 1; i < len(fields); i++ {
				if strings.Contains(fields[i], ")") {
					nameIdx = i + 1
					break
				}
			}
		}
		if nameIdx < len(fields) {
			return strings.Split(strings.Split(fields[nameIdx], "(")[0], "[")[0]
		}
	case "type", "var", "const":
		if fields[1] == "(" {
			return ""
		}
		return strings.Split(fields[1], "[")[0]
	case "package":
		return "Package"
	}
	return ""
}
