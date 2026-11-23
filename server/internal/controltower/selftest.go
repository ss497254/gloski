package controltower

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ss497254/gloski/internal/agentdist"
	"github.com/ss497254/gloski/internal/database"
	"github.com/ss497254/gloski/internal/httpapi"
	"github.com/ss497254/gloski/internal/system"
)

// selfTestLine is one read-only observation about the master itself. A skip is
// a check that cannot apply yet, never a failure.
type selfTestLine struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Detail  string `json:"detail,omitempty"`
}

type selfTestResult struct {
	Status      string         `json:"status"`
	Lines       []selfTestLine `json:"lines"`
	GeneratedAt time.Time      `json:"generated_at"`
}

type runtimeSample struct {
	At         time.Time
	Goroutines int
	HeapMiB    uint64
}

const (
	selfTestMinFree   = 512 << 20 // below this, warn about the data directory
	selfTestLowFree   = 100 << 20 // below this, call it critical
	selfTestTrendSize = 3
)

func (t *Tower) handleSelfTest(w http.ResponseWriter, r *http.Request) {
	httpapi.SuccessWithMessage(w, t.selfTest(r.Context()))
}

// selfTest walks the master's own health. It never changes configuration and
// the only write is a probe file that is deleted in the same call.
func (t *Tower) selfTest(ctx context.Context) selfTestResult {
	result := selfTestResult{Lines: []selfTestLine{}, GeneratedAt: time.Now().UTC()}
	add := func(line selfTestLine) { result.Lines = append(result.Lines, line) }

	t.selfTestDatabase(ctx, add)
	t.selfTestDataDir(add)
	t.selfTestJoinToken(add)
	t.selfTestFreeSpace(add)
	add(selfTestLine{
		Key:     "backups",
		Title:   "Backups",
		Status:  diagSkipped,
		Summary: "Backups are not configured yet.",
		Detail:  "Nightly backups are a later roadmap step; there is nothing fresh to check.",
	})
	t.selfTestLocalAgent(add)
	t.selfTestBuilds(add)
	add(selfTestLine{
		Key:     "log",
		Title:   "Log",
		Status:  diagSkipped,
		Summary: "Logs go to stdout; the supervisor owns them.",
	})
	t.selfTestMaintenance(add)
	t.selfTestRuntime(add)

	result.Status = selfTestStatus(result.Lines)
	return result
}

// selfTestStatus folds the lines into one verdict. Skipped lines never count;
// a single critical beats any number of warnings.
func selfTestStatus(lines []selfTestLine) string {
	status := diagOK
	for _, line := range lines {
		if line.Status == diagCritical {
			return diagCritical
		}
		if line.Status == diagWarning {
			status = diagWarning
		}
	}
	return status
}

func (t *Tower) selfTestDatabase(ctx context.Context, add func(selfTestLine)) {
	db := t.db.DB()
	var quick string
	quickErr := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&quick)
	var version int
	versionErr := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)

	switch {
	case quickErr != nil:
		add(selfTestLine{
			Key: "database", Title: "Database", Status: diagCritical,
			Summary: "The master database could not be checked.",
			Detail:  quickErr.Error(),
		})
	case quick != "ok":
		add(selfTestLine{
			Key: "database", Title: "Database", Status: diagCritical,
			Summary: "SQLite reported a damaged database.",
			Detail:  quick,
		})
	case versionErr != nil:
		add(selfTestLine{
			Key: "database", Title: "Database", Status: diagWarning,
			Summary: "The migration version could not be read.",
			Detail:  versionErr.Error(),
		})
	case version < database.LatestVersion():
		add(selfTestLine{
			Key: "database", Title: "Database", Status: diagCritical,
			Summary: "The database schema is behind the master.",
			Detail:  fmt.Sprintf("schema version %d, master expects %d", version, database.LatestVersion()),
		})
	case version > database.LatestVersion():
		add(selfTestLine{
			Key: "database", Title: "Database", Status: diagWarning,
			Summary: "The database schema is newer than this master.",
			Detail:  fmt.Sprintf("schema version %d, this master expects %d; run an older master or migrate forward", version, database.LatestVersion()),
		})
	default:
		add(selfTestLine{
			Key: "database", Title: "Database", Status: diagOK,
			Summary: "The database passed quick_check and is on the latest schema.",
		})
	}

	path := t.cfg.MasterDBPath()
	dbFile, dbErr := os.Stat(path)
	walFile, walErr := os.Stat(path + "-wal")
	if dbErr != nil && walErr != nil {
		add(selfTestLine{
			Key: "size", Title: "Database size", Status: diagSkipped,
			Summary: "Database and WAL sizes are not available.",
		})
		return
	}
	size := fmt.Sprintf("database %s", mebibytes(dbFile, dbErr))
	if walErr == nil {
		size += fmt.Sprintf(", WAL %s", mebibytes(walFile, nil))
	}
	add(selfTestLine{
		Key: "size", Title: "Database size", Status: diagOK, Summary: size + ".",
	})
}

