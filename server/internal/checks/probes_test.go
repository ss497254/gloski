package checks

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStableID(t *testing.T) {
	a := StableID(ProbeTCP, "host:80", []int{200, 404})
	b := StableID(ProbeTCP, "host:80", []int{404, 200})
	if a != b {
		t.Fatalf("accepted status order changed the ID: %s vs %s", a, b)
	}
	if a == StableID(ProbeTCP, "host:81", nil) {
		t.Fatal("target did not change the ID")
	}
	if a == StableID(ProbeHTTP, "host:80", []int{200, 404}) {
		t.Fatal("kind did not change the ID")
	}
	if !strings.HasPrefix(a, "connect_") {
		t.Fatalf("ID %q will not land in the Connectivity matrix column", a)
	}
}

func TestProbeTCP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	definition := Definition{Kind: ProbeTCP, Target: listener.Addr().String()}
	if got := definition.Run(context.Background(), "dev"); got.Status != StatusOK {
		t.Fatalf("live TCP probe = %+v, want ok", got)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	definition.Target = address
	if got := definition.Run(context.Background(), "dev"); got.Status != StatusCritical {
		t.Fatalf("dead TCP probe = %+v, want critical", got)
	}
}

func TestProbeHTTPAcceptedStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ok := Definition{Kind: ProbeHTTP, Target: server.URL, Accepted: []int{200}}.Run(context.Background(), "dev")
	if ok.Status != StatusOK {
		t.Fatalf("accepted status = %+v, want ok", ok)
	}
	warning := Definition{Kind: ProbeHTTP, Target: server.URL, Accepted: []int{201}}.Run(context.Background(), "dev")
	if warning.Status != StatusWarning {
		t.Fatalf("unaccepted status = %+v, want warning", warning)
	}
}

func TestProbeDNS(t *testing.T) {
	got := Definition{Kind: ProbeDNS, Target: "localhost"}.Run(context.Background(), "dev")
	if got.Status != StatusOK {
		t.Fatalf("localhost lookup = %+v, want ok", got)
	}
	got = Definition{Kind: ProbeDNS, Target: "no-such-host.invalid"}.Run(context.Background(), "dev")
	if got.Status != StatusCritical {
		t.Fatalf("invalid lookup = %+v, want critical", got)
	}
}

func TestProbeTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	target := strings.TrimPrefix(server.URL, "https://")
	got := Definition{Kind: ProbeTLS, Target: target, SkipVerify: true}.Run(context.Background(), "dev")
	if got.Status != StatusOK {
		t.Fatalf("TLS probe = %+v, want ok", got)
	}
}

func TestProbeUnknownKind(t *testing.T) {
	got := Definition{Kind: ProbeKind("bogus"), Target: "x"}.Run(context.Background(), "dev")
	if got.Status != StatusUnknown {
		t.Fatalf("unknown kind = %+v, want unknown", got)
	}
}

func TestAggregatorIsBoundedAndFlappingIsCritical(t *testing.T) {
	var agg Aggregator
	now := time.Now()
	for i := 0; i < 70; i++ {
		result := Result{Status: StatusOK}
		if i == 60 {
			result.Status = StatusCritical
			result.Summary = "down"
		}
		agg.Add(result, now.Add(time.Duration(i)*time.Minute))
	}
	raw, minutes := agg.sizes()
	if raw != rawSampleLimit || minutes != minuteBucketLimit {
		t.Fatalf("aggregator sizes = %d, %d; want %d, %d", raw, minutes, rawSampleLimit, minuteBucketLimit)
	}
	// The single failure is outside the last five raw samples.
	got := agg.Add(Result{Status: StatusOK, Summary: "up"}, now.Add(70*time.Minute))
	if got.Status != StatusOK {
		t.Fatalf("old failure still dominates: %+v", got)
	}
	agg.Add(Result{Status: StatusCritical, Summary: "down"}, now.Add(71*time.Minute))
	got = agg.Add(Result{Status: StatusOK, Summary: "up"}, now.Add(72*time.Minute))
	if got.Status != StatusCritical {
		t.Fatalf("recent failure was ignored: %+v", got)
	}
}
