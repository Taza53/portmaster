package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/safing/portbase/api"
	"github.com/safing/portbase/config"
	"github.com/safing/portmaster/profile"
)

// SettingsExport holds an export of settings.
type SettingsExport struct {
	Type Type `json:"type"`

	Config map[string]any `json:"config"`
}

// SettingsImportRequest is a request to import settings.
type SettingsImportRequest struct {
	ImportRequest `json:",inline"`

	// Reset all settings of target before import.
	// The ImportResult also reacts to this flag and correctly reports whether
	// any settings would be replaced or deleted.
	Reset bool `json:"reset"`

	Export *SettingsExport `json:"export"`
}

func registerSettingsAPI() error {
	if err := api.RegisterEndpoint(api.Endpoint{
		Name:        "Export Settings",
		Description: "Exports settings in a share-able format.",
		Path:        "sync/settings/export",
		Read:        api.PermitAdmin,
		Write:       api.PermitAdmin,
		Parameters: []api.Parameter{{
			Method:      http.MethodGet,
			Field:       "from",
			Description: "Specify where to export from.",
		}},
		BelongsTo: module,
		DataFunc:  handleExportSettings,
	}); err != nil {
		return err
	}

	if err := api.RegisterEndpoint(api.Endpoint{
		Name:        "Import Settings",
		Description: "Imports settings from the share-able format.",
		Path:        "sync/settings/import",
		Read:        api.PermitAdmin,
		Write:       api.PermitAdmin,
		Parameters: []api.Parameter{{
			Method:      http.MethodPost,
			Field:       "to",
			Description: "Specify where to import to.",
		}, {
			Method:      http.MethodPost,
			Field:       "validate",
			Description: "Validate only.",
		}, {
			Method:      http.MethodPost,
			Field:       "reset",
			Description: "Replace all existing settings.",
		}},
		BelongsTo:  module,
		StructFunc: handleImportSettings,
	}); err != nil {
		return err
	}

	return nil
}

func handleExportSettings(ar *api.Request) (data []byte, err error) {
	var request *ExportRequest

	// Get parameters.
	q := ar.URL.Query()
	if len(q) > 0 {
		request = &ExportRequest{
			From: q.Get("from"),
		}
	} else {
		request = &ExportRequest{}
		if err := json.Unmarshal(ar.InputData, request); err != nil {
			return nil, fmt.Errorf("%w: failed to parse export request: %s", ErrExportFailed, err)
		}
	}

	// Check parameters.
	if request.From == "" {
		return nil, errors.New("missing parameters")
	}

	// Export.
	export, err := ExportSettings(request.From)
	if err != nil {
		return nil, err
	}

	// Make some yummy yaml.
	yamlData, err := yaml.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal to yaml: %s", ErrExportFailed, err)
	}

	// TODO: Add checksum for integrity.

	return yamlData, nil
}

func handleImportSettings(ar *api.Request) (any, error) {
	var request *SettingsImportRequest

	// Get parameters.
	q := ar.URL.Query()
	if len(q) > 0 {
		request = &SettingsImportRequest{
			ImportRequest: ImportRequest{
				Target:       q.Get("to"),
				ValidateOnly: q.Has("validate"),
				RawExport:    string(ar.InputData),
			},
			Reset: q.Has("reset"),
		}
	} else {
		request = &SettingsImportRequest{}
		if err := json.Unmarshal(ar.InputData, request); err != nil {
			return nil, fmt.Errorf("%w: failed to parse import request: %s", ErrInvalidImport, err)
		}
	}

	// Check if we need to parse the export.
	switch {
	case request.Export != nil && request.RawExport != "":
		return nil, fmt.Errorf("%w: both Export and RawExport are defined", ErrInvalidImport)
	case request.RawExport != "":
		// TODO: Verify checksum for integrity.

		export := &SettingsExport{}
		if err := yaml.Unmarshal([]byte(request.RawExport), export); err != nil {
			return nil, fmt.Errorf("%w: failed to parse export: %s", ErrInvalidImport, err)
		}
		request.Export = export
	}

	// Import.
	return ImportSettings(request)
}

