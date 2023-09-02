package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/atlasgurus/rulestone/api"
	"github.com/atlasgurus/rulestone/types"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type EventGenerator struct {
	dec *json.Decoder
}

func NewEventGenerator(input io.Reader) *EventGenerator {
	return &EventGenerator{dec: json.NewDecoder(input)}
}

func (gen *EventGenerator) Next() interface{} {
	var result interface{}
	if err := gen.dec.Decode(&result); err != nil {
		return err
	}
	return result
}

func ReadEvent(path string) (interface{}, error) {
	if f, err := os.Open(path); err != nil {
		return nil, err
	} else {
		defer f.Close()
		fileType := filepath.Ext(path)
		fileType = fileType[1:] // Remove the dot from the extension
		var result interface{}
		switch strings.ToLower(fileType) {
		case "json":
			decoder := json.NewDecoder(f)
			if err := decoder.Decode(&result); err != nil {
				return nil, fmt.Errorf("error parsing JSON:%s", err)
			} else {
				return result, nil
			}
		case "yaml", "yml":
			decoder := yaml.NewDecoder(f)
			if err := decoder.Decode(&result); err != nil {
				return nil, fmt.Errorf("error parsing YAML:%s", err)
			} else {
				return result, nil
			}
		default:
			return nil, fmt.Errorf("unsupported file type:%s", fileType)
		}
	}
}

func ReadEvents(path string, callback func(interface{}) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var result interface{}
		line := scanner.Text()

		err := json.Unmarshal([]byte(line), &result)
		if err != nil {
			return err
		}

		// Call the callback function for each object
		err = callback(result)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func ReadRuleFromFile(path string, ctx *types.AppContext) (*api.Rule, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileType := filepath.Ext(path)
	fileType = fileType[1:] // Remove the dot from the extension

	fapi := api.NewRuleApi(ctx)
	return fapi.ReadRule(f, fileType)
}

func ReadRuleFromString(rule string, format string, ctx *types.AppContext) (*api.Rule, error) {
	r := strings.NewReader(rule)
	fapi := api.NewRuleApi(ctx)
	return fapi.ReadRule(r, format)
}
