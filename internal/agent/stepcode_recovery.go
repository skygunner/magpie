package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"

	"github.com/tidwall/gjson"
	"github.com/tidwall/jsonc"
	"github.com/yetone/magpie/internal/edit"
)

type stepcodeRecovery struct {
	Version          int          `json:"version"`
	ConfigBefore     stepcodeTOML `json:"config_before"`
	ConfigAfter      stepcodeTOML `json:"config_after"`
	ModelBefore      *string      `json:"models_before"`
	ModelAfter       *string      `json:"models_after"`
	Wiring           *string      `json:"wiring"`
	Gone             *string      `json:"gone"`
	ConfigExisted    bool         `json:"config_existed"`
	ModelsExisted    bool         `json:"models_existed"`
	ProvidersExisted bool         `json:"providers_existed"`
}

func stepcodeStashValue(m map[string]string, key string) *string {
	if v, ok := m[key]; ok {
		return &v
	}
	return nil
}

func stepcodeModelValue(path string) (*string, error) {
	if err := stepcodeValidateModels(path); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := gjson.GetBytes(jsonc.ToJSON(b), "providers."+magpieID)
	if !v.Exists() {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal([]byte(v.Raw), &value); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	s := string(canonical)
	return &s, nil
}

func stepcodePlan(at place, path, models string) (string, error) {
	unlock, err := stashLock()
	if err != nil {
		return "", err
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		return "", err
	}
	wiring := stepcodeStashValue(m, at.key("stepcode.wiring"))
	var state stepcodeState
	if wiring != nil {
		if err := json.Unmarshal([]byte(*wiring), &state); err != nil {
			return "", err
		}
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil {
		return "", err
	}
	model, err := stepcodeModelValue(models)
	if err != nil {
		return "", err
	}
	plan := stepcodeRecovery{Version: 1, ConfigBefore: cfg, ConfigAfter: cfg, ModelBefore: model, Wiring: wiring, Gone: stepcodeStashValue(m, at.key("stepcode.scope.gone")), ConfigExisted: state.ConfigFileExisted, ModelsExisted: state.ModelsFileExisted, ProvidersExisted: state.ProvidersMapExisted}
	if wiring == nil {
		plan.ModelAfter = model
		plan.ConfigExisted, plan.ModelsExisted, plan.ProvidersExisted = true, true, true
		b, err := json.Marshal(plan)
		return string(b), err
	}
	if stepcodeValue(cfg.Provider) == magpieID {
		plan.ConfigAfter.Provider, plan.ConfigAfter.Model = state.OrigProvider, state.OrigModel
	}
	if (cfg.Scope != nil) == state.LastHasScope && (cfg.Scope == nil || slices.Equal(*cfg.Scope, state.LastScope)) {
		plan.ConfigAfter.Scope = nil
		if state.OrigHasScope {
			scope := slices.Clone(state.OrigScope)
			plan.ConfigAfter.Scope = &scope
		}
	} else if cfg.Scope != nil && len(state.AppendedScope) > 0 {
		scope := slices.DeleteFunc(slices.Clone(*cfg.Scope), func(p string) bool { return slices.Contains(state.AppendedScope, p) })
		plan.ConfigAfter.Scope = &scope
	}
	if state.OrigMagpieRaw != nil {
		var value any
		if err := json.Unmarshal([]byte(*state.OrigMagpieRaw), &value); err != nil {
			return "", err
		}
		b, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		s := string(b)
		plan.ModelAfter = &s
	}
	b, err := json.Marshal(plan)
	return string(b), err
}

var stepcodeRecoveryLock = stashLock

func stepcodeRecover(at place, path, models string, snapshot *string) error {
	if snapshot == nil {
		return errors.New("StepCode recovery lacks a snapshot; back up configuration and stash and restore manually")
	}
	var plan stepcodeRecovery
	if json.Unmarshal([]byte(*snapshot), &plan) != nil || plan.Version != 1 {
		return errors.New("StepCode recovery lacks a supported snapshot; back up configuration and stash and restore manually")
	}
	unlock, err := stepcodeRecoveryLock()
	if err != nil {
		return err
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		return err
	}
	var marker map[string]json.RawMessage
	pendingKey := at.key("stepcode.disconnect.pending")
	if err := json.Unmarshal([]byte(m[pendingKey]), &marker); err != nil || marker == nil {
		return errors.New("StepCode pending recovery changed; user values left unchanged")
	}
	var saved string
	if err := json.Unmarshal(marker["snapshot"], &saved); err != nil || saved != *snapshot {
		return errors.New("StepCode pending recovery changed; user values left unchanged")
	}
	wiring, gone := stepcodeStashValue(m, at.key("stepcode.wiring")), stepcodeStashValue(m, at.key("stepcode.scope.gone"))
	if !(reflect.DeepEqual(wiring, plan.Wiring) && reflect.DeepEqual(gone, plan.Gone) || wiring == nil && gone == nil && string(marker["stepcode_done"]) == "true") {
		return errors.New("StepCode stash changed during disconnect; user values left unchanged")
	}
	cfg, err := readStepcodeTOML(path)
	if err != nil {
		return err
	}
	model, err := stepcodeModelValue(models)
	if err != nil {
		return err
	}
	// Each managed value may be either side of its own planned write.
	if !(reflect.DeepEqual(cfg.Provider, plan.ConfigBefore.Provider) || reflect.DeepEqual(cfg.Provider, plan.ConfigAfter.Provider)) || !(reflect.DeepEqual(cfg.Model, plan.ConfigBefore.Model) || reflect.DeepEqual(cfg.Model, plan.ConfigAfter.Model)) || !(reflect.DeepEqual(cfg.Scope, plan.ConfigBefore.Scope) || reflect.DeepEqual(cfg.Scope, plan.ConfigAfter.Scope)) || !(reflect.DeepEqual(model, plan.ModelBefore) || reflect.DeepEqual(model, plan.ModelAfter)) {
		return errors.New("StepCode settings changed during disconnect; user values left unchanged")
	}
	if plan.Wiring == nil {
		return fmt.Errorf("StepCode's original settings are missing; back up %s, %s and %s, select a non-magpie provider and model in StepCode, then remove only %q from the stash before retrying", path, models, stashPath(), pendingKey)
	}
	if !reflect.DeepEqual(cfg.Provider, plan.ConfigAfter.Provider) {
		if err := stepcodeSetString(path, "defaultProvider", plan.ConfigAfter.Provider); err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(cfg.Model, plan.ConfigAfter.Model) {
		if err := stepcodeSetString(path, "defaultModel", plan.ConfigAfter.Model); err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(cfg.Scope, plan.ConfigAfter.Scope) {
		if plan.ConfigAfter.Scope == nil {
			err = edit.DelTOMLTop(path, "enabledModels")
		} else {
			var raw edit.Raw
			raw, err = stepcodeScopeRaw(*plan.ConfigAfter.Scope)
			if err == nil {
				err = edit.SetTOMLTopPreserving(path, edit.KV{Path: "enabledModels", Value: raw})
			}
		}
		if err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(model, plan.ModelAfter) {
		if plan.ModelAfter == nil {
			err = edit.DelJSON(models, "providers."+magpieID)
		} else {
			err = edit.SetJSON(models, edit.KV{Path: "providers." + magpieID, Value: json.RawMessage(*plan.ModelAfter)})
		}
		if err != nil {
			return err
		}
	}
	if !plan.ProvidersExisted && plan.ModelAfter == nil {
		if raw, ok := edit.GetJSON(models, "providers"); ok && raw == "{}" {
			if err := edit.DelJSON(models, "providers"); err != nil {
				return err
			}
		}
	}
	if !plan.ModelsExisted {
		b, err := os.ReadFile(models)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && stepcodeJSONEmpty(b) {
			if err := edit.Remove(models); err != nil {
				return err
			}
		}
	}
	if !plan.ConfigExisted {
		b, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && len(bytes.TrimSpace(b)) == 0 {
			if err := edit.Remove(path); err != nil {
				return err
			}
		}
	}
	delete(m, at.key("stepcode.wiring"))
	delete(m, at.key("stepcode.scope.gone"))
	marker["stepcode_done"] = json.RawMessage("true")
	checkpoint, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	m[pendingKey] = string(checkpoint)
	if err := stashWrite(m); err != nil {
		return fmt.Errorf("StepCode recovery stash: %w", err)
	}
	return nil
}
