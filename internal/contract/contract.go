package contract

type ContractConfig struct {
	Type        string   `json:"type"`
	Schema      string   `json:"schema,omitempty"`
	SchemaPath  string   `json:"schemaPath,omitempty"`
	Command     string   `json:"command,omitempty"`
	CommandArgs []string `json:"commandArgs,omitempty"`
	StrictMode  bool     `json:"strictMode,omitempty"`
}

type ContractValidator interface {
	Validate(cfg ContractConfig, workspacePath string) error
}

func NewValidator(cfg ContractConfig) ContractValidator {
	switch cfg.Type {
	case "json_schema":
		return &jsonSchemaValidator{}
	case "typescript_interface":
		return &typeScriptValidator{}
	case "test_suite":
		return &testSuiteValidator{}
	default:
		return nil
	}
}

func Validate(cfg ContractConfig, workspacePath string) error {
	validator := NewValidator(cfg)
	if validator == nil {
		return nil
	}
	return validator.Validate(cfg, workspacePath)
}
