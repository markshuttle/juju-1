// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package metricsmanager_test

import (
	stdtesting "testing"

	jc "github.com/juju/testing/checkers"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/api/metricsmanager"
	"github.com/juju/juju/apiserver/params"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/testing"
)

type metricsManagerSuite struct {
	jujutesting.JujuConnSuite

	manager *metricsmanager.Client
}

var _ = gc.Suite(&metricsManagerSuite{})

func TestAll(t *stdtesting.T) {
	testing.MgoTestPackage(t)
}

func (s *metricsManagerSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.manager = metricsmanager.NewClient(s.APIState)
	c.Assert(s.manager, gc.NotNil)
}

func (s *metricsManagerSuite) TestCleanupOldMetrics(c *gc.C) {
	var called bool
	metricsmanager.PatchFacadeCall(s, s.manager, func(request string, args, response interface{}) error {
		called = true
		c.Assert(request, gc.Equals, "CleanupOldMetrics")
		result := response.(*params.ErrorResults)
		result.Results = make([]params.ErrorResult, 1)
		return nil
	})
	err := s.manager.CleanupOldMetrics()
	c.Assert(err, gc.IsNil)
	c.Assert(called, jc.IsTrue)
}

func (s *metricsManagerSuite) TestSendMetrics(c *gc.C) {
	// TODO (mattyw) Can remove mock sender from statetesting?
	var called bool
	metricsmanager.PatchFacadeCall(s, s.manager, func(request string, args, response interface{}) error {
		called = true
		c.Assert(request, gc.Equals, "SendMetrics")
		result := response.(*params.ErrorResults)
		result.Results = make([]params.ErrorResult, 1)
		return nil
	})
	err := s.manager.SendMetrics()
	c.Assert(err, gc.IsNil)
	c.Assert(called, jc.IsTrue)
}
