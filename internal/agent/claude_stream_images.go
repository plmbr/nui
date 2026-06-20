// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
)

type extractedImage struct {
	MediaType string
	Data      string
	IsURL     bool
}

func extractImagesFromJSON(raw json.RawMessage) []extractedImage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return extractImagesFromValue(v)
}

func extractImagesFromValue(v any) []extractedImage {
	var out []extractedImage
	walkExtractImages(v, &out)
	return out
}

func walkExtractImages(v any, out *[]extractedImage) {
	switch val := v.(type) {
	case map[string]any:
		if img, ok := extractImageBlock(val); ok {
			*out = append(*out, img)
		}
		for _, child := range val {
			walkExtractImages(child, out)
		}
	case []any:
		for _, child := range val {
			walkExtractImages(child, out)
		}
	}
}

func extractImageBlock(block map[string]any) (extractedImage, bool) {
	typ, _ := block["type"].(string)
	if typ != "image" {
		return extractedImage{}, false
	}

	// MCP tool result format: {type, data, mimeType}
	if data, ok := block["data"].(string); ok && data != "" {
		mediaType, _ := block["mimeType"].(string)
		if mediaType == "" {
			mediaType, _ = block["media_type"].(string)
		}
		if mediaType == "" {
			mediaType = "image/png"
		}
		return extractedImage{MediaType: mediaType, Data: data}, true
	}

	// Anthropic API format: {type, source: {type, media_type, data, url}}
	source, _ := block["source"].(map[string]any)
	if source == nil {
		return extractedImage{}, false
	}

	srcType, _ := source["type"].(string)
	data, _ := source["data"].(string)
	url, _ := source["url"].(string)
	mediaType, _ := source["media_type"].(string)
	if mediaType == "" {
		mediaType, _ = source["mediaType"].(string)
	}

	switch {
	case (srcType == "base64" || data != "") && data != "":
		if mediaType == "" {
			mediaType = "image/png"
		}
		return extractedImage{MediaType: mediaType, Data: data}, true
	case url != "":
		return extractedImage{MediaType: mediaType, Data: url, IsURL: true}, true
	}
	return extractedImage{}, false
}

func emitImageEvents(raw json.RawMessage, events chan<- Event) {
	for _, img := range extractImagesFromJSON(raw) {
		events <- Event{
			Type:           EventImage,
			ImageMediaType: img.MediaType,
			ImageData:      img.Data,
		}
	}
}
