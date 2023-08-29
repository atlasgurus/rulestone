# Rulestone #

Lightweight and fast [rule engine](https://en.wikipedia.org/wiki/Business_rules_engine) written in Go, with Go and Java API available.

With Rulestone you can define thousands of rules and then process tens of thousands events/objects per second getting the matching rules
for each object.

# Installation

## Go

Install the package:

```bash
go get github.com/atlasgurus/rulestone
```

## Java

Add Maven dependency:

```xml
<dependency>
    <groupId>com.atlasgurus</groupId>
    <artifactId>rulestone</artifactId>
    <version>0.1.0</version>
</dependency>
```

# Usage

## Go

The following Go example shows how to load a rule from file and match an object against this rule:

```go
    package main
    
    import (
        "fmt"
        "github.com/rulestone/Utils"
        "github.com/rulestone/api"
        "github.com/rulestone/engine"
        "github.com/rulestone/types"
    )
    
    func Match() {
        // Create new application context to keep track of errors and other info
        ctx := types.NewAppContext()
        
        repo := engine.NewRuleEngineRepo(ctx)
        _, err := repo.RegisterRuleFromFile("rule.json")
        if err != nil {
            return
        }
        
        ruleEngine, err := engine.NewRuleEngine(repo)
        if err != nil {
            return
        }
        
        event, err := utils.ReadEvent("object.json")
        if err != nil {
            return
        }

        matches := ruleEngine.MatchEvent(event)

        for _, ruleId := range matches {
			// Optionally get matching rules metadata
            ruleDefinition := ruleEngine.GetRuleDefinition(ruleId)
            if ruleIdStr, ok := ruleDefinition.Metadata["rule_id"].(string); ok {
                fmt.Println("Rule matched: ", ruleIdStr)
            }
        }
        // Report all the errors if any
        if ctx.NumErrors() > 0 {
            ctx.PrintErrors()
        }
    }
    
    func main() {
        Match()
    }
```

This example assumes rule contains the metadata field called `rule_id`.
See for more Go usage examples in `go/tests/rule_api_test.go`.

## Java

The following Java example shows how to load a rule from file and match an object against this rule:

```java
import com.atlasgurus.rulestone.RuleMetadata;
import com.atlasgurus.rulestone.Rulestone;

public class RulestoneMatch {
    public static void main(String[] args) {
        String json = "{\"field1\": \"value1\"}";
        Rulestone rulestoneEngine = Rulestone.getInstance("/rules_directory");
        int[] matches = rulestoneEngine.match(json);
        for (int ruleId: matches) {
            RuleMetadata metadata = rulestoneEngine.getRuleMetadata(ruleId);
            System.out.println("Rule matched: " + metadata.getValue("rule_id"));
        }
    }
}
```

This example assumes rule contains the metadata field called `rule_id`.


## Rules

### Simple rule definition

```json
{
  "metadata": {
    "created": "2023-03-29",
    "priority": 10,
    "rule_id": "BUSINESS_RULE_1"
  },
  "condition": {
    "expression": "name == \"Frank\" && age == 20"
  }
}
```

The rule match if the JSON object has a `name` field with value `Frank` and an `age` field with value `20`.
The metadata section may store any information linked to the rule, including the rule ID. 
The condition section contains the expression that will be evaluated against the JSON object.s

See `go/examples/rules` for more rules examples.

### Operators and functions

Rulestone expressions supports:
* Comparison and negation operators like `==`, `>`, `>=`, `<`, `<=`
* Arithmetic operations: `+`, `-`, `*`, `/`
* Logical operators: `&&`, `||`, `!`
* Parentheses: `(`, `)`
* String literals: `"string"`
* Numeric literals: `1`, `2.3`
* Field access: `field1`, `field1.field2`
* Functions: `hasValue`, `isEqualToAny`, `regexpMatch`, `date`
* Date literals: `date("11/29/1968")`
* Date comparison operators: `<`, `<=`, `>`, `>=`, `==`
* Date arithmetic operations: `+`, `-`


* `hasValue` - check that object has specified field, for example `hasValue(field1)`
* `isEqualToAny` - check that object field is equal to any specified value, for example `isEqualToAny(field1, 1, 2, 3, '4')`
* `regexpMatch` - match the Go regexp, for example `regexpMatch("^\\d{4}/\\d{2}/\\d{2}$", child.dob)`

### Dates

Rulestone handles dates and comparison operators on them, but since JSON doesn't provide field type information,
need to use `date()` function. The function can parse date string in different formats:

```json
{
  "metadata": {
    "created": "2023-03-29",
    "priority": 10
  },
  "condition": {
    "expression": "name == \"Frank\" && date(dob) < date(child.dob) && date(\"11/29/1968\") > date(dob) && date(dob) == date(\"11/28/1968\")"
  }
}
```

## Contributing
We love contributions! If you have any suggestions, bug reports, or feature requests, please open an issue on our [GitHub page]().

## License
This project is licensed under the MIT License - see the LICENSE file for details.








