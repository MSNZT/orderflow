package config

import "fmt"

type Environment string

const (
	EnvironmentProduction Environment = "production"
	EnvironmentLocal      Environment = "local"
)

func (e Environment) Validate() error {
	switch e {
	case EnvironmentLocal, EnvironmentProduction:
		return nil
	default:
		return fmt.Errorf("unsupported application environment: %q", e)
	}
}

func (e Environment) IsProduction() bool {
	return e == EnvironmentProduction
}
