package service_test

import (
	"cf"
	. "cf/commands/service"
	"cf/configuration"
	"cf/net"
	"errors"
	"github.com/stretchr/testify/assert"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	"testhelpers/maker"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
	"testing"
)

func TestPurgeServiceRequirements(t *testing.T) {
	deps := setupDependencies()

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{}),
		deps.reqFactory,
	)

	assert.False(t, testcmd.CommandDidPassRequirements)
	assert.True(t, deps.ui.FailedWithUsage)
	assert.Equal(t, deps.ui.FailedWithUsageCommandName, "purge-service-offering")
}

func TestPurgeServiceWorksWithProvider(t *testing.T) {
	deps := setupDependencies()

	offering := maker.NewServiceOffering("the-service-name")
	deps.serviceRepo.FindServiceOfferingByLabelAndProviderServiceOffering = offering

	deps.ui.Inputs = []string{"yes"}

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{"-p", "the-provider", "the-service-name"}),
		deps.reqFactory,
	)

	assert.Equal(t, deps.serviceRepo.FindServiceOfferingByLabelAndProviderName, "the-service-name")
	assert.Equal(t, deps.serviceRepo.FindServiceOfferingByLabelAndProviderProvider, "the-provider")
	assert.Equal(t, deps.serviceRepo.PurgedServiceOffering, offering)
}

func TestPurgeServiceWorksWithoutProvider(t *testing.T) {
	deps := setupDependencies()

	offering := maker.NewServiceOffering("the-service-name")
	deps.serviceRepo.FindServiceOfferingByLabelAndProviderServiceOffering = offering

	deps.ui.Inputs = []string{"yes"}

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{"the-service-name"}),
		deps.reqFactory,
	)

	testassert.SliceContains(t, deps.ui.Outputs, testassert.Lines{
		{"Warning"},
	})
	testassert.SliceContains(t, deps.ui.Prompts, testassert.Lines{
		{"Really purge service", "the-service-name"},
	})

	assert.Equal(t, deps.serviceRepo.FindServiceOfferingByLabelAndProviderName, "the-service-name")
	assert.Equal(t, deps.serviceRepo.FindServiceOfferingByLabelAndProviderProvider, "")
	assert.Equal(t, deps.serviceRepo.PurgedServiceOffering, offering)

	testassert.SliceContains(t, deps.ui.Outputs, testassert.Lines{
		{"OK"},
	})
}

func TestPurgeServiceExitsWhenUserDoesNotConfirm(t *testing.T) {
	deps := setupDependencies()

	deps.ui.Inputs = []string{"no"}

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{"the-service-name"}),
		deps.reqFactory,
	)

	assert.Equal(t, deps.serviceRepo.FindServiceOfferingByLabelAndProviderCalled, false)
	assert.Equal(t, deps.serviceRepo.PurgeServiceOfferingCalled, false)
}

func TestPurgeServiceDoesNotPromptWhenForcePassed(t *testing.T) {
	deps := setupDependencies()

	offering := maker.NewServiceOffering("the-service-name")
	deps.serviceRepo.FindServiceOfferingByLabelAndProviderServiceOffering = offering

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{"-f", "the-service-name"}),
		deps.reqFactory,
	)

	assert.Equal(t, len(deps.ui.Prompts), 0)
	assert.Equal(t, deps.serviceRepo.PurgeServiceOfferingCalled, true)
}

func TestPurgeServiceIndicatesWhenAPIRequestFails(t *testing.T) {
	deps := setupDependencies()

	deps.serviceRepo.FindServiceOfferingByLabelAndProviderApiResponse = net.NewApiResponseWithError("oh no!", errors.New("!"))

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{"-f", "-p", "the-provider", "the-service-name"}),
		deps.reqFactory,
	)

	testassert.SliceContains(t, deps.ui.Outputs, testassert.Lines{
		{"FAILED"},
		{"oh no!"},
	})

	assert.Equal(t, deps.serviceRepo.PurgeServiceOfferingCalled, false)
}

func TestPurgeServiceIndicatesWhenServiceDoesntExist(t *testing.T) {
	deps := setupDependencies()

	deps.serviceRepo.FindServiceOfferingByLabelAndProviderApiResponse = net.NewNotFoundApiResponse("uh oh cant find it")

	testcmd.RunCommand(
		NewPurgeServiceOffering(deps.ui, deps.config, deps.serviceRepo),
		testcmd.NewContext("purge-service-offering", []string{"-f", "-p", "the-provider", "the-service-name"}),
		deps.reqFactory,
	)

	testassert.SliceContains(t, deps.ui.Outputs, testassert.Lines{
		{"OK"},
		{"Service offering", "does not exist"},
	})

	assert.Equal(t, deps.serviceRepo.PurgeServiceOfferingCalled, false)
}

type commandDependencies struct {
	ui          *testterm.FakeUI
	config      *configuration.Configuration
	serviceRepo *testapi.FakeServiceRepo
	reqFactory  *testreq.FakeReqFactory
}

func setupDependencies() (obj commandDependencies) {
	obj.ui = &testterm.FakeUI{}

	token, _ := testconfig.CreateAccessTokenWithTokenInfo(configuration.TokenInfo{
		Username: "my-user",
	})
	org := cf.OrganizationFields{}
	org.Name = "my-org"
	space := cf.SpaceFields{}
	space.Name = "my-space"
	obj.config = &configuration.Configuration{
		SpaceFields:        space,
		OrganizationFields: org,
		AccessToken:        token,
	}

	obj.reqFactory = &testreq.FakeReqFactory{LoginSuccess: true, TargetedSpaceSuccess: true}
	obj.serviceRepo = new(testapi.FakeServiceRepo)
	return
}
