package application_test

import (
	"cf"
	"cf/api"
	. "cf/commands/application"
	"cf/configuration"
	"code.google.com/p/gogoprotobuf/proto"
	"errors"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	"github.com/stretchr/testify/assert"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
	"testing"
	"time"
)

var (
	defaultAppForStart        = cf.Application{}
	defaultInstanceReponses   = [][]cf.ApplicationInstance{}
	defaultInstanceErrorCodes = []string{"", ""}
)

func init() {
	defaultAppForStart.Name = "my-app"
	defaultAppForStart.Guid = "my-app-guid"
	defaultAppForStart.InstanceCount = 2

	domain := cf.DomainFields{}
	domain.Name = "example.com"

	route := cf.RouteSummary{}
	route.Host = "my-app"
	route.Domain = domain

	defaultAppForStart.Routes = []cf.RouteSummary{route}

	instance1 := cf.ApplicationInstance{}
	instance1.State = cf.InstanceStarting

	instance2 := cf.ApplicationInstance{}
	instance2.State = cf.InstanceStarting

	instance3 := cf.ApplicationInstance{}
	instance3.State = cf.InstanceRunning

	instance4 := cf.ApplicationInstance{}
	instance4.State = cf.InstanceStarting

	defaultInstanceReponses = [][]cf.ApplicationInstance{
		[]cf.ApplicationInstance{instance1, instance2},
		[]cf.ApplicationInstance{instance3, instance4},
	}
}

func callStart(args []string, config *configuration.Configuration, reqFactory *testreq.FakeReqFactory, appRepo api.ApplicationRepository, appInstancesRepo api.AppInstancesRepository, logRepo api.LogsRepository) (ui *testterm.FakeUI) {
	ui = new(testterm.FakeUI)
	ctxt := testcmd.NewContext("start", args)

	cmd := NewStart(ui, config, appRepo, appInstancesRepo, logRepo)
	testcmd.RunCommand(cmd, ctxt, reqFactory)
	return
}

func startAppWithInstancesAndErrors(t *testing.T, app cf.Application, instances [][]cf.ApplicationInstance, errorCodes []string) (ui *testterm.FakeUI, appRepo *testapi.FakeApplicationRepository, appInstancesRepo *testapi.FakeAppInstancesRepo, reqFactory *testreq.FakeReqFactory) {
	token, err := testconfig.CreateAccessTokenWithTokenInfo(configuration.TokenInfo{
		Username: "my-user",
	})
	assert.NoError(t, err)
	space := cf.SpaceFields{}
	space.Name = "my-space"
	org := cf.OrganizationFields{}
	org.Name = "my-org"
	config := &configuration.Configuration{
		Space:                   space,
		Organization:            org,
		AccessToken:             token,
		ApplicationStartTimeout: 2,
	}

	appRepo = &testapi.FakeApplicationRepository{
		FindByNameApp:   app,
		StartUpdatedApp: app,
	}
	appInstancesRepo = &testapi.FakeAppInstancesRepo{
		GetInstancesResponses:  instances,
		GetInstancesErrorCodes: errorCodes,
	}

	currentTime := time.Now()
	messageType := logmessage.LogMessage_ERR
	sourceType := logmessage.LogMessage_DEA
	logMessage1 := logmessage.LogMessage{
		Message:     []byte("Log Line 1"),
		AppId:       proto.String(app.Guid),
		MessageType: &messageType,
		SourceType:  &sourceType,
		Timestamp:   proto.Int64(currentTime.UnixNano()),
	}

	logMessage2 := logmessage.LogMessage{
		Message:     []byte("Log Line 2"),
		AppId:       proto.String(app.Guid),
		MessageType: &messageType,
		SourceType:  &sourceType,
		Timestamp:   proto.Int64(currentTime.UnixNano()),
	}

	logRepo := &testapi.FakeLogsRepository{
		TailLogMessages: []logmessage.LogMessage{
			logMessage1,
			logMessage2,
		},
	}

	args := []string{"my-app"}
	reqFactory = &testreq.FakeReqFactory{Application: app}
	ui = callStart(args, config, reqFactory, appRepo, appInstancesRepo, logRepo)
	return
}

func TestStartCommandFailsWithUsage(t *testing.T) {
	t.Parallel()

	config := &configuration.Configuration{}
	appRepo := &testapi.FakeApplicationRepository{}
	appInstancesRepo := &testapi.FakeAppInstancesRepo{
		GetInstancesResponses: [][]cf.ApplicationInstance{
			[]cf.ApplicationInstance{},
		},
		GetInstancesErrorCodes: []string{""},
	}
	logRepo := &testapi.FakeLogsRepository{}

	reqFactory := &testreq.FakeReqFactory{}

	ui := callStart([]string{}, config, reqFactory, appRepo, appInstancesRepo, logRepo)
	assert.True(t, ui.FailedWithUsage)

	ui = callStart([]string{"my-app"}, config, reqFactory, appRepo, appInstancesRepo, logRepo)
	assert.False(t, ui.FailedWithUsage)
}

