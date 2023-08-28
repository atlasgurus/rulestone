package utils

import (
	"bufio"
	"encoding/json"
	"github.com/rulestone/api"
	"github.com/rulestone/types"
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
		dec := json.NewDecoder(f)
		var result interface{}
		if err := dec.Decode(&result); err != nil {
			return nil, err
		}
		return result, nil
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
