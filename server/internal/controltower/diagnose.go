package controltower

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ss497254/gloski/internal/fleet"
	"github.com/ss497254/gloski/internal/httpapi"
)

// Diagnostic states. Skipped is neutral: a check that does not apply, or one
// that was not needed because an earlier step answered the question, must not
// look like a failure.
const (
	diagOK       = "ok"
	diagWarning  = "warning"
	diagCritical = "critical"
	diagSkipped  = "skipped"
)

const (
	diagnoseDialTimeout = 5 * time.Second
	diagnoseSSHTimeout  = 25 * time.Second
	diagnoseTailnetWait = 5 * time.Second
)

// diagnosisStep is one link in the connection chain. Summary is the factual
// one-liner; Detail holds the raw error when there is one; Advice is what a
// person can do about it.
type diagnosisStep struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
	Advice  string `json:"advice,omitempty"`
}

// diagnosis is the result of walking a device's connection chain. Steps are in
// cause order; the first critical step is the headline.
type diagnosis struct {
	DeviceID    string          `json:"device_id"`
	Device      string          `json:"device"`
	Status      string          `json:"status"`
	Headline    string          `json:"headline"`
	Advice      string          `json:"advice,omitempty"`
	Steps       []diagnosisStep `json:"steps"`
	GeneratedAt time.Time       `json:"generated_at"`
}

func (d *diagnosis) add(step diagnosisStep) {
	d.Steps = append(d.Steps, step)
}

// finish picks the overall status and the headline. Critical beats warning;
// skipped steps never become a verdict on their own.
func (d *diagnosis) finish() {
	for _, wanted := range []string{diagCritical, diagWarning} {
		for _, step := range d.Steps {
			if step.Status == wanted {
				d.Status = wanted
				d.Headline = step.Summary
				d.Advice = step.Advice
				return
			}
		}
	}
	d.Status = diagOK
	d.Headline = "The agent is connected and runs the master's build."
}

func (d *deployHandlers) handleDiagnoseDevice(w http.ResponseWriter, r *http.Request) {
	device, err := d.fleetSvc.Get(r.PathValue("id"))
	if err != nil {
		httpapi.NotFound(w, "device not found")
		return
	}
	httpapi.SuccessWithMessage(w, d.diagnoseDevice(r.Context(), device))
}

// diagnoseDevice walks the chain from the master outwards and stops at the
// first failure. When the agent is connected the SSH links are not needed, so
// they are left out instead of being reported as unknown.
func (d *deployHandlers) diagnoseDevice(ctx context.Context, device *fleet.Device) diagnosis {
	result := diagnosis{
		DeviceID:    device.ID,
		Device:      device.Name,
		Steps:       []diagnosisStep{},
		GeneratedAt: time.Now().UTC(),
	}
	deployment := device.Deploy
	connected := d.agentService.Connected(device.ID)

	if deployment == nil {
		result.add(diagnosisStep{
			Key:     "deployment",
			Title:   "Deployment record",
			Status:  diagSkipped,
			Summary: "This device was not deployed by the master.",
			Detail:  "The master has no SSH details, so it cannot repair this agent itself.",
			Advice:  "Redeploy the device from the dashboard so the master can manage its agent.",
		})
	} else {
		step := diagnosisStep{
			Key:     "deployment",
			Title:   "Deployment record",
			Status:  diagOK,
			Summary: fmt.Sprintf("The master installs this agent over SSH at %s:%d.", deployment.Host, sshPort(deployment)),
		}
		if deployment.Error != "" {
			step.Detail = "Last recorded error: " + firstLine(deployment.Error, 500)
		}
		result.add(step)
	}

	if deployment != nil && deployment.Status == fleet.DeployStopped {
		result.add(diagnosisStep{
			Key:     "agent",
			Title:   "Agent session",
			Status:  diagWarning,
			Summary: "The agent is stopped on purpose.",
			Advice:  "Press Start agent to run it again.",
		})
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagSkipped,
			Summary: "Not checked: the agent is stopped.",
		})
		result.finish()
		return result
	}

	if connected {
		result.add(diagnosisStep{
			Key:     "agent",
			Title:   "Agent session",
			Status:  diagOK,
			Summary: "The agent has a live session with the master.",
		})
		d.diagnoseBuild(device, &result)
		result.finish()
		return result
	}

	if deployment != nil {
		if d.diagnoseTailscale(ctx, deployment, &result) {
			d.diagnoseSSH(ctx, deployment, &result)
		} else {
			skipSSHChain(&result, "the device is offline in the tailnet")
		}
	}

	// Everything above was healthy, or could not be checked, yet the agent is
	// not connected. This step names the remaining problem and what to do.
	summary := "The agent has no live session with the master."
	detail := fmt.Sprintf("The master listens for agents at %s.", d.agentGRPCURL())
	if !device.LastSeen.IsZero() {
		detail += fmt.Sprintf(" The device was last seen %s.", time.Since(device.LastSeen).Round(time.Second))
	}
	advice := fmt.Sprintf("Check that %s can reach %s, then press Start agent.", device.Name, d.agentGRPCURL())
	if deployment == nil {
		advice = "Redeploy the device from the dashboard so the master can install and start its agent."
	}
	if rejection := d.agentService.Rejection(device.Name); rejection != "" {
		summary = "The master refused the agent's connection."
		detail = "The master turned the agent away: " + rejection + "."
		advice = "Press Start agent so the master installs a compatible build."
	}
	result.add(diagnosisStep{
		Key:     "agent",
		Title:   "Agent session",
		Status:  diagCritical,
		Summary: summary,
		Detail:  detail,
		Advice:  advice,
	})
	result.add(diagnosisStep{
		Key:     "build",
		Title:   "Agent build",
		Status:  diagSkipped,
		Summary: "Not checked: the agent is not connected.",
	})
	result.finish()
	return result
}

// diagnoseBuild compares the connected agent's binary with the one the master
// would install. An old build is a warning, not a failure: the agent works, and
// the lifecycle will replace it.
func (d *deployHandlers) diagnoseBuild(device *fleet.Device, result *diagnosis) {
	arch, hash, connected := d.agentService.Build(device.ID)
	if !connected {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagSkipped,
			Summary: "Not checked: the agent is not connected.",
		})
		return
	}
	if arch == "" || hash == "" {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagWarning,
			Summary: "The agent did not report which build it runs.",
			Advice:  "Press Restart agent to reinstall it.",
		})
		return
	}
	build := d.dist.Build(arch)
	if build == nil {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagWarning,
			Summary: fmt.Sprintf("The master has no agent build for linux/%s.", arch),
			Advice:  "Run 'make build' on the master.",
		})
		return
	}
	if hash != build.SHA256 {
		result.add(diagnosisStep{
			Key:     "build",
			Title:   "Agent build",
			Status:  diagWarning,
			Summary: "The agent runs an older build than the master.",
			Advice:  "The master replaces it automatically when the device has no running jobs; press Restart agent to do it now.",
		})
		return
	}
	result.add(diagnosisStep{
		Key:     "build",
		Title:   "Agent build",
		Status:  diagOK,
		Summary: "The agent runs the master's build.",
	})
}