func TestStartApplication(t *testing.T) {
	t.Parallel()

	ui, appRepo, _, reqFactory := startAppWithInstancesAndErrors(t, defaultAppForStart, defaultInstanceReponses, defaultInstanceErrorCodes)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[0], "my-org")
	assert.Contains(t, ui.Outputs[0], "my-space")
	assert.Contains(t, ui.Outputs[0], "my-user")
	assert.Contains(t, ui.Outputs[1], "OK")
	assert.Contains(t, ui.Outputs[6], "0 of 2 instances running (2 starting)")
	assert.Contains(t, ui.Outputs[7], "Started")
	assert.Contains(t, ui.Outputs[7], "my-app")
	assert.Contains(t, ui.Outputs[7], "my-app.example.com")

	assert.Equal(t, reqFactory.ApplicationName, "my-app")
	assert.Equal(t, appRepo.StartAppGuid, "my-app-guid")
}

func TestStartApplicationWhenAppHasNoURL(t *testing.T) {
	t.Parallel()

	app := defaultAppForStart
	app.Routes = []cf.RouteSummary{}
	appInstance5 := cf.ApplicationInstance{}
	appInstance5.State = cf.InstanceRunning
	instances := [][]cf.ApplicationInstance{
		[]cf.ApplicationInstance{appInstance5},
	}

	errorCodes := []string{""}
	ui, appRepo, _, reqFactory := startAppWithInstancesAndErrors(t, app, instances, errorCodes)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[1], "OK")
	assert.Contains(t, ui.Outputs[6], "Started")

	assert.Equal(t, reqFactory.ApplicationName, "my-app")
	assert.Equal(t, appRepo.StartAppGuid, "my-app-guid")
}

func TestStartApplicationWhenAppIsStillStaging(t *testing.T) {
	t.Parallel()
	appInstance6 := cf.ApplicationInstance{}
	appInstance6.State = cf.InstanceDown
	appInstance7 := cf.ApplicationInstance{}
	appInstance7.State = cf.InstanceStarting
	appInstance8 := cf.ApplicationInstance{}
	appInstance8.State = cf.InstanceStarting
	appInstance9 := cf.ApplicationInstance{}
	appInstance9.State = cf.InstanceStarting
	appInstance10 := cf.ApplicationInstance{}
	appInstance10.State = cf.InstanceRunning
	appInstance11 := cf.ApplicationInstance{}
	appInstance11.State = cf.InstanceRunning
	instances := [][]cf.ApplicationInstance{
		[]cf.ApplicationInstance{},
		[]cf.ApplicationInstance{},
		[]cf.ApplicationInstance{appInstance6, appInstance7},
		[]cf.ApplicationInstance{appInstance8, appInstance9},
		[]cf.ApplicationInstance{appInstance10, appInstance11},
	}

	errorCodes := []string{cf.APP_NOT_STAGED, cf.APP_NOT_STAGED, "", "", ""}

	ui, _, appInstancesRepo, _ := startAppWithInstancesAndErrors(t, defaultAppForStart, instances, errorCodes)

	assert.Equal(t, appInstancesRepo.GetInstancesAppGuid, "my-app-guid")

	assert.Contains(t, ui.Outputs[2], "Staging")
	assert.Contains(t, ui.Outputs[3], "Log Line 1")
	assert.Contains(t, ui.Outputs[4], "Log Line 2")

	assert.Contains(t, ui.Outputs[6], "0 of 2 instances running (1 starting, 1 down)")
	assert.Contains(t, ui.Outputs[7], "0 of 2 instances running (2 starting)")
}

func TestStartApplicationWhenStagingFails(t *testing.T) {
	t.Parallel()

	instances := [][]cf.ApplicationInstance{[]cf.ApplicationInstance{}}
	errorCodes := []string{"170001"}

	ui, _, _, _ := startAppWithInstancesAndErrors(t, defaultAppForStart, instances, errorCodes)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[1], "OK")
	assert.Contains(t, ui.Outputs[6], "FAILED")
	assert.Contains(t, ui.Outputs[7], "Error staging app")
}

func TestStartApplicationWhenOneInstanceFlaps(t *testing.T) {
	t.Parallel()
	appInstance12 := cf.ApplicationInstance{}
	appInstance12.State = cf.InstanceStarting
	appInstance13 := cf.ApplicationInstance{}
	appInstance13.State = cf.InstanceStarting
	appInstance14 := cf.ApplicationInstance{}
	appInstance14.State = cf.InstanceStarting
	appInstance15 := cf.ApplicationInstance{}
	appInstance15.State = cf.InstanceFlapping
	instances := [][]cf.ApplicationInstance{
		[]cf.ApplicationInstance{appInstance12, appInstance13},
		[]cf.ApplicationInstance{appInstance14, appInstance15},
	}

	errorCodes := []string{"", ""}

	ui, _, _, _ := startAppWithInstancesAndErrors(t, defaultAppForStart, instances, errorCodes)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[1], "OK")
	assert.Contains(t, ui.Outputs[6], "0 of 2 instances running (2 starting)")
	assert.Contains(t, ui.Outputs[7], "FAILED")
	assert.Contains(t, ui.Outputs[8], "Start unsuccessful")
}

