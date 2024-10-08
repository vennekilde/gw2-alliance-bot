package backend

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

const PlatformID = 2

const (
	SettingWvWWorld                 = "wvw_world"
	SettingPrimaryRole              = "wvw_primary_role"
	SettingLinkedRole               = "wvw_linked_role"
	SettingAssociatedRoles          = "wvw_associated_roles"
	SettingAccRepEnabled            = "acc_rep_nick"
	SettingGuildTagRepEnabled       = "guild_tag_rep_nick"
	SettingEnforceGuildRep          = "enforce_guild_rep"
	SettingGuildCommonRole          = "verification_role"
	SettingGuildVerifyRoles         = "guild_verify_roles"
	SettingGuildRequiredPermissions = "guild_required_permissions"
)

type Service struct {
	m           sync.Mutex
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
	} else if resp.JSON200 == nil {
		zap.L().Error("unexpected response", zap.Any("response", resp))
		return errors.New("unexpected response")
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

	s.m.Lock()
	defer s.m.Unlock()
	s.settings = settings
	return nil
}

func (s *Service) GetSetting(subject string, name string) string {
	s.m.Lock()
	defer s.m.Unlock()

	subjectSettings, ok := s.settings[subject]
	if !ok {
		return ""
	}
	return subjectSettings[name]
}

func (s *Service) GetSettingSlice(subject string, name string) []string {
	s.m.Lock()
	defer s.m.Unlock()

	subjectSettings, ok := s.settings[subject]
	if !ok {
		return nil
	}

	if subjectSettings[name] == "" {
		return nil
	}

	return strings.Split(subjectSettings[name], ",")
}

func (s *Service) GetSettingDefault(subject string, name string, def string) string {
	s.m.Lock()
	defer s.m.Unlock()

	subjectSettings, ok := s.settings[subject]
	if !ok {
		return def
	}
	return subjectSettings[name]
}

func (s *Service) SetSetting(ctx context.Context, subject string, name string, value string) error {
	s.m.Lock()
	defer s.m.Unlock()

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
