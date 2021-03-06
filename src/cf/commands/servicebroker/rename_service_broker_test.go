package servicebroker_test

import (
	. "cf/commands/servicebroker"
	"cf/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testapi "testhelpers/api"
	testassert "testhelpers/assert"
	testcmd "testhelpers/commands"
	testconfig "testhelpers/configuration"
	testreq "testhelpers/requirements"
	testterm "testhelpers/terminal"
)

func callRenameServiceBroker(args []string, reqFactory *testreq.FakeReqFactory, repo *testapi.FakeServiceBrokerRepo) (ui *testterm.FakeUI) {
	ui = &testterm.FakeUI{}
	config := testconfig.NewRepositoryWithDefaults()
	cmd := NewRenameServiceBroker(ui, config, repo)
	ctxt := testcmd.NewContext("rename-service-broker", args)
	testcmd.RunCommand(cmd, ctxt, reqFactory)

	return
}

var _ = Describe("Testing with ginkgo", func() {
	It("TestRenameServiceBrokerFailsWithUsage", func() {
		reqFactory := &testreq.FakeReqFactory{}
		repo := &testapi.FakeServiceBrokerRepo{}

		ui := callRenameServiceBroker([]string{}, reqFactory, repo)
		Expect(ui.FailedWithUsage).To(BeTrue())

		ui = callRenameServiceBroker([]string{"arg1"}, reqFactory, repo)
		Expect(ui.FailedWithUsage).To(BeTrue())

		ui = callRenameServiceBroker([]string{"arg1", "arg2"}, reqFactory, repo)
		Expect(ui.FailedWithUsage).To(BeFalse())
	})
	It("TestRenameServiceBrokerRequirements", func() {

		reqFactory := &testreq.FakeReqFactory{}
		repo := &testapi.FakeServiceBrokerRepo{}
		args := []string{"arg1", "arg2"}

		reqFactory.LoginSuccess = false
		callRenameServiceBroker(args, reqFactory, repo)
		Expect(testcmd.CommandDidPassRequirements).To(BeFalse())

		reqFactory.LoginSuccess = true
		callRenameServiceBroker(args, reqFactory, repo)
		Expect(testcmd.CommandDidPassRequirements).To(BeTrue())
	})
	It("TestRenameServiceBroker", func() {

		reqFactory := &testreq.FakeReqFactory{LoginSuccess: true}
		broker := models.ServiceBroker{}
		broker.Name = "my-found-broker"
		broker.Guid = "my-found-broker-guid"
		repo := &testapi.FakeServiceBrokerRepo{
			FindByNameServiceBroker: broker,
		}
		args := []string{"my-broker", "my-new-broker"}

		ui := callRenameServiceBroker(args, reqFactory, repo)

		Expect(repo.FindByNameName).To(Equal("my-broker"))

		testassert.SliceContains(ui.Outputs, testassert.Lines{
			{"Renaming service broker", "my-found-broker", "my-new-broker", "my-user"},
			{"OK"},
		})

		Expect(repo.RenamedServiceBrokerGuid).To(Equal("my-found-broker-guid"))
		Expect(repo.RenamedServiceBrokerName).To(Equal("my-new-broker"))
	})
})