func TestStartApplicationWhenStartTimesOut(t *testing.T) {
	t.Parallel()
	appInstance16 := cf.ApplicationInstance{}
	appInstance16.State = cf.InstanceStarting
	appInstance17 := cf.ApplicationInstance{}
	appInstance17.State = cf.InstanceStarting
	appInstance18 := cf.ApplicationInstance{}
	appInstance18.State = cf.InstanceStarting
	appInstance19 := cf.ApplicationInstance{}
	appInstance19.State = cf.InstanceDown
	appInstance20 := cf.ApplicationInstance{}
	appInstance20.State = cf.InstanceDown
	appInstance21 := cf.ApplicationInstance{}
	appInstance21.State = cf.InstanceDown
	instances := [][]cf.ApplicationInstance{
		[]cf.ApplicationInstance{appInstance16, appInstance17},
		[]cf.ApplicationInstance{appInstance18, appInstance19},
		[]cf.ApplicationInstance{appInstance20, appInstance21},
	}

	errorCodes := []string{"", "", ""}

	ui, _, _, _ := startAppWithInstancesAndErrors(t, defaultAppForStart, instances, errorCodes)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[1], "OK")
	assert.Contains(t, ui.Outputs[6], "0 of 2 instances running (2 starting)")
	assert.Contains(t, ui.Outputs[7], "0 of 2 instances running (1 starting, 1 down)")
	assert.Contains(t, ui.Outputs[8], "0 of 2 instances running (2 down)")
	assert.Contains(t, ui.Outputs[9], "FAILED")
	assert.Contains(t, ui.Outputs[10], "Start app timeout")
}

func TestStartApplicationWhenStartFails(t *testing.T) {
	t.Parallel()

	config := &configuration.Configuration{}
	app := cf.Application{}
	app.Name = "my-app"
	app.Guid = "my-app-guid"
	appRepo := &testapi.FakeApplicationRepository{FindByNameApp: app, StartAppErr: true}
	appInstancesRepo := &testapi.FakeAppInstancesRepo{}
	logRepo := &testapi.FakeLogsRepository{}
	args := []string{"my-app"}
	reqFactory := &testreq.FakeReqFactory{Application: app}
	ui := callStart(args, config, reqFactory, appRepo, appInstancesRepo, logRepo)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[1], "FAILED")
	assert.Contains(t, ui.Outputs[2], "Error starting application")
	assert.Equal(t, appRepo.StartAppGuid, "my-app-guid")
}

func TestStartApplicationIsAlreadyStarted(t *testing.T) {
	t.Parallel()

	config := &configuration.Configuration{}
	app := cf.Application{}
	app.Name = "my-app"
	app.Guid = "my-app-guid"
	app.State = "started"
	appRepo := &testapi.FakeApplicationRepository{FindByNameApp: app}
	appInstancesRepo := &testapi.FakeAppInstancesRepo{}
	logRepo := &testapi.FakeLogsRepository{}

	reqFactory := &testreq.FakeReqFactory{Application: app}

	args := []string{"my-app"}
	ui := callStart(args, config, reqFactory, appRepo, appInstancesRepo, logRepo)

	assert.Contains(t, ui.Outputs[0], "my-app")
	assert.Contains(t, ui.Outputs[0], "is already started")
	assert.Equal(t, appRepo.StartAppGuid, "")
}

func TestStartApplicationWithLoggingFailure(t *testing.T) {
	t.Parallel()

	token, err := testconfig.CreateAccessTokenWithTokenInfo(configuration.TokenInfo{Username: "my-user"})
	assert.NoError(t, err)
	space2 := cf.SpaceFields{}
	space2.Name = "my-space"
	org2 := cf.OrganizationFields{}
	org2.Name = "my-org"
	config := &configuration.Configuration{
		Space:                   space2,
		Organization:            org2,
		AccessToken:             token,
		ApplicationStartTimeout: 2,
	}

	appRepo := &testapi.FakeApplicationRepository{FindByNameApp: defaultAppForStart}
	appInstancesRepo := &testapi.FakeAppInstancesRepo{
		GetInstancesResponses:  defaultInstanceReponses,
		GetInstancesErrorCodes: defaultInstanceErrorCodes,
	}

	logRepo := &testapi.FakeLogsRepository{
		TailLogErr: errors.New("Ooops"),
	}

	reqFactory := &testreq.FakeReqFactory{Application: defaultAppForStart}

	ui := new(testterm.FakeUI)

	ctxt := testcmd.NewContext("start", []string{"my-app"})

	cmd := NewStart(ui, config, appRepo, appInstancesRepo, logRepo)

	testcmd.RunCommand(cmd, ctxt, reqFactory)

	testassert.SliceContains(t, ui.Outputs, []string{
		"error tailing logs",
		"Ooops",
	})
}
