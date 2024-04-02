package backend

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/vennekilde/gw2-alliance-bot/internal/api"
)

const PlatformID = 2

const (
	SettingWvWWorld           = "wvw_world"
	SettingPrimaryRole        = "wvw_primary_role"
	SettingLinkedRole         = "wvw_linked_role"
	SettingAssociatedRoles    = "wvw_associated_roles"
	SettingAccRepEnabled      = "acc_rep_nick"
	SettingGuildTagRepEnabled = "guild_tag_rep_nick"
	SettingGuildCommonRole    = "verification_role"
)

type Service struct {
	backend     *api.ClientWithResponses
	settings    map[string]map[string]string
	serviceUUID string
}

func NewService(backend *api.ClientWithResponses, serviceUUID string) *Service {
	return &Service{
		backend:     backend,
		serviceUUID: serviceUUID,
		settings:    make(map[string]map[string]string),
	}
}

func (s *Service) Synchronize() error {
	// settings
	ctx := context.Background()
	resp, err := s.backend.GetServicePropertiesWithResponse(ctx, s.serviceUUID)
	if err != nil {
		return err
	}

	if resp.JSON200 == nil {
		return nil
	}

	settings := make(map[string]map[string]string)
	for _, property := range *resp.JSON200 {
		subject := *property.Subject
		subjectSettings, ok := settings[subject]
		if !ok {
			subjectSettings = make(map[string]string)
			settings[subject] = subjectSettings
		}
		subjectSettings[property.Name] = property.Value
	}

	s.settings = settings
	return nil
}

func (s *Service) GetSetting(subject string, name string) string {
	subjectSettings, ok := s.settings[subject]
	if !ok {
		return ""
	}
	return subjectSettings[name]
}

func (s *Service) SetSetting(ctx context.Context, subject string, name string, value string) error {
	_, err := s.backend.PutServiceSubjectPropertyWithResponse(ctx, s.serviceUUID, subject, name, func(ctx context.Context, req *http.Request) error {
		req.Body = io.NopCloser(strings.NewReader(value))
		return nil
	})
	if err != nil {
		return err
	}

	subjectSettings, ok := s.settings[subject]
	if !ok {
		subjectSettings = make(map[string]string)
		s.settings[subject] = subjectSettings
	}
	subjectSettings[name] = value
	return nil
}
