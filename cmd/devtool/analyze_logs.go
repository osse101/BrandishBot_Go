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

func (c *AnalyzeLogsCommand) scanLogFile(file *os.File) (userJobs map[string]map[string]int, userNames map[string]string, err error) {
	// userJobs[uid][job] -> count
	userJobs = make(map[string]map[string]int)
	// userNames[uid] -> username
	userNames = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if uid, uname, ok := c.extractUserInfo(line); ok {
			userNames[uid] = uname
		}

		if uid, job, ok := c.extractXPAward(line); ok {
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

func (c *AnalyzeLogsCommand) extractUserInfo(line string) (uid, uname string, ok bool) {
	if !strings.Contains(line, "user_id=") || !strings.Contains(line, "username=") {
		return "", "", false
	}

	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.HasPrefix(p, "user_id=") {
			uid = strings.TrimPrefix(p, "user_id=")
		}
		if strings.HasPrefix(p, "username=") {
			uname = strings.TrimPrefix(p, "username=")
		}
	}

	if uid != "" && uname != "" {
		return uid, uname, true
	}
	return "", "", false
}

func (c *AnalyzeLogsCommand) extractXPAward(line string) (uid, job string, ok bool) {
	if !strings.Contains(line, `msg="Awarded job XP"`) {
		return "", "", false
	}

	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.HasPrefix(p, "user_id=") {
			uid = strings.TrimPrefix(p, "user_id=")
		}
		if strings.HasPrefix(p, "job=") {
			job = strings.TrimPrefix(p, "job=")
		}
	}

	if uid != "" && job != "" {
		return uid, job, true
	}
	return "", "", false
}
