package route_test

import (
	. "cf/commands/route"
	"cf/configuration"
	"cf/models"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	mr "github.com/tjarratt/mr_t"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
)

func callListRoutes(t mr.TestingT, args []string, reqFactory *testreq.FakeReqFactory, routeRepo *testapi.FakeRouteRepository) (ui *testterm.FakeUI) {

	ui = &testterm.FakeUI{}

	ctxt := testcmd.NewContext("routes", args)

	token, err := testconfig.CreateAccessTokenWithTokenInfo(configuration.TokenInfo{
		Username: "my-user",
	})
	assert.NoError(t, err)
	space := models.SpaceFields{}
	space.Name = "my-space"
	org := models.OrganizationFields{}
	org.Name = "my-org"
	config := &configuration.Configuration{
		SpaceFields:        space,
		OrganizationFields: org,
		AccessToken:        token,
	}

	cmd := NewListRoutes(ui, config, routeRepo)
	testcmd.RunCommand(cmd, ctxt, reqFactory)

	return
}
func init() {
	Describe("Testing with ginkgo", func() {
		It("TestListingRoutes", func() {
			domain := models.DomainFields{}
			domain.Name = "example.com"
			domain2 := models.DomainFields{}
			domain2.Name = "cfapps.com"
			domain3 := models.DomainFields{}
			domain3.Name = "another-example.com"

			app1 := models.ApplicationFields{}
			app1.Name = "dora"
			app2 := models.ApplicationFields{}
			app2.Name = "dora2"

			app3 := models.ApplicationFields{}
			app3.Name = "my-app"
			app4 := models.ApplicationFields{}
			app4.Name = "my-app2"

			app5 := models.ApplicationFields{}
			app5.Name = "july"

			route := models.Route{}
			route.Host = "hostname-1"
			route.Domain = domain
			route.Apps = []models.ApplicationFields{app1, app2}
			route2 := models.Route{}
			route2.Host = "hostname-2"
			route2.Domain = domain2
			route2.Apps = []models.ApplicationFields{app3, app4}
			route3 := models.Route{}
			route3.Host = "hostname-3"
			route3.Domain = domain3
			route3.Apps = []models.ApplicationFields{app5}
			routes := []models.Route{route, route2, route3}

			routeRepo := &testapi.FakeRouteRepository{Routes: routes}

			ui := callListRoutes(mr.T(), []string{}, &testreq.FakeReqFactory{}, routeRepo)

			testassert.SliceContains(mr.T(), ui.Outputs, testassert.Lines{
				{"Getting routes", "my-user"},
				{"host", "domain", "apps"},
				{"hostname-1", "example.com", "dora", "dora2"},
				{"hostname-2", "cfapps.com", "my-app", "my-app2"},
				{"hostname-3", "another-example.com", "july"},
			})
		})
		It("TestListingRoutesWhenNoneExist", func() {

			routes := []models.Route{}
			routeRepo := &testapi.FakeRouteRepository{Routes: routes}

			ui := callListRoutes(mr.T(), []string{}, &testreq.FakeReqFactory{}, routeRepo)

			testassert.SliceContains(mr.T(), ui.Outputs, testassert.Lines{
				{"Getting routes"},
				{"No routes found"},
			})
		})
		It("TestListingRoutesWhenFindFails", func() {

			routeRepo := &testapi.FakeRouteRepository{ListErr: true}

			ui := callListRoutes(mr.T(), []string{}, &testreq.FakeReqFactory{}, routeRepo)

			testassert.SliceContains(mr.T(), ui.Outputs, testassert.Lines{
				{"Getting routes"},
				{"FAILED"},
			})
		})
	})
}
