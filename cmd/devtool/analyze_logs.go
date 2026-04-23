package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
)

type AnalyzeLogsCommand struct{}

func (c *AnalyzeLogsCommand) Name() string {
	return "analyze-logs"
}

func (c *AnalyzeLogsCommand) Description() string {
	return "Analyze logs to count job XP awards"
}

func (c *AnalyzeLogsCommand) Run(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: devtool analyze-logs <log-file>")
	}

	logFile := args[0]
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	userJobs, userNames, err := c.scanLogFile(file)
	if err != nil {
		return err
	}

	PrintHeader("Log Analysis")

	// Prepare tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Username\tScholar (Engage)\tExplorer (Search)\tFarmer (Harvest)")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	// Sort users by UID for deterministic output
	uids := make([]string, 0, len(userJobs))
	for uid := range userJobs {
		uids = append(uids, uid)
	}
	sort.Strings(uids)

	for _, uid := range uids {
		uname := userNames[uid]
		if uname == "" {
			uname = uid
		}
		jobs := userJobs[uid]
		scholar := jobs["job_scholar"]
		explorer := jobs["job_explorer"]
		farmer := jobs["job_farmer"]
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\n", uname, scholar, explorer, farmer)
	}
	w.Flush()

	PrintSuccess("Finished.")

	return nil
}

type logEntry struct {
	Msg      string `json:"msg"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Job      string `json:"job"`
}

func (c *AnalyzeLogsCommand) scanLogFile(file *os.File) (userJobs map[string]map[string]int, userNames map[string]string, err error) {
	// userJobs[uid][job] -> count
	userJobs = make(map[string]map[string]int)
	// userNames[uid] -> username
	userNames = make(map[string]string)

	scanner := newScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		msg, uid, uname, job := c.parseLogLine(line)

		if uid != "" && uname != "" {
			userNames[uid] = uname
		}

		if msg == "Awarded job XP" && uid != "" && job != "" {
			if userJobs[uid] == nil {
				userJobs[uid] = make(map[string]int)
			}
			userJobs[uid][job]++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading log file: %w", err)
	}

	return userJobs, userNames, nil
}

func (c *AnalyzeLogsCommand) parseLogLine(line string) (msg, uid, uname, job string) {
	if strings.HasPrefix(line, "{") {
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			return entry.Msg, entry.UserID, entry.Username, entry.Job
		}
	} else {
		msg = c.extractLogfmtValue(line, "msg")
		uid = c.extractLogfmtValue(line, "user_id")
		uname = c.extractLogfmtValue(line, "username")
		job = c.extractLogfmtValue(line, "job")
	}
	return msg, uid, uname, job
}

func (c *AnalyzeLogsCommand) extractLogfmtValue(line, key string) string {
	prefix := key + "="
	idx := strings.Index(line, prefix)
	if idx == -1 {
		return ""
	}

	start := idx + len(prefix)
	if start >= len(line) {
		return ""
	}

	if line[start] == '"' {
		// Quoted string
		end := strings.Index(line[start+1:], `"`)
		if end == -1 {
			return line[start+1:]
		}
		return line[start+1 : start+1+end]
	}

	// Unquoted string
	end := strings.IndexByte(line[start:], ' ')
	if end == -1 {
		return line[start:]
	}
	return line[start : start+end]
}