func (t *Tower) selfTestDataDir(add func(selfTestLine)) {
	probe, err := os.CreateTemp(t.cfg.DataDir, "selftest-*")
	if err != nil {
		add(selfTestLine{
			Key: "data-dir", Title: "Data directory", Status: diagCritical,
			Summary: "The master cannot write to its data directory.",
			Detail:  err.Error(),
		})
		return
	}
	name := probe.Name()
	_, writeErr := probe.WriteString("ok")
	closeErr := probe.Close()
	removeErr := os.Remove(name)
	if writeErr != nil || closeErr != nil || removeErr != nil {
		detail := firstError(writeErr, closeErr, removeErr)
		add(selfTestLine{
			Key: "data-dir", Title: "Data directory", Status: diagCritical,
			Summary: "The master cannot write and delete a probe file in its data directory.",
			Detail:  detail.Error(),
		})
		return
	}
	add(selfTestLine{
		Key: "data-dir", Title: "Data directory", Status: diagOK,
		Summary: "The master can write and delete files in its data directory.",
	})
}

func (t *Tower) selfTestJoinToken(add func(selfTestLine)) {
	path := filepath.Join(t.cfg.DataDir, "master-join-token")
	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		add(selfTestLine{
			Key: "join-token", Title: "Join token", Status: diagCritical,
			Summary: "The master join token is missing.",
			Detail:  "Agents cannot enroll until the master generates it on startup.",
		})
	case err != nil:
		add(selfTestLine{
			Key: "join-token", Title: "Join token", Status: diagWarning,
			Summary: "The master join token could not be read.",
			Detail:  err.Error(),
		})
	case info.IsDir():
		add(selfTestLine{
			Key: "join-token", Title: "Join token", Status: diagCritical,
			Summary: "The join token path is a directory.",
			Detail:  path,
		})
	case info.Mode().Perm() != 0600:
		add(selfTestLine{
			Key: "join-token", Title: "Join token", Status: diagWarning,
			Summary: fmt.Sprintf("The join token mode is %04o, not 0600.", info.Mode().Perm()),
			Detail:  path,
		})
	default:
		add(selfTestLine{
			Key: "join-token", Title: "Join token", Status: diagOK,
			Summary: "The join token exists with mode 0600.",
		})
	}
}

func (t *Tower) selfTestFreeSpace(add func(selfTestLine)) {
	var stat system.StatFS
	if err := system.Statfs(t.cfg.DataDir, &stat); err != nil {
		add(selfTestLine{
			Key: "free-space", Title: "Free space", Status: diagSkipped,
			Summary: "Free space on the data directory could not be read.",
			Detail:  err.Error(),
		})
		return
	}
	free := stat.Bfree * uint64(stat.Bsize)
	status := diagOK
	if free < selfTestLowFree {
		status = diagCritical
	} else if free < selfTestMinFree {
		status = diagWarning
	}
	add(selfTestLine{
		Key: "free-space", Title: "Free space", Status: status,
		Summary: fmt.Sprintf("The data directory has %s free.", mebibytesNumber(free)),
	})
}

func (t *Tower) selfTestLocalAgent(add func(selfTestLine)) {
	devices, err := t.fleetSvc.List()
	if err != nil {
		add(selfTestLine{
			Key: "local-agent", Title: "Master agent", Status: diagWarning,
			Summary: "The device list could not be read.",
			Detail:  err.Error(),
		})
		return
	}
	var masterID string
	for i := range devices {
		if devices[i].Master {
			masterID = devices[i].ID
			break
		}
	}
	if masterID == "" {
		add(selfTestLine{
			Key: "local-agent", Title: "Master agent", Status: diagCritical,
			Summary: "No master device has enrolled.",
		})
		return
	}
	if !t.agentService.Connected(masterID) {
		add(selfTestLine{
			Key: "local-agent", Title: "Master agent", Status: diagCritical,
			Summary: "The master's own agent has no live session.",
			Detail:  "Every device session goes through the master agent; check the master log.",
		})
		return
	}
	arch, hash, _ := t.agentService.Build(masterID)
	if arch == "" || hash == "" {
		add(selfTestLine{
			Key: "local-agent", Title: "Master agent", Status: diagWarning,
			Summary: "The local agent did not report its build.",
		})
		return
	}
	build := t.dist.Build(arch)
	if build == nil {
		add(selfTestLine{
			Key: "local-agent", Title: "Master agent", Status: diagWarning,
			Summary: fmt.Sprintf("The master has no agent build for linux/%s.", arch),
		})
		return
	}
	if hash != build.SHA256 {
		add(selfTestLine{
			Key: "local-agent", Title: "Master agent", Status: diagWarning,
			Summary: "The local agent runs an older build than the master.",
		})
		return
	}
	add(selfTestLine{
		Key: "local-agent", Title: "Master agent", Status: diagOK,
		Summary: "The local agent is connected and runs the master's build.",
	})
}

