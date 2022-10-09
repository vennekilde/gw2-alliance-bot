// DO NOT EDIT THIS FILE. This file will be overwritten when re-running go-raml.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/vennekilde/gw2-alliance-bot/internal/api/goraml"
	"github.com/vennekilde/gw2-alliance-bot/internal/api/types"
)

type V1Service service

// Collect statistics based on the provided parameters and save them for historical purposes
func (s *V1Service) V1ChannelsService_idChannelStatisticsPost(channel, service_id string, body types.ChannelMetadata, headers, queryParams map[string]interface{}) (*http.Response, error) {
	var err error

	resp, err := s.client.doReqWithBody("POST", s.client.BaseURI+"/v1/channels/"+service_id+"/"+channel+"/statistics", &body, headers, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 500:
		var respBody500 types.Error
		err = goraml.NewAPIError(resp, &respBody500)
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return resp, err
}

// Get a configuration containing relevant information for running a service bot
func (s *V1Service) V1ConfigurationGet(headers, queryParams map[string]interface{}) (types.Configuration, *http.Response, error) {
	var err error
	var respBody200 types.Configuration

	resp, err := s.client.doReqNoBody("GET", s.client.BaseURI+"/v1/configuration", headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Long polling rest endpoint for receiving verification updates
func (s *V1Service) V1UpdatesService_idSubscribeGet(service_id string, headers, queryParams map[string]interface{}) (types.VerificationStatus, *http.Response, error) {
	var err error
	var respBody200 types.VerificationStatus

	resp, err := s.client.doReqNoBody("GET", s.client.BaseURI+"/v1/updates/"+service_id+"/subscribe", headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Get a service user's apikey name they are required to use if apikey name restriction is enforced
func (s *V1Service) V1UsersService_idService_user_idApikeyNameGet(service_user_id, service_id string, headers, queryParams map[string]interface{}) (types.APIKeyName, *http.Response, error) {
	var err error
	var respBody200 types.APIKeyName

	resp, err := s.client.doReqNoBody("GET", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/apikey/name", headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Set a service user's API key
func (s *V1Service) V1UsersService_idService_user_idApikeyPut(service_user_id, service_id string, body types.APIKeyData, headers, queryParams map[string]interface{}) (*http.Response, error) {
	var err error

	resp, err := s.client.doReqWithBody("PUT", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/apikey", &body, headers, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	case 500:
		var respBody500 types.Error
		err = goraml.NewAPIError(resp, &respBody500)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return resp, err
}

// Ban a user's gw2 account from being verified
func (s *V1Service) V1UsersService_idService_user_idBanPut(service_user_id, service_id string, body types.BanData, headers, queryParams map[string]interface{}) (*http.Response, error) {
	var err error

	resp, err := s.client.doReqWithBody("PUT", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/ban", &body, headers, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	case 500:
		var respBody500 types.Error
		err = goraml.NewAPIError(resp, &respBody500)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return resp, err
}

// Get a user property
func (s *V1Service) V1UsersService_idService_user_idPropertiesPropertyGet(property, service_user_id, service_id string, headers, queryParams map[string]interface{}) (types.Property, *http.Response, error) {
	var err error
	var respBody200 types.Property

	resp, err := s.client.doReqNoBody("GET", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/properties/"+property, headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Get all user properties
func (s *V1Service) V1UsersService_idService_user_idPropertiesGet(service_user_id, service_id string, headers, queryParams map[string]interface{}) ([]types.Property, *http.Response, error) {
	var err error
	var respBody200 []types.Property

	resp, err := s.client.doReqNoBody("GET", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/properties", headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Set a user property
func (s *V1Service) V1UsersService_idService_user_idPropertiesPut(service_user_id, service_id string, headers, queryParams map[string]interface{}) (*http.Response, error) {
	var err error

	resp, err := s.client.doReqWithBody("PUT", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/properties", nil, headers, queryParams)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return resp, err
}

// Forces a refresh of the API data and returns the new verification status after the API data has been refreshed. Note this can take a few seconds
func (s *V1Service) V1UsersService_idService_user_idVerificationRefreshPost(service_user_id, service_id string, headers, queryParams map[string]interface{}) (types.VerificationStatus, *http.Response, error) {
	var err error
	var respBody200 types.VerificationStatus

	resp, err := s.client.doReqWithBody("POST", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/verification/refresh", nil, headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	case 500:
		var respBody500 types.Error
		err = goraml.NewAPIError(resp, &respBody500)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Get a users verification status
func (s *V1Service) V1UsersService_idService_user_idVerificationStatusGet(service_user_id, service_id string, headers, queryParams map[string]interface{}) (types.VerificationStatus, *http.Response, error) {
	var err error
	var respBody200 types.VerificationStatus

	resp, err := s.client.doReqNoBody("GET", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/verification/status", headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}

// Grant a user temporary world relation. Additionally, the "temp_expired" property will be removed from the user's properties
func (s *V1Service) V1UsersService_idService_user_idVerificationTemporaryPut(service_user_id, service_id string, body types.TemporaryData, headers, queryParams map[string]interface{}) (int, *http.Response, error) {
	var err error
	var respBody200 int

	resp, err := s.client.doReqWithBody("PUT", s.client.BaseURI+"/v1/users/"+service_id+"/"+service_user_id+"/verification/temporary", &body, headers, queryParams)
	if err != nil {
		return respBody200, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		err = json.NewDecoder(resp.Body).Decode(&respBody200)
	case 400:
		var respBody400 types.Error
		err = goraml.NewAPIError(resp, &respBody400)
	case 500:
		var respBody500 types.Error
		err = goraml.NewAPIError(resp, &respBody500)
	default:
		err = goraml.NewAPIError(resp, nil)
	}

	return respBody200, resp, err
}
