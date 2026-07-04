// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const extHITLPrefix = "ext:"

// HITLChannelsContribution declares extension HITL delivery channels.
type HITLChannelsContribution struct {
	Source  Source         `yaml:"source"`
	Runtime *RuntimeConfig `yaml:"runtime,omitempty"`
}

// HITLChannelEntry is one HITL channel in a list file.
type HITLChannelEntry struct {
	ID          string `yaml:"id"                    json:"id"`
	DisplayName string `yaml:"displayName"           json:"displayName"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// HITLChannelRef returns the canonical channel id for an extension channel.
func HITLChannelRef(extensionName, channelID string) string {
	return extHITLPrefix + extensionName + "/" + channelID
}

func loadHITLChannelsFromFile(path string) ([]HITLChannelEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		HITLChannels []HITLChannelEntry `json:"hitlChannels" yaml:"hitlChannels"`
	}
	if err := yaml.Unmarshal(data, &wrap); err != nil {
		return nil, err
	}
	for i, ch := range wrap.HITLChannels {
		if strings.TrimSpace(ch.ID) == "" {
			return nil, fmt.Errorf("hitlChannels[%d]: id is required", i)
		}
	}
	return wrap.HITLChannels, nil
}
