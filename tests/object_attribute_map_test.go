package tests

import (
	"github.com/atlasgurus/rulestone/objectmap"
	"github.com/atlasgurus/rulestone/types"
	"github.com/atlasgurus/rulestone/utils"
	"testing"
)

type CTX struct {
	ctx *types.AppContext
}

func (ctx *CTX) MapScalar(v interface{}) interface{} {
	return v
}

func (ctx *CTX) GetAppCtx() *types.AppContext {
	return ctx.ctx
}

func TestObjectAttributeMap_01(t *testing.T) {
	ctx := &CTX{types.NewAppContext()}
	mapper := objectmap.NewObjectAttributeMapper(ctx)
	if _, err := mapper.RootDictRec.AttributePathToAddress("name"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("age"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("child.age"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("child"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("nested_array[0].foo[1]"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("nested_array[0][1]"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("nested_array[0].bar[1]"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("children[]"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("children[0].age"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("children[1].children[0].name"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}
	if _, err := mapper.RootDictRec.AttributePathToAddress("[0][1]"); err != nil {
		t.Fatalf("failed AttributePathToAddress")
	}

	if f, err := utils.ReadEvent("../examples/data/data1.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		objectMap := mapper.MapObject(f, func(addr []int) {})
		age, err := objectMap.GetAttribute("children[0].age")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if age.(float64) != 5 {
			t.Fatalf("failed: wrong age")
		}
		age, err = objectMap.GetAttribute("children[1].age")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if age.(float64) != 2 {
			t.Fatalf("failed: wrong age")
		}

		child, err := objectMap.GetAttribute("child")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if child.(string) != "Bob" {
			t.Fatalf("failed: wrong value for child attribute")
		}

		addr, err := mapper.RootDictRec.AttributePathToAddress("children[]")
		if err != nil {
			t.Fatalf("failed AttributePathToAddress")
		} else {
			address := &objectmap.AttributeAddress{
				Address:              addr,
				Path:                 "children[]",
				ParentParameterIndex: 0,
				FullAddress:          addr}
			numElems, err := objectMap.GetNumElementsAtAddress(address, []interface{}{objectMap.Values})
			if err != nil {
				t.Fatalf("failed GetNumElementsAtAddress")
			} else if numElems != 2 {
				t.Fatalf("failed: wrong number of children")
			}
		}

		name, err := objectMap.GetAttribute("children[1].children[0].name")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if name.(string) != "Davidson" {
			t.Fatalf("failed: wrong name")
		}

		name, err = objectMap.GetAttribute("children[1].children[1].name")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if name.(string) != "Junior" {
			t.Fatalf("failed: wrong name")
		}

		val, err := objectMap.GetAttribute("nested_array[0][1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 1 {
			t.Fatalf("failed: wrong name")
		}
		val, err = objectMap.GetAttribute("nested_array[1][1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 5 {
			t.Fatalf("failed: wrong name")
		}
	}

	if event, err := utils.ReadEvent("../examples/data/data2.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		objectMap := mapper.MapObject(event, func(addr []int) {})
		val, err := objectMap.GetAttribute("[0][1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 1 {
			t.Fatalf("failed: wrong name")
		}
		val, err = objectMap.GetAttribute("[1][1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 5 {
			t.Fatalf("failed: wrong name")
		}

		if ctx.ctx.NumErrors() > 0 {
			ctx.ctx.PrintErrors()
			t.Fatalf("failed due to %d errors", ctx.ctx.NumErrors())
		}
	}

	if event, err := utils.ReadEvent("../examples/data/data3.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		objectMap := mapper.MapObject(event, func(addr []int) {})
		val, err := objectMap.GetAttribute("nested_array[0].foo[1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 1 {
			t.Fatalf("failed: wrong name")
		}
		val, err = objectMap.GetAttribute("nested_array[0].bar[1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 3 {
			t.Fatalf("failed: wrong name")
		}
		val, err = objectMap.GetAttribute("nested_array[1][1]")
		if err != nil {
			t.Fatalf("failed GetAttribute")
		} else if val.(float64) != 5 {
			t.Fatalf("failed: wrong name")
		}
		if ctx.ctx.NumErrors() > 0 {
			ctx.ctx.PrintErrors()
			t.Fatalf("failed due to %d errors", ctx.ctx.NumErrors())
		}
	}
}
