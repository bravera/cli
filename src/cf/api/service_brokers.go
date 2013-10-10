package api

import (
	"cf"
	"cf/configuration"
	"cf/net"
	"fmt"
	"strings"
)

type ServiceBrokerRepository interface {
	FindAll() (serviceBrokers []cf.ServiceBroker, apiResponse net.ApiResponse)
	FindByName(name string) (serviceBroker cf.ServiceBroker, apiResponse net.ApiResponse)
	Create(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse)
	Update(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse)
	Rename(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse)
	Delete(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse)
}

type CloudControllerServiceBrokerRepository struct {
	config  *configuration.Configuration
	gateway net.Gateway
}

func NewCloudControllerServiceBrokerRepository(config *configuration.Configuration, gateway net.Gateway) (repo CloudControllerServiceBrokerRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceBrokerRepository) FindAll() (serviceBrokers []cf.ServiceBroker, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_brokers", repo.config.Target)
	req, apiResponse := repo.gateway.NewRequest("GET", path, repo.config.AccessToken, nil)

	resources := new(PaginatedResources)
	_, apiResponse = repo.gateway.PerformRequestForJSONResponse(req, resources)

	for _, serviceBrokerResponse := range resources {

		serviceBrokers = append(serviceBrokers,
			cf.ServiceBroker{
				Name:     serviceBrokerResponse.Entity.Name,
				Guid:     serviceBrokerResponse.Metadata.Guid,
				Url:      serviceBrokerResponse.Entity.Url,
				Username: serviceBrokerResponse.Entity.Username,
				Password: serviceBrokerResponse.Entity.Password,
			},
		)
	}

	return
}

func (repo CloudControllerServiceBrokerRepository) FindByName(name string) (serviceBroker cf.ServiceBroker, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_brokers?q=name%%3A%s", repo.config.Target, name)
	req, apiResponse := repo.gateway.NewRequest("GET", path, repo.config.AccessToken, nil)
	if apiResponse.IsNotSuccessful() {
		return
	}

	resources := new(PaginatedResources)
	_, apiResponse = repo.gateway.PerformRequestForJSONResponse(req, resources)
	if apiResponse.IsNotSuccessful() {
		return
	}

	if len(resources.Resources) == 0 {
		apiResponse = net.NewNotFoundApiResponse("%s %s not found", "Service Broker", name)
		return
	}

	resource := resources.Resources[0]
	serviceBroker.Name = resource.Entity.Name
	serviceBroker.Username = resource.Entity.Username
	serviceBroker.Password = resource.Entity.Password
	serviceBroker.Url = resource.Entity.Url
	serviceBroker.Guid = resource.Metadata.Guid

	return
}

func (repo CloudControllerServiceBrokerRepository) Create(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse) {
	body := fmt.Sprintf(
		`{"name":"%s","broker_url":"%s","auth_username":"%s","auth_password":"%s"}`,
		serviceBroker.Name, serviceBroker.Url, serviceBroker.Username, serviceBroker.Password,
	)

	return repo.createOrUpdate(serviceBroker, body)
}

func (repo CloudControllerServiceBrokerRepository) Update(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse) {
	body := fmt.Sprintf(
		`{"broker_url":"%s","auth_username":"%s","auth_password":"%s"}`,
		serviceBroker.Url, serviceBroker.Username, serviceBroker.Password,
	)

	return repo.createOrUpdate(serviceBroker, body)
}

func (repo CloudControllerServiceBrokerRepository) Rename(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse) {
	body := fmt.Sprintf(`{"name":"%s"}`, serviceBroker.Name)

	return repo.createOrUpdate(serviceBroker, body)
}

func (repo CloudControllerServiceBrokerRepository) createOrUpdate(serviceBroker cf.ServiceBroker, body string) (apiResponse net.ApiResponse) {
	method := "POST"
	path := fmt.Sprintf("%s/v2/service_brokers", repo.config.Target)

	if serviceBroker.Guid != "" {
		method = "PUT"
		path = fmt.Sprintf("%s/v2/service_brokers/%s", repo.config.Target, serviceBroker.Guid)
	}

	req, apiResponse := repo.gateway.NewRequest(method, path, repo.config.AccessToken, strings.NewReader(body))
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(req)
	return
}

func (repo CloudControllerServiceBrokerRepository) Delete(serviceBroker cf.ServiceBroker) (apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_brokers/%s", repo.config.Target, serviceBroker.Guid)
	req, apiResponse := repo.gateway.NewRequest("DELETE", path, repo.config.AccessToken, nil)
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(req)
	return
}
