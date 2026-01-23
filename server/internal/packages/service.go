// Package packages provides system package management functionality.
package packages

import (
	"bufio"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var (
	ErrNoPackageManager = errors.New("no supported package manager found")
)

// ManagerType represents the type of package manager.
type ManagerType string

const (
	ManagerAPT ManagerType = "apt"
	ManagerDNF ManagerType = "dnf"
	ManagerYUM ManagerType = "yum"
	ManagerAPK ManagerType = "apk"
)

// Service provides package management operations.
type Service struct {
	manager ManagerType
}

// NewService creates a new package service.
func NewService() (*Service, error) {
	if runtime.GOOS != "linux" {
		return nil, ErrNoPackageManager
	}

	s := &Service{}
	if err := s.detectManager(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Service) detectManager() error {
	managers := []struct {
		cmd string
		typ ManagerType
	}{
		{"apt", ManagerAPT},
		{"dnf", ManagerDNF},
		{"yum", ManagerYUM},
		{"apk", ManagerAPK},
	}

	for _, m := range managers {
		if _, err := exec.LookPath(m.cmd); err == nil {
			s.manager = m.typ
			return nil
		}
	}

	return ErrNoPackageManager
}

// Manager returns the detected package manager type.
func (s *Service) Manager() ManagerType {
	return s.manager
}

// Package represents an installed or available package.
type Package struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Architecture string `json:"architecture,omitempty"`
	Description  string `json:"description,omitempty"`
	Status       string `json:"status,omitempty"`
}

// UpgradeInfo contains information about available upgrades.
type UpgradeInfo struct {
	UpgradableCount int       `json:"upgradable_count"`
	SecurityCount   int       `json:"security_count"`
	Packages        []Package `json:"packages"`
}

// ListInstalled returns a list of installed packages.
func (s *Service) ListInstalled(limit int) ([]Package, error) {
	switch s.manager {
	case ManagerAPT:
		return s.listInstalledAPT(limit)
	case ManagerDNF, ManagerYUM:
		return s.listInstalledDNF(limit)
	case ManagerAPK:
		return s.listInstalledAPK(limit)
	default:
		return nil, ErrNoPackageManager
	}
}

func (s *Service) listInstalledAPT(limit int) ([]Package, error) {
	cmd := exec.Command("dpkg-query", "-W", "-f=${Package}\t${Version}\t${Architecture}\t${Status}\n")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) >= 4 && strings.Contains(parts[3], "installed") {
			packages = append(packages, Package{
				Name:         parts[0],
				Version:      parts[1],
				Architecture: parts[2],
				Status:       "installed",
			})
			if limit > 0 && len(packages) >= limit {
				break
			}
		}
	}

	return packages, nil
}

func (s *Service) listInstalledDNF(limit int) ([]Package, error) {
	cmd := exec.Command("rpm", "-qa", "--queryformat", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) >= 3 {
			packages = append(packages, Package{
				Name:         parts[0],
				Version:      parts[1],
				Architecture: parts[2],
				Status:       "installed",
			})
			if limit > 0 && len(packages) >= limit {
				break
			}
		}
	}

	return packages, nil
}

func (s *Service) listInstalledAPK(limit int) ([]Package, error) {
	cmd := exec.Command("apk", "list", "--installed")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: package-name-version arch {origin} (license) [flags]
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			nameVer := parts[0]
			arch := parts[1]

			// Split name and version (last hyphen separates them)
			lastHyphen := strings.LastIndex(nameVer, "-")
			name := nameVer
			version := ""
			if lastHyphen > 0 {
				// Find the version part (after second-to-last hyphen usually)
				name = nameVer[:lastHyphen]
				version = nameVer[lastHyphen+1:]
			}

			packages = append(packages, Package{
				Name:         name,
				Version:      version,
				Architecture: arch,
				Status:       "installed",
			})
			if limit > 0 && len(packages) >= limit {
				break
			}
		}
	}

	return packages, nil
}

// CheckUpgrades checks for available package upgrades.
func (s *Service) CheckUpgrades() (*UpgradeInfo, error) {
	switch s.manager {
	case ManagerAPT:
		return s.checkUpgradesAPT()
	case ManagerDNF, ManagerYUM:
		return s.checkUpgradesDNF()
	case ManagerAPK:
		return s.checkUpgradesAPK()
	default:
		return nil, ErrNoPackageManager
	}
}

