package settings

// CooldownPolicy contains optional overrides. Nil inherits; explicit false/zero
// can disable a switch/model without losing the parent's other settings.
type CooldownPolicy struct {
	RateStart     *int           `json:"rateStart,omitempty"`
	RateCap       *int           `json:"rateCap,omitempty"`
	SlowStart     *int           `json:"slowStart,omitempty"`
	SlowThreshold *int           `json:"slowThreshold,omitempty"`
	SlowCap       *int           `json:"slowCap,omitempty"`
	FailureCap    *int           `json:"failureCap,omitempty"`
	QuotaRest     *int           `json:"quotaRest,omitempty"`
	QuotaResetAt  *string        `json:"quotaResetAt,omitempty"`
	FailRest      *bool          `json:"failRest,omitempty"`
	LimitRest     *bool          `json:"limitRest,omitempty"`
	Models        map[string]int `json:"models,omitempty"`
}

func SetCooldown(policy CooldownPolicy) error {
	defer lockFile()()
	s, err := loadChecked()
	if err != nil {
		return err
	}
	s.Cooldown = policy
	return save(s, false)
}
