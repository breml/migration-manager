package migration

import (
	"encoding/json"
	"net/url"

	"github.com/FuturFusion/migration-manager/shared/api"
)

type Source struct {
	ID         int
	Name       string
	SourceType api.SourceType

	Properties json.RawMessage
}

func (s Source) Validate() error {
	if s.ID < 0 {
		return NewValidationErrf("Invalid source, id can not be negative")
	}

	if s.Name == "" {
		return NewValidationErrf("Invalid source, name can not be empty")
	}

	if s.SourceType < api.SOURCETYPE_COMMON || s.SourceType > api.SOURCETYPE_VMWARE {
		return NewValidationErrf("Invalid source, %d is not a valid source type", s.SourceType)
	}

	if s.Properties == nil {
		return NewValidationErrf("Invalid source, properties can not be null")
	}

	var err error
	switch s.SourceType {
	case api.SOURCETYPE_COMMON:
		err = s.validateSourceTypeCommon()
	case api.SOURCETYPE_VMWARE:
		err = s.validateSourceTypeVMware()
	}

	if err != nil {
		return err
	}

	return nil
}

func (s Source) validateSourceTypeCommon() error {
	var v any
	err := json.Unmarshal(s.Properties, &v)
	if err != nil {
		return NewValidationErrf("Invalid properties for common type: %v", err)
	}

	return nil
}

func (s Source) validateSourceTypeVMware() error {
	var properties api.VMwareProperties

	err := json.Unmarshal(s.Properties, &properties)
	if err != nil {
		return NewValidationErrf("Invalid properties for VMware type: %v", err)
	}

	_, err = url.Parse(properties.Endpoint)
	if err != nil {
		return NewValidationErrf("Invalid source, endpoint %q is not a valid URL: %v", properties.Endpoint, err)
	}

	if properties.Username == "" {
		return NewValidationErrf("Invalid source, username can not be empty for source type VMware")
	}

	if properties.Password == "" {
		return NewValidationErrf("Invalid source, password can not be empty for source type VMware")
	}

	return nil
}

type Sources []Source
