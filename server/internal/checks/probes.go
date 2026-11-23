// Package checks also owns the native probes run by an agent. A probe returns
// a checks.Result directly, so its verdict lands in the same bounded history
// as shell-based checks and in the fleet matrix.
package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ProbeKind selects the native probe implementation.
type ProbeKind string

const (
	ProbeTCP  ProbeKind = "tcp"  // TCP connect
	ProbeHTTP ProbeKind = "http" // HTTP status code
	ProbeDNS  ProbeKind = "dns"  // DNS address lookup
	ProbeTLS  ProbeKind = "tls"  // TLS handshake and leaf expiry
)

const (
	// DefaultProbeTimeout applies when a definition has no timeout.
	DefaultProbeTimeout = 5 * time.Second
	// DefaultProbeInterval applies when a definition has no interval. The
	// master uses it when scheduling, not just the agent when running.
	DefaultProbeInterval = 5 * time.Minute

	tlsWarningWindow = 30 * 24 * time.Hour
)

// Definition is one native probe. Definitions are supplied by the master; the
// agent only executes what it is asked to. StableID ignores Interval.
type Definition struct {
	Kind     ProbeKind
	Target   string
	Accepted []int
	Timeout  time.Duration
	Interval time.Duration
	// SkipVerify is for devices with internal, self-signed certificates. It
	// is deliberately excluded from StableID: changing it updates the same
	// check rather than creating a new series.
	SkipVerify bool
}

// ID returns the stable check ID for this definition.
func (d Definition) ID() string { return StableID(d.Kind, d.Target, d.Accepted) }

// EffectiveInterval returns the configured interval or DefaultProbeInterval.
func (d Definition) EffectiveInterval() time.Duration {
	return d.interval()
}

func (d Definition) timeout() time.Duration {
	if d.Timeout > 0 {
		return d.Timeout
	}
	return DefaultProbeTimeout
}

func (d Definition) interval() time.Duration {
	if d.Interval > 0 {
		return d.Interval
	}
	return DefaultProbeInterval
}

// Run executes one probe and returns a result ready for the check store. A
// failure is critical; a TLS certificate nearing expiry is a warning. Nothing
// is stored here — the master decides when to save.
func (d Definition) Run(ctx context.Context, deviceID string) Result {
	result := Result{
		CheckID:  d.ID(),
		DeviceID: deviceID,
		RanAt:    time.Now().UTC(),
		Interval: d.interval(),
		Status:   StatusOK,
	}
	ctx, cancel := context.WithTimeout(ctx, d.timeout())
	defer cancel()

	started := time.Now()
	var detail []Detail
	switch d.Kind {
	case ProbeTCP:
		detail = probeTCP(ctx, d.Target)
	case ProbeHTTP:
		detail = probeHTTP(ctx, d.Target, d.Accepted)
	case ProbeDNS:
		detail = probeDNS(ctx, d.Target)
	case ProbeTLS:
		detail = probeTLS(ctx, d.Target, d.SkipVerify)
	default:
		result.Status = StatusUnknown
		result.Summary = "unknown probe kind " + string(d.Kind)
		result.DurationMS = time.Since(started).Milliseconds()
		return result
	}
	result.DurationMS = time.Since(started).Milliseconds()
	result.Details = detail
	if len(detail) > 0 {
		result.Summary = detail[0].Value
	}
	for _, line := range detail {
		if line.Key == "status" {
			switch line.Value {
			case "warning":
				result.Status = StatusWarning
			case "critical":
				result.Status = StatusCritical
			case "skipped":
				result.Status = StatusSkipped
			}
			break
		}
	}
	if result.Status == StatusOK {
		result.Summary = fmt.Sprintf("%s in %dms", result.Summary, result.DurationMS)
	}
	result.RefreshStale(time.Now())
	return result
}

func failure(format string, args ...interface{}) []Detail {
	return []Detail{{Key: "status", Value: "critical"}, {Key: "error", Value: fmt.Sprintf(format, args...)}}
}

func probeTCP(ctx context.Context, target string) []Detail {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return failure("tcp connect failed: %v", err)
	}
	defer conn.Close()
	return []Detail{{Key: "reachable", Value: fmt.Sprintf("tcp %s reachable", target)}}
}

func probeHTTP(ctx context.Context, target string, accepted []int) []Detail {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return failure("invalid http target: %v", err)
	}
	client := &http.Client{Timeout: 0} // the context owns the deadline
	response, err := client.Do(request)
	if err != nil {
		return failure("http request failed: %v", err)
	}
	defer response.Body.Close()

	status := response.StatusCode
	ok := false
	if len(accepted) == 0 {
		ok = status >= 200 && status < 300
	} else {
		for _, code := range accepted {
			if status == code {
				ok = true
				break
			}
		}
	}
	line := Detail{Key: "reachable", Value: fmt.Sprintf("http %s returned %d", target, status)}
	if !ok {
		return []Detail{{Key: "status", Value: "warning"}, line}
	}
	return []Detail{line}
}

func probeDNS(ctx context.Context, target string) []Detail {
	name := strings.TrimSuffix(target, ".")
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, name)
	if err != nil {
		return failure("dns lookup failed: %v", err)
	}
	if len(addresses) == 0 {
		return failure("dns lookup returned no addresses")
	}
	shown := make([]string, 0, 3)
	for _, address := range addresses {
		shown = append(shown, address.IP.String())
		if len(shown) == 3 {
			break
		}
	}
	return []Detail{{Key: "resolved", Value: fmt.Sprintf("%s -> %s", name, strings.Join(shown, ", "))}}
}

func probeTLS(ctx context.Context, target string, skipVerify bool) []Detail {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	// tls.Dialer applies the context to the whole handshake, not just the TCP
	// connect; a server that accepts but never completes TLS must not hang.
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{},
		Config: &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: skipVerify, // #nosec G402 -- opt-in for internal CAs
			MinVersion:         tls.VersionTLS12,
		},
	}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return failure("tls handshake failed: %v", err)
	}
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return failure("tls connection is not a TLS connection")
	}
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return failure("tls peer sent no certificate")
	}
	leaf := state.PeerCertificates[0]
	remaining := time.Until(leaf.NotAfter)
	line := Detail{Key: "expires", Value: fmt.Sprintf("certificate for %s expires in %d days", host, int(remaining.Hours()/24))}
	if remaining <= 0 {
		return []Detail{{Key: "status", Value: "critical"}, line}
	}
	if remaining < tlsWarningWindow {
		return []Detail{{Key: "status", Value: "warning"}, line}
	}
	return []Detail{line}
}
