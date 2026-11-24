package controltower

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/checks"
	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/jobs"
	"github.com/ss497254/gloski/internal/logger"
)

const (
	highCPUPercent    = 85
	highMemoryPercent = 90
	maxIssueDetail    = 140
	maxJobDetail      = 120
	jobIssueLimit     = 10
)

type dashboardIssue struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Advice   string `json:"advice,omitempty"`
	Action   string `json:"action,omitempty"`
	DeviceID string `json:"device_id,omitempty"`

	// since is only used to sort oldest-first within a severity.
	since time.Time
}

// issueRank puts errors before warnings before informational notes. The UI
// shows the first items, so the most actionable ones must come first.
func issueRank(severity string) int {
	switch severity {
	case "error":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}

func sortIssues(issues []dashboardIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		if rankI, rankJ := issueRank(issues[i].Severity), issueRank(issues[j].Severity); rankI != rankJ {
			return rankI < rankJ
		}
		return issues[i].since.Before(issues[j].since)
	})
}

func (d *dashboardHandlers) handleDashboard(w http.ResponseWriter, r *http.Request) {
	d.touchStats()
	list, err := d.fleetSvc.List()
	if err != nil {
		httpapi.InternalError(w, "failed to load dashboard", err.Error())
		return
	}
	if list == nil {
		list = []fleet.Device{}
	}

	online := 0
	masterConnected := false
	issues := []dashboardIssue{}
	for i := range list {
		device := &list[i]
		if device.Master && d.agentService.Connected(device.ID) {
			masterConnected = true
		}
		if device.Status == fleet.StatusOnline {
			online++
		} else {
			since := device.LastSeen
			if since.IsZero() {
				since = device.CreatedAt
			}
			issues = append(issues, dashboardIssue{
				ID:       "offline-" + device.ID,
				Severity: "warning",
				Title:    "Device offline",
				Detail:   fmt.Sprintf("%s has not sent a heartbeat recently", device.Name),
				Advice:   "Open Diagnose to find the first broken link between the master and the agent.",
				Action:   "/devices/" + device.ID,
				DeviceID: device.ID,
				since:    since,
			})
		}
		if device.Stats != nil {
			if device.Stats.CPU.UsagePercent > highCPUPercent {
				issues = append(issues, dashboardIssue{
					ID:       "cpu-" + device.ID,
					Severity: "warning",
					Title:    "High CPU usage",
					Detail:   fmt.Sprintf("%s is at %.0f%% CPU", device.Name, device.Stats.CPU.UsagePercent),
					Advice:   "Open the device's processes to find what is using the CPU.",
					Action:   "/devices/" + device.ID,
					DeviceID: device.ID,
					since:    device.LastSeen,
				})
			}
			if device.Stats.Memory.UsedPercent > highMemoryPercent {
				issues = append(issues, dashboardIssue{
					ID:       "memory-" + device.ID,
					Severity: "warning",
					Title:    "High memory usage",
					Detail:   fmt.Sprintf("%s is at %.0f%% memory", device.Name, device.Stats.Memory.UsedPercent),
					Advice:   "Open the device's processes; a stuck task may need to be stopped.",
					Action:   "/devices/" + device.ID,
					DeviceID: device.ID,
					since:    device.LastSeen,
				})
			}
		}
	}

	if !masterConnected {
		issues = append(issues, dashboardIssue{
			ID:       "master-agent",
			Severity: "error",
			Title:    "Master agent not connected",
			Detail:   "The agent on the master device has no live gRPC session",
			Advice:   "The master's own agent carries every session. Check the master log or restart the master.",
			Action:   "/devices",
			since:    d.startedAt,
		})
	}

	deployments, err := d.fleetSvc.ListDeployments()
	if err != nil {
		logger.Warn("Dashboard could not load deployments: %v", err)
	} else {
		for _, deployment := range deployments {
			if deployment.Status == fleet.DeployFailed {
				action := "/devices"
				if deployment.DeviceID != "" {
					action = "/devices/" + deployment.DeviceID
				}
				issues = append(issues, dashboardIssue{
					ID:       "deploy-" + deployment.ID,
					Severity: "error",
					Title:    "Deployment failed",
					Detail:   fmt.Sprintf("%s: %s", deployment.Name, firstLine(deployment.Error, maxIssueDetail)),
					Advice:   "Open Diagnose for the exact SSH or install failure, then press Start or Trust new key.",
					Action:   action,
					DeviceID: deployment.DeviceID,
					since:    deployment.UpdatedAt,
				})
			}
		}
	}

	problemJobs, err := d.agentService.ProblemJobs(jobIssueLimit)
	if err != nil {
		logger.Warn("Dashboard could not load problem jobs: %v", err)
	} else {
		for _, job := range problemJobs {
			severity := "error"
			if job.Status == jobs.JobAbandoned {
				severity = "warning"
			}
			detail := strings.TrimSpace(job.Command)
			if job.Error != "" {
				detail += " — " + firstLine(job.Error, maxJobDetail)
			}
			advice := "Open the task log to see why it failed."
			if job.Status == jobs.JobAbandoned {
				advice = "The agent disappeared while this task was running. Diagnose the device, then run it again."
			}
			issues = append(issues, dashboardIssue{
				ID:       "job-" + job.ID,
				Severity: severity,
				Title:    "Task did not finish",
				Detail:   detail,
				Advice:   advice,
				Action:   "/devices/" + job.DeviceID + "/jobs",
				DeviceID: job.DeviceID,
				since:    job.CreatedAt,
			})
		}
	}

	recentJobs, err := d.agentService.RecentJobs(jobIssueLimit)
	if err != nil {
		logger.Warn("Dashboard could not load recent jobs: %v", err)
		recentJobs = []jobs.Job{}
	}

	matrixResults, err := d.checkStore().LatestForAllDevices(r.Context())
	if err != nil {
		logger.Warn("Dashboard could not load check results: %v", err)
		matrixResults = []checks.Result{}
		issues = append(issues, dashboardIssue{
			ID:       "check-results-unavailable",
			Severity: "warning",
			Title:    "Check results unavailable",
			Detail:   "The saved check results could not be read: " + firstLine(err.Error(), maxIssueDetail),
			Advice:   "The check matrix is empty until the master can read its results again.",
			since:    time.Now(),
		})
	}

	problemsByDevice, err := d.agentService.ProblemCountsByDevice(r.Context())
	if err != nil {
		logger.Warn("Dashboard could not count problem jobs: %v", err)
		problemsByDevice = map[string]int{}
	}

	matrix := buildCheckMatrix(list, matrixResults, problemsByDevice)
	sortIssues(issues)

	httpapi.SuccessWithMessage(w, map[string]interface{}{
		"devices":      len(list),
		"online":       online,
		"offline":      len(list) - online,
		"verdict":      matrixVerdict(matrix, issues),
		"issues":       issues,
		"check_matrix": matrix,
		"recent_jobs":  recentJobs,
		"generated_at": time.Now().UTC(),
	})
}

func firstLine(value string, max int) string {
	value = strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
	if len(value) > max {
		return value[:max] + "…"
	}
	return value
}
