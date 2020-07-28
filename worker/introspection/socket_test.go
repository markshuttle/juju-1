/// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package introspection_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/juju/clock/testclock"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/juju/worker/v2"
	"github.com/juju/worker/v2/workertest"
	"github.com/prometheus/client_golang/prometheus"
	gc "gopkg.in/check.v1"

	// Bring in the state package for the tracker profile.
	"github.com/juju/juju/core/presence"
	_ "github.com/juju/juju/state"
	"github.com/juju/juju/worker/introspection"
)

type suite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&suite{})

func (s *suite) TestConfigValidation(c *gc.C) {
	w, err := introspection.NewWorker(introspection.Config{})
	c.Check(w, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "empty SocketName not valid")
}

func (s *suite) TestStartStop(c *gc.C) {
	if runtime.GOOS != "linux" {
		c.Skip("introspection worker not supported on non-linux")
	}

	w, err := introspection.NewWorker(introspection.Config{
		SocketName:         "introspection-test",
		PrometheusGatherer: prometheus.NewRegistry(),
	})
	c.Assert(err, jc.ErrorIsNil)
	workertest.CheckKill(c, w)
}

type introspectionSuite struct {
	testing.IsolationSuite

	name     string
	worker   worker.Worker
	reporter introspection.DepEngineReporter
	gatherer prometheus.Gatherer
	recorder presence.Recorder
}

var _ = gc.Suite(&introspectionSuite{})

func (s *introspectionSuite) SetUpTest(c *gc.C) {
	if runtime.GOOS != "linux" {
		c.Skip("introspection worker not supported on non-linux")
	}
	s.IsolationSuite.SetUpTest(c)
	s.reporter = nil
	s.worker = nil
	s.recorder = nil
	s.gatherer = newPrometheusGatherer()
	s.startWorker(c)
}

func (s *introspectionSuite) startWorker(c *gc.C) {
	s.name = fmt.Sprintf("introspection-test-%d", os.Getpid())
	w, err := introspection.NewWorker(introspection.Config{
		SocketName:         s.name,
		DepEngine:          s.reporter,
		PrometheusGatherer: s.gatherer,
		Presence:           s.recorder,
	})
	c.Assert(err, jc.ErrorIsNil)
	s.worker = w
	s.AddCleanup(func(c *gc.C) {
		workertest.CheckKill(c, w)
	})
}

func (s *introspectionSuite) call(c *gc.C, path string) *http.Response {
	client := unixSocketHTTPClient("@" + s.name)
	c.Assert(strings.HasPrefix(path, "/"), jc.IsTrue)
	targetURL, err := url.Parse("http://unix.socket" + path)
	c.Assert(err, jc.ErrorIsNil)

	resp, err := client.Get(targetURL.String())
	c.Assert(err, jc.ErrorIsNil)
	return resp
}

func (s *introspectionSuite) post(c *gc.C, path string, values url.Values) *http.Response {
	client := unixSocketHTTPClient("@" + s.name)
	c.Assert(strings.HasPrefix(path, "/"), jc.IsTrue)
	targetURL, err := url.Parse("http://unix.socket" + path)
	c.Assert(err, jc.ErrorIsNil)

	resp, err := client.PostForm(targetURL.String(), values)
	c.Assert(err, jc.ErrorIsNil)
	return resp
}

func (s *introspectionSuite) body(c *gc.C, r *http.Response) string {
	response, err := ioutil.ReadAll(r.Body)
	c.Assert(err, jc.ErrorIsNil)
	return string(response)
}

func (s *introspectionSuite) assertBody(c *gc.C, response *http.Response, value string) {
	body := s.body(c, response)
	c.Assert(body, gc.Equals, value+"\n")
}

func (s *introspectionSuite) assertContains(c *gc.C, value, expected string) {
	c.Assert(strings.Contains(value, expected), jc.IsTrue,
		gc.Commentf("missing %q in %v", expected, value))
}

func (s *introspectionSuite) assertBodyContains(c *gc.C, response *http.Response, value string) {
	body := s.body(c, response)
	s.assertContains(c, body, value)
}

func (s *introspectionSuite) TestCmdLine(c *gc.C) {
	response := s.call(c, "/debug/pprof/cmdline")
	s.assertBodyContains(c, response, "/introspection.test")
}

func (s *introspectionSuite) TestGoroutineProfile(c *gc.C) {
	response := s.call(c, "/debug/pprof/goroutine?debug=1")
	body := s.body(c, response)
	c.Check(body, gc.Matches, `(?s)^goroutine profile: total \d+.*`)
}

func (s *introspectionSuite) TestTrace(c *gc.C) {
	response := s.call(c, "/debug/pprof/trace?seconds=1")
	c.Assert(response.Header.Get("Content-Type"), gc.Equals, "application/octet-stream")
}

