// Builds the dashboard summary from master-owned state: devices, deployments, and jobs.

package master

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/agentgrpc"
	"github.com/ss497254/gloski/internal/api/handlers"
	"github.com/ss497254/gloski/internal/devices"
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
	DeviceID string `json:"device_id,omitempty"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	list, err := s.devices.List()
	if err != nil {
		handlers.InternalError(w, "failed to load dashboard", err.Error())
		return
	}
	if list == nil {
		list = []devices.Device{}
	}

	online := 0
	masterConnected := false
	issues := []dashboardIssue{}
	for i := range list {
		device := &list[i]
		if device.Master && s.agents.Connected(device.ID) {
			masterConnected = true
		}
		if device.Status == devices.StatusOnline {
			online++
		} else {
			issues = append(issues, dashboardIssue{
				ID:       "offline-" + device.ID,
				Severity: "warning",
				Title:    "Device offline",
				Detail:   fmt.Sprintf("%s has not sent a heartbeat recently", device.Name),
				DeviceID: device.ID,
			})
		}
		if device.Stats != nil {
			if device.Stats.CPU.UsagePercent > highCPUPercent {
				issues = append(issues, dashboardIssue{
					ID:       "cpu-" + device.ID,
					Severity: "warning",
					Title:    "High CPU usage",
					Detail:   fmt.Sprintf("%s is at %.0f%% CPU", device.Name, device.Stats.CPU.UsagePercent),
					DeviceID: device.ID,
				})
			}
			if device.Stats.Memory.UsedPercent > highMemoryPercent {
				issues = append(issues, dashboardIssue{
					ID:       "memory-" + device.ID,
					Severity: "warning",
					Title:    "High memory usage",
					Detail:   fmt.Sprintf("%s is at %.0f%% memory", device.Name, device.Stats.Memory.UsedPercent),
					DeviceID: device.ID,
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
		})
	}

	deployments, err := s.devices.ListDeployments()
	if err != nil {
		logger.Warn("Dashboard could not load deployments: %v", err)
	} else {
		for _, deployment := range deployments {
			if deployment.Status == devices.DeployFailed {
				issues = append(issues, dashboardIssue{
					ID:       "deploy-" + deployment.ID,
					Severity: "error",
					Title:    "Deployment failed",
					Detail:   fmt.Sprintf("%s: %s", deployment.Name, firstLine(deployment.Error, maxIssueDetail)),
					DeviceID: deployment.DeviceID,
				})
			}
		}
	}

	problemJobs, err := s.agents.ProblemJobs(jobIssueLimit)
	if err != nil {
		logger.Warn("Dashboard could not load problem jobs: %v", err)
	} else {
		for _, job := range problemJobs {
			severity := "error"
			if job.Status == agentgrpc.JobAbandoned {
				severity = "warning"
			}
			detail := strings.TrimSpace(job.Command)
			if job.Error != "" {
				detail += " — " + firstLine(job.Error, maxJobDetail)
			}
			issues = append(issues, dashboardIssue{
				ID:       "job-" + job.ID,
				Severity: severity,
				Title:    "Task did not finish",
				Detail:   detail,
				DeviceID: job.DeviceID,
			})
		}
	}

	recentJobs, err := s.agents.RecentJobs(jobIssueLimit)
	if err != nil {
		logger.Warn("Dashboard could not load recent jobs: %v", err)
		recentJobs = []agentgrpc.Job{}
	}

	handlers.SuccessWithMessage(w, map[string]interface{}{
		"devices":      len(list),
		"online":       online,
		"offline":      len(list) - online,
		"issues":       issues,
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
