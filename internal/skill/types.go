package skill

// SkillConfig declares an external skill with install, init, and check commands.
type SkillConfig struct {
	Install      string `yaml:"install,omitempty"`       // Command to install the skill
	Init         string `yaml:"init,omitempty"`          // Command to initialize the skill after install
	Check        string `yaml:"check,omitempty"`         // Command to verify the skill is installed
	CommandsGlob string `yaml:"commands_glob,omitempty"` // Glob pattern for skill command files (default: .claude/commands/<name>.*.md)
}
