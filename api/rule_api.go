package api

import (
	"encoding/json"
	c "github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

type RuleApi struct {
	ctx *types.AppContext
}

type RuleDefinition struct {
	Metadata  map[string]interface{}
	Condition c.Condition
}

type Rule struct {
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Expression string                 `json:"expression"`
}

func (api *RuleApi) ReadRule(r io.Reader, fileType string) (*Rule, error) {
	var result Rule

	switch strings.ToLower(fileType) {
	case "json":
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&result); err != nil {
			return nil, api.ctx.Errorf("error parsing JSON:%s", err)
		}
	case "yaml", "yml":
		decoder := yaml.NewDecoder(r)
		if err := decoder.Decode(&result); err != nil {
			return nil, api.ctx.Errorf("error parsing YAML:%s", err)
		}
	default:
		return nil, api.ctx.Errorf("unsupported file type:%s", fileType)
	}

	return &result, nil
}

func (api *RuleApi) RuleToRuleDefinition(rule *Rule) (*RuleDefinition, error) {
	var result RuleDefinition
	firstError := api.ctx.NumErrors()

	if rule.Expression != "" {
		result = RuleDefinition{
			Metadata:  rule.Metadata,
			Condition: c.NewExprCondition(rule.Expression)}
	}

	if api.ctx.NumErrors() > firstError {
		// Return the first error if errors found
		return nil, api.ctx.GetError(firstError)
	}
	return &result, nil
}

func NewRuleApi(ctx *types.AppContext) *RuleApi {
	return &RuleApi{ctx: ctx}
}