func (s *Service) checkUpgradesAPT() (*UpgradeInfo, error) {
	// First update package lists
	// exec.Command("apt", "update").Run() // Skip for now, requires root

	cmd := exec.Command("apt", "list", "--upgradable")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := &UpgradeInfo{
		Packages: []Package{},
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Listing...") {
			continue
		}

		// Format: package/release version arch [upgradable from: old-version]
		parts := strings.Split(line, " ")
		if len(parts) >= 2 {
			namePart := strings.Split(parts[0], "/")
			info.Packages = append(info.Packages, Package{
				Name:    namePart[0],
				Version: parts[1],
			})
		}
	}

	info.UpgradableCount = len(info.Packages)
	return info, nil
}

func (s *Service) checkUpgradesDNF() (*UpgradeInfo, error) {
	cmd := exec.Command("dnf", "check-update", "-q")
	output, _ := cmd.Output() // dnf returns exit code 100 if updates available

	info := &UpgradeInfo{
		Packages: []Package{},
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			info.Packages = append(info.Packages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}

	info.UpgradableCount = len(info.Packages)
	return info, nil
}

func (s *Service) checkUpgradesAPK() (*UpgradeInfo, error) {
	cmd := exec.Command("apk", "version", "-v", "-l", "<")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := &UpgradeInfo{
		Packages: []Package{},
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			info.Packages = append(info.Packages, Package{
				Name: parts[0],
			})
		}
	}

	info.UpgradableCount = len(info.Packages)
	return info, nil
}

// Search searches for packages by name.
func (s *Service) Search(query string, limit int) ([]Package, error) {
	switch s.manager {
	case ManagerAPT:
		return s.searchAPT(query, limit)
	case ManagerDNF, ManagerYUM:
		return s.searchDNF(query, limit)
	case ManagerAPK:
		return s.searchAPK(query, limit)
	default:
		return nil, ErrNoPackageManager
	}
}

func (s *Service) searchAPT(query string, limit int) ([]Package, error) {
	cmd := exec.Command("apt-cache", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), " - ", 2)
		if len(parts) >= 1 {
			pkg := Package{Name: parts[0]}
			if len(parts) >= 2 {
				pkg.Description = parts[1]
			}
			packages = append(packages, pkg)
			if limit > 0 && len(packages) >= limit {
				break
			}
		}
	}

	return packages, nil
}

func (s *Service) searchDNF(query string, limit int) ([]Package, error) {
	cmd := exec.Command("dnf", "search", "-q", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "=") || line == "" {
			continue
		}
		parts := strings.SplitN(line, " : ", 2)
		if len(parts) >= 1 {
			nameParts := strings.Split(parts[0], ".")
			pkg := Package{Name: nameParts[0]}
			if len(parts) >= 2 {
				pkg.Description = parts[1]
			}
			packages = append(packages, pkg)
			if limit > 0 && len(packages) >= limit {
				break
			}
		}
	}

	return packages, nil
}

func (s *Service) searchAPK(query string, limit int) ([]Package, error) {
	cmd := exec.Command("apk", "search", "-d", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		packages = append(packages, Package{Name: line})
		if limit > 0 && len(packages) >= limit {
			break
		}
	}

	return packages, nil
}

// GetPackageInfo returns detailed information about a package.
func (s *Service) GetPackageInfo(name string) (*Package, error) {
	switch s.manager {
	case ManagerAPT:
		return s.getPackageInfoAPT(name)
	case ManagerDNF, ManagerYUM:
		return s.getPackageInfoDNF(name)
	default:
		return nil, fmt.Errorf("package info not supported for %s", s.manager)
	}
}

func (s *Service) getPackageInfoAPT(name string) (*Package, error) {
	cmd := exec.Command("apt-cache", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pkg := &Package{Name: name}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Version:") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Architecture:") {
			pkg.Architecture = strings.TrimSpace(strings.TrimPrefix(line, "Architecture:"))
		} else if strings.HasPrefix(line, "Description:") {
			pkg.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	return pkg, nil
}

func (s *Service) getPackageInfoDNF(name string) (*Package, error) {
	cmd := exec.Command("dnf", "info", "-q", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pkg := &Package{Name: name}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Version") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				pkg.Version = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Architecture") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				pkg.Architecture = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Summary") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				pkg.Description = strings.TrimSpace(parts[1])
			}
		}
	}

	return pkg, nil
}