// ExportSettings exports the global settings.
func ExportSettings(from string) (*SettingsExport, error) {
	var settings map[string]any
	if from == ExportTargetGlobal {
		// Collect all changed global settings.
		settings = make(map[string]any)
		_ = config.ForEachOption(func(option *config.Option) error {
			v := option.UserValue()
			if v != nil {
				settings[option.Key] = v
			}
			return nil
		})
	} else {
		r, err := db.Get(profile.ProfilesDBPath + from)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to find profile: %s", ErrTargetNotFound, err)
		}
		p, err := profile.EnsureProfile(r)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to load profile: %s", ErrExportFailed, err)
		}
		settings = config.Flatten(p.Config)
	}

	// Check if there any changed settings.
	if len(settings) == 0 {
		return nil, ErrUnchanged
	}

	// Expand config to hierarchical form.
	settings = config.Expand(settings)

	return &SettingsExport{
		Type:   TypeSettings,
		Config: settings,
	}, nil
}

// ImportSettings imports the global settings.
func ImportSettings(r *SettingsImportRequest) (*ImportResult, error) {
	// Check import.
	if r.Export.Type != TypeSettings {
		return nil, ErrMismatch
	}

	// Flatten config.
	settings := config.Flatten(r.Export.Config)

	// Validate config and gather some metadata.
	var (
		result  = &ImportResult{}
		checked int
	)
	err := config.ForEachOption(func(option *config.Option) error {
		// Check if any setting is set.
		if r.Reset && option.IsSetByUser() {
			result.ReplacesExisting = true
		}

		newValue, ok := settings[option.Key]
		if ok {
			checked++

			// Validate the new value.
			if err := option.ValidateValue(newValue); err != nil {
				return fmt.Errorf("%w: configuration value for %s is invalid: %s", ErrInvalidSetting, option.Key, err)
			}

			// Collect metadata.
			if option.RequiresRestart {
				result.RestartRequired = true
			}
			if !r.Reset && option.IsSetByUser() {
				result.ReplacesExisting = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if checked < len(settings) {
		return nil, fmt.Errorf("%w: the export contains unknown settings", ErrInvalidImport)
	}

	// Import global settings.
	if r.Target == ExportTargetGlobal {
		// Stop here if we are only validating.
		if r.ValidateOnly {
			return result, nil
		}

		// Import to global config.
		vErrs, restartRequired := config.ReplaceConfig(settings)
		if len(vErrs) > 0 {
			s := make([]string, 0, len(vErrs))
			for _, err := range vErrs {
				s = append(s, err.Error())
			}
			return nil, fmt.Errorf(
				"%w: the supplied configuration could not be applied:\n%s",
				ErrImportFailed,
				strings.Join(s, "\n"),
			)
		}

		result.RestartRequired = restartRequired
		return result, nil
	}

	// Import settings into profile.
	rec, err := db.Get(profile.ProfilesDBPath + r.Target)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to find profile: %s", ErrTargetNotFound, err)
	}
	p, err := profile.EnsureProfile(rec)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load profile: %s", ErrImportFailed, err)
	}

	// FIXME: check if there are any global-only setting in the import

	// Stop here if we are only validating.
	if r.ValidateOnly {
		return result, nil
	}

	// Import settings into profile.
	if r.Reset {
		p.Config = config.Expand(settings)
	} else {
		flattenedProfileConfig := config.Flatten(p.Config)
		for k, v := range settings {
			flattenedProfileConfig[k] = v
		}
		p.Config = config.Expand(flattenedProfileConfig)
	}

	// Save profile back to db.
	err = p.Save()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to save profile: %s", ErrImportFailed, err)
	}

	return result, nil
}