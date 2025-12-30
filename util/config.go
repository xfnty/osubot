package util

import (
	"io"
	"os"
	"fmt"
	"bufio"
	"errors"
	"encoding/json"
)

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type CredentialsConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Session string  `json:"session,omitempty"`
}

type LobbyNameConfig struct {
	Template string `json:"template"`
	AutoUpdate bool `json:"auto-update"`
}

type HostRotationConfig struct {
	Enabled bool                    `json:"enabled"`
	AllowTransfers bool             `json:"allow-transfers"`
	PrintQueueOnMatchEnd bool       `json:"print-queue-on-match-end"`
	ReportIllegalHostTransfers bool `json:"report-illegal-host-transfers"`
}

type DifficultyConstraintConfig struct {
	Enabled bool          `json:"enabled"`
	Range [2]float32      `json:"range"`
	AllowChanging bool    `json:"allow-changing"`
	AllowViolation bool   `json:"allow-violation"`
	ReportViolations bool `json:"report-violations"`
	MaxViolations int     `json:"max-violations"`
}

type AutoStartConfig struct {
	Enabled bool             `json:"enabled"`
	Delay float32            `json:"delay"`
	PrintInitialWarning bool `json:"print-initial-warning"`
}

type VotingConfig struct {
	Enabled bool               `json:"enabled"`
	StartVoteThreshold float32 `json:"start-vote-threshold"`
	SkipVoteThreshold float32  `json:"skip-vote-threshold"`
}

type Config struct {
	Server ServerConfig                             `json:"server"`
	Credentials CredentialsConfig                   `json:"credentials"`
	LobbyName LobbyNameConfig                       `json:"lobby-name"`
	HostRotation HostRotationConfig                 `json:"host-rotation"`
	DifficultyConstraint DifficultyConstraintConfig `json:"difficulty-constraint"`
	AutoStart AutoStartConfig                       `json:"auto-start"`
	Voting VotingConfig                             `json:"voting"`
}

var defaultConfig = &Config{
	Server: ServerConfig{
		Host: "irc.ppy.sh",
		Port: 6667,
	},
	LobbyName: LobbyNameConfig{
		Template: "{min}-{max}* | Auto Host Rotate",
		AutoUpdate: true,
	},
	HostRotation: HostRotationConfig{
		Enabled: true,
		PrintQueueOnMatchEnd: true,
		ReportIllegalHostTransfers: true,
	},
	DifficultyConstraint: DifficultyConstraintConfig{
		Enabled: true,
		Range: [2]float32{ 4, 6 },
		ReportViolations: true,
		MaxViolations: 3,
	},
	AutoStart: AutoStartConfig{
		Enabled: true,
		Delay: 120,
	},
	Voting: VotingConfig{
		Enabled: true,
		StartVoteThreshold: 0.75,
		SkipVoteThreshold: 0.75,
	},
}

const DefaultConfigPath = "config.json"

func LoadConfigFile(path string) (*Config, error) {
	file, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	cfg := &Config{}
	e = decoder.Decode(cfg)
	return cfg, e
}

func SaveConfig(cfg *Config, path string) error {
	file, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	return encoder.Encode(cfg)
}

func LoadConfig() (*Config, error) {
	cmdCfg, e := GetCommandLineConfig()
	if e != nil {
		fmt.Println(e)
		fmt.Println(UsageText)
		os.Exit(0)
	}
	if cmdCfg.Help {
		fmt.Println(UsageText)
		os.Exit(0)
	}

	configPath := DefaultConfigPath
	if cmdCfg.Config != "" {
		configPath = cmdCfg.Config
	}

	var pathError *os.PathError
	cfg, e := LoadConfigFile(configPath)
	if e != nil {
		if errors.As(e, &pathError) && cmdCfg.Config == "" {
			defaultConfigCopy := *defaultConfig
			for defaultConfigCopy.Credentials.Username, e = getInput("Enter IRC username: "); e != nil; {}
			for defaultConfigCopy.Credentials.Password, e = getInput("Enter IRC password: "); e != nil; {}
			cfg = &defaultConfigCopy

			if e = SaveConfig(cfg, DefaultConfigPath); e != nil {
				return nil, e
			}
		} else {
			return nil, e
		}
	}

	StdoutLogger.Printf("Loaded configuration from \"%v\"", configPath)
	return cfg, nil
}

type CommandLineConfig struct {
	Config string
	Channel string
	Help bool
}

func GetCommandLineConfig() (CommandLineConfig, error) {
	if len(os.Args) <= 1 {
		return CommandLineConfig{}, nil
	}

	cfg := CommandLineConfig{}
	
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-c" {
			if i >= len(os.Args) - 1 {
				return CommandLineConfig{}, errors.New("missing config file path")
			}
			cfg.Config = os.Args[i+1]
			i += 1
		} else if os.Args[i] == "--help" || os.Args[i] == "-h" {
			cfg.Help = true
		} else if i == len(os.Args) - 1 {
			cfg.Channel = os.Args[i]
		} else {
			return CommandLineConfig{}, fmt.Errorf("invalid argument \"%v\"", os.Args[i])
		}
	}

	return cfg, nil
}

const UsageText = `Usage: osubot [-h] [-c config] [channel]\n
Options:
    -h, --help  Print this help message.
    -c config   Specify path to configuration file. Default is \"config.json\".
    channel     Join existing lobby instead of creating a new one.
`

func getInput(prompt string) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(prompt)
	if !scanner.Scan() {
		e := scanner.Err()
		if e == nil {
			e = io.EOF
		}
		return "", e
	}
	return scanner.Text(), nil
}
