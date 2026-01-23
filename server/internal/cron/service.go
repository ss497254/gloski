// Package cron provides cron job management functionality.
package cron

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

var (
	ErrCronNotFound = errors.New("cron not available")
)

// Service provides cron job management operations.
type Service struct {
	available bool
}

// NewService creates a new cron service.
func NewService() (*Service, error) {
	s := &Service{}

	// Check if crontab command exists
	if _, err := exec.LookPath("crontab"); err != nil {
		return nil, ErrCronNotFound
	}

	s.available = true
	return s, nil
}

// CronJob represents a cron job entry.
type CronJob struct {
	Schedule    string `json:"schedule"`
	Command     string `json:"command"`
	User        string `json:"user,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source"` // "user", "system", "etc"
}

// CronSchedule represents parsed schedule components.
type CronSchedule struct {
	Minute     string `json:"minute"`
	Hour       string `json:"hour"`
	DayOfMonth string `json:"day_of_month"`
	Month      string `json:"month"`
	DayOfWeek  string `json:"day_of_week"`
}

// ParseSchedule parses a cron schedule string into components.
func ParseSchedule(schedule string) (*CronSchedule, error) {
	parts := strings.Fields(schedule)
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid schedule: expected 5 fields, got %d", len(parts))
	}

	return &CronSchedule{
		Minute:     parts[0],
		Hour:       parts[1],
		DayOfMonth: parts[2],
		Month:      parts[3],
		DayOfWeek:  parts[4],
	}, nil
}

// ListUserJobs returns cron jobs for the current user.
func (s *Service) ListUserJobs() ([]CronJob, error) {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		// Empty crontab returns error, that's ok
		if strings.Contains(err.Error(), "no crontab") {
			return []CronJob{}, nil
		}
		return nil, err
	}

	currentUser, _ := user.Current()
	username := ""
	if currentUser != nil {
		username = currentUser.Username
	}

	return s.parseCrontab(string(output), username, "user"), nil
}

// ListSystemJobs returns system-wide cron jobs from /etc/cron.d and /etc/crontab.
func (s *Service) ListSystemJobs() ([]CronJob, error) {
	var jobs []CronJob

	// Parse /etc/crontab
	if content, err := os.ReadFile("/etc/crontab"); err == nil {
		jobs = append(jobs, s.parseSystemCrontab(string(content))...)
	}

	// Parse /etc/cron.d/*
	cronDPath := "/etc/cron.d"
	entries, err := os.ReadDir(cronDPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			filePath := filepath.Join(cronDPath, entry.Name())
			if content, err := os.ReadFile(filePath); err == nil {
				cronJobs := s.parseSystemCrontab(string(content))
				for i := range cronJobs {
					cronJobs[i].Source = "cron.d/" + entry.Name()
				}
				jobs = append(jobs, cronJobs...)
			}
		}
	}

	return jobs, nil
}

// ListAllJobs returns all cron jobs (user and system).
func (s *Service) ListAllJobs() ([]CronJob, error) {
	var allJobs []CronJob

	userJobs, err := s.ListUserJobs()
	if err == nil {
		allJobs = append(allJobs, userJobs...)
	}

	systemJobs, err := s.ListSystemJobs()
	if err == nil {
		allJobs = append(allJobs, systemJobs...)
	}

	return allJobs, nil
}

// parseCrontab parses a user crontab content.
func (s *Service) parseCrontab(content, username, source string) []CronJob {
	var jobs []CronJob
	var description string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			description = ""
			continue
		}

		// Check for comments (potential descriptions)
		if strings.HasPrefix(line, "#") {
			// Check if it's a disabled job
			trimmed := strings.TrimPrefix(line, "#")
			trimmed = strings.TrimSpace(trimmed)
			if isValidCronLine(trimmed) {
				job := parseCronLine(trimmed)
				if job != nil {
					job.User = username
					job.Source = source
					job.Enabled = false
					job.Description = description
					jobs = append(jobs, *job)
				}
			} else {
				// It's a comment, use as description for next job
				description = strings.TrimSpace(trimmed)
			}
			continue
		}

		// Parse active cron line
		if isValidCronLine(line) {
			job := parseCronLine(line)
			if job != nil {
				job.User = username
				job.Source = source
				job.Enabled = true
				job.Description = description
				jobs = append(jobs, *job)
			}
		}
		description = ""
	}

	return jobs
}

// parseSystemCrontab parses a system crontab content (includes user field).
func (s *Service) parseSystemCrontab(content string) []CronJob {
	var jobs []CronJob
	var description string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			description = ""
			continue
		}

		// Skip environment variable declarations
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
			if !isValidCronLine(line) {
				continue
			}
		}

		if strings.HasPrefix(line, "#") {
			trimmed := strings.TrimPrefix(line, "#")
			description = strings.TrimSpace(trimmed)
			continue
		}

		// System crontab has 6 fields + command (minute hour day month dow user command)
		fields := strings.Fields(line)
		if len(fields) >= 7 {
			job := &CronJob{
				Schedule:    strings.Join(fields[:5], " "),
				User:        fields[5],
				Command:     strings.Join(fields[6:], " "),
				Source:      "system",
				Enabled:     true,
				Description: description,
			}
			jobs = append(jobs, *job)
		}
		description = ""
	}

	return jobs
}

// isValidCronLine checks if a line looks like a valid cron entry.
func isValidCronLine(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return false
	}

	// Check if first 5 fields look like cron schedule
	scheduleChars := "0123456789*,-/"
	for i := 0; i < 5; i++ {
		field := fields[i]
		// Allow @reboot, @hourly, etc.
		if strings.HasPrefix(field, "@") {
			return true
		}
		// Check each character
		valid := true
		for _, c := range field {
			if !strings.ContainsRune(scheduleChars, c) {
				valid = false
				break
			}
		}
		if !valid {
			return false
		}
	}

	return true
}

// parseCronLine parses a cron line into a CronJob (user crontab format).
func parseCronLine(line string) *CronJob {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return nil
	}

	// Handle special schedules like @reboot, @hourly
	if strings.HasPrefix(fields[0], "@") {
		return &CronJob{
			Schedule: fields[0],
			Command:  strings.Join(fields[1:], " "),
		}
	}

	return &CronJob{
		Schedule: strings.Join(fields[:5], " "),
		Command:  strings.Join(fields[5:], " "),
	}
}

// AddJob adds a new cron job for the current user.
func (s *Service) AddJob(schedule, command string) error {
	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, _ := cmd.Output() // Ignore error for empty crontab

	// Append new job
	newCrontab := string(output)
	if !strings.HasSuffix(newCrontab, "\n") && newCrontab != "" {
		newCrontab += "\n"
	}
	newCrontab += fmt.Sprintf("%s %s\n", schedule, command)

	// Write new crontab
	return s.writeCrontab(newCrontab)
}

// RemoveJob removes a cron job for the current user.
func (s *Service) RemoveJob(schedule, command string) error {
	// Get current crontab
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Remove matching line
	var newLines []string
	targetLine := fmt.Sprintf("%s %s", schedule, command)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != targetLine {
			newLines = append(newLines, line)
		}
	}

	// Write new crontab
	return s.writeCrontab(strings.Join(newLines, "\n") + "\n")
}

func (s *Service) writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}
