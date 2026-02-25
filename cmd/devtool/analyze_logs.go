package main

import (
	"bufio"
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

	// userJobs[uid][job] -> count
	userJobs := make(map[string]map[string]int)
	// userNames[uid] -> username
	userNames := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Extract user_id and username mappings
		// Looking for lines containing both "user_id=" and "username="
		if strings.Contains(line, "user_id=") && strings.Contains(line, "username=") {
			parts := strings.Fields(line)
			var uid, uname string
			for _, p := range parts {
				if strings.HasPrefix(p, "user_id=") {
					uid = strings.TrimPrefix(p, "user_id=")
				}
				if strings.HasPrefix(p, "username=") {
					uname = strings.TrimPrefix(p, "username=")
				}
			}
			if uid != "" && uname != "" {
				userNames[uid] = uname
			}
		}

		// Count XP awards
		// Looking for 'msg="Awarded job XP"'
		if strings.Contains(line, `msg="Awarded job XP"`) {
			parts := strings.Fields(line)
			var uid, job string
			for _, p := range parts {
				if strings.HasPrefix(p, "user_id=") {
					uid = strings.TrimPrefix(p, "user_id=")
				}
				if strings.HasPrefix(p, "job=") {
					job = strings.TrimPrefix(p, "job=")
				}
			}
			if uid != "" && job != "" {
				if userJobs[uid] == nil {
					userJobs[uid] = make(map[string]int)
				}
				userJobs[uid][job]++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	PrintHeader("Log Analysis")

	// Prepare tabwriter
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Username\tScholar (Engage)\tExplorer (Search)\tFarmer (Harvest)")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	// Sort users by UID for deterministic output
	var uids []string
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