func (t *Tower) selfTestBuilds(add func(selfTestLine)) {
	available := make(map[string]bool)
	for _, arch := range t.dist.Builds() {
		available[arch] = true
	}
	missing := []string{}
	for _, arch := range agentdist.Arches {
		if !available[arch] {
			missing = append(missing, arch)
		}
	}
	if len(missing) > 0 {
		add(selfTestLine{
			Key: "builds", Title: "Agent builds", Status: diagWarning,
			Summary: "The master is missing agent builds: " + strings.Join(missing, ", ") + ".",
			Detail:  "Devices with those CPU architectures cannot be installed. Run 'make build' on the master.",
		})
		return
	}
	add(selfTestLine{
		Key: "builds", Title: "Agent builds", Status: diagOK,
		Summary: "Agent builds for amd64, arm64 and 32-bit ARM are present.",
	})
}

func (t *Tower) selfTestMaintenance(add func(selfTestLine)) {
	last, err := t.maintenanceState()
	switch {
	case last.IsZero():
		add(selfTestLine{
			Key: "maintenance", Title: "Maintenance", Status: diagSkipped,
			Summary: "The first maintenance run has not happened yet.",
		})
	case err != nil:
		add(selfTestLine{
			Key: "maintenance", Title: "Maintenance", Status: diagCritical,
			Summary: "The last maintenance run failed.",
			Detail:  err.Error(),
		})
	default:
		add(selfTestLine{
			Key: "maintenance", Title: "Maintenance", Status: diagOK,
			Summary: fmt.Sprintf("The last maintenance run succeeded %s ago.", time.Since(last).Round(time.Second)),
		})
	}
}

func (t *Tower) selfTestRuntime(add func(selfTestLine)) {
	sample := runtimeSample{At: time.Now().UTC(), Goroutines: runtime.NumGoroutine()}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	sample.HeapMiB = mem.HeapAlloc >> 20

	t.maintenanceMu.Lock()
	t.runtimeSamples = append(t.runtimeSamples, sample)
	if len(t.runtimeSamples) > selfTestTrendSize {
		t.runtimeSamples = append([]runtimeSample{}, t.runtimeSamples[len(t.runtimeSamples)-selfTestTrendSize:]...)
	}
	samples := append([]runtimeSample{}, t.runtimeSamples...)
	t.maintenanceMu.Unlock()

	detail := make([]string, 0, len(samples))
	for _, point := range samples {
		detail = append(detail, fmt.Sprintf("%s: %d goroutines, %d MiB heap", point.At.Format(time.RFC3339), point.Goroutines, point.HeapMiB))
	}
	detail = append(detail, trend(samples))
	add(selfTestLine{
		Key: "runtime", Title: "Runtime", Status: diagOK,
		Summary: fmt.Sprintf("%d goroutines, %d MiB heap.", sample.Goroutines, sample.HeapMiB),
		Detail:  strings.Join(detail, "\n"),
	})
}

// trend needs three points before it says anything, so one stray spike does
// not look like a problem.
func trend(samples []runtimeSample) string {
	if len(samples) < selfTestTrendSize {
		return fmt.Sprintf("Trend: need %d runs, have %d.", selfTestTrendSize, len(samples))
	}
	first, last := samples[0], samples[len(samples)-1]
	switch {
	case last.Goroutines > first.Goroutines && last.HeapMiB > first.HeapMiB:
		return "Trend: goroutines and heap are rising across the last three runs."
	case last.Goroutines < first.Goroutines && last.HeapMiB < first.HeapMiB:
		return "Trend: goroutines and heap are falling across the last three runs."
	default:
		return "Trend: no combined rise or fall across the last three runs."
	}
}

func mebibytes(info os.FileInfo, err error) string {
	if err != nil {
		return "unavailable"
	}
	return mebibytesNumber(uint64(info.Size()))
}

func mebibytesNumber(bytes uint64) string {
	return fmt.Sprintf("%.1f MiB", float64(bytes)/(1<<20))
}

func firstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
