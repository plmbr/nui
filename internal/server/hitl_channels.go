// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"nui/internal/extensions"
	"nui/internal/hitl"
)

type hitlChannelRef struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

func listHITLChannelRefs() []hitlChannelRef {
	out := []hitlChannelRef{{
		ID:          hitl.ChannelnuiUI,
		DisplayName: "nui UI",
		Source:      "builtin",
	}}
	if extensions.Default == nil {
		return out
	}
	for _, ext := range extensions.Default.All() {
		for _, ch := range ext.HITLChannels {
			out = append(out, hitlChannelRef{
				ID:          extensions.HITLChannelRef(ext.Manifest.Name, ch.ID),
				DisplayName: ch.DisplayName,
				Description: ch.Description,
				Source:      "extension:" + ext.Manifest.Name,
			})
		}
	}
	return out
}