func (s *introspectionSuite) TestMissingDepEngineReporter(c *gc.C) {
	response := s.call(c, "/depengine")
	c.Assert(response.StatusCode, gc.Equals, http.StatusNotFound)
	s.assertBody(c, response, "missing dependency engine reporter")
}

func (s *introspectionSuite) TestMissingStatePoolReporter(c *gc.C) {
	response := s.call(c, "/statepool")
	c.Assert(response.StatusCode, gc.Equals, http.StatusNotFound)
	s.assertBody(c, response, "State Pool Report: missing reporter")
}

func (s *introspectionSuite) TestMissingPubSubReporter(c *gc.C) {
	response := s.call(c, "/pubsub")
	c.Assert(response.StatusCode, gc.Equals, http.StatusNotFound)
	s.assertBody(c, response, "PubSub Report: missing reporter")
}

func (s *introspectionSuite) TestMissingMachineLock(c *gc.C) {
	response := s.call(c, "/machinelock/")
	c.Assert(response.StatusCode, gc.Equals, http.StatusNotFound)
	s.assertBody(c, response, "missing machine lock reporter")
}

func (s *introspectionSuite) TestStateTrackerReporter(c *gc.C) {
	response := s.call(c, "/debug/pprof/juju/state/tracker?debug=1")
	c.Assert(response.StatusCode, gc.Equals, http.StatusOK)
	s.assertBodyContains(c, response, "juju/state/tracker profile: total")
}

func (s *introspectionSuite) TestEngineReporter(c *gc.C) {
	// We need to make sure the existing worker is shut down
	// so we can connect to the socket.
	workertest.CheckKill(c, s.worker)
	s.reporter = &reporter{
		values: map[string]interface{}{
			"working": true,
		},
	}
	s.startWorker(c)
	response := s.call(c, "/depengine")
	c.Assert(response.StatusCode, gc.Equals, http.StatusOK)
	// TODO: perhaps make the output of the dependency engine YAML parseable.
	// This could be done by having the first line start with a '#'.
	s.assertBody(c, response, `
Dependency Engine Report

working: true`[1:])
}

func (s *introspectionSuite) TestMissingPresenceReporter(c *gc.C) {
	response := s.call(c, "/presence/")
	c.Assert(response.StatusCode, gc.Equals, http.StatusNotFound)
	s.assertBody(c, response, "404 page not found")
}

func (s *introspectionSuite) TestDisabledPresenceReporter(c *gc.C) {
	// We need to make sure the existing worker is shut down
	// so we can connect to the socket.
	workertest.CheckKill(c, s.worker)
	s.recorder = presence.New(testclock.NewClock(time.Now()))
	s.startWorker(c)

	response := s.call(c, "/presence/")
	c.Assert(response.StatusCode, gc.Equals, http.StatusNotFound)
	s.assertBody(c, response, "agent is not an apiserver")
}

func (s *introspectionSuite) TestEnabledPresenceReporter(c *gc.C) {
	// We need to make sure the existing worker is shut down
	// so we can connect to the socket.
	workertest.CheckKill(c, s.worker)
	s.recorder = presence.New(testclock.NewClock(time.Now()))
	s.recorder.Enable()
	s.recorder.Connect("server", "model-uuid", "agent-1", 42, false, "")
	s.startWorker(c)

	response := s.call(c, "/presence/")
	c.Assert(response.StatusCode, gc.Equals, http.StatusOK)
	s.assertBody(c, response, `
[model-uuid]

AGENT    SERVER  CONN ID  STATUS
agent-1  server  42       alive
`[1:])
}

func (s *introspectionSuite) TestPrometheusMetrics(c *gc.C) {
	response := s.call(c, "/metrics/")
	c.Assert(response.StatusCode, gc.Equals, http.StatusOK)
	body := s.body(c, response)
	s.assertContains(c, body, "# HELP tau Tau")
	s.assertContains(c, body, "# TYPE tau counter")
	s.assertContains(c, body, "tau 6.283185")
}

type reporter struct {
	values map[string]interface{}
}

func (r *reporter) Report() map[string]interface{} {
	return r.values
}

func newPrometheusGatherer() prometheus.Gatherer {
	counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "tau", Help: "Tau."})
	counter.Add(6.283185)
	r := prometheus.NewPedanticRegistry()
	r.MustRegister(counter)
	return r
}

func unixSocketHTTPClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: unixSocketHTTPTransport(socketPath),
	}
}

func unixSocketHTTPTransport(socketPath string) *http.Transport {
	return &http.Transport{
		Dial: func(proto, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
}
