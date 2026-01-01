package objectmap

import (
	"fmt"
	"github.com/atlasgurus/rulestone/types"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type AttrDictionaryRec struct {
	mapIndex      int
	dict          map[string]*AttrDictionaryRec
	dictIndex     []*AttrDictionaryRec
	attribute     string
	numAttributes int
	mapper        *ObjectAttributeMapper
	path          string
	Address       []int
}

// ObjectAttributeMapper object indexes object field names used in general filters.
// It then uses these indexes to create EventAttributeMap object representation that
// only store the fields referenced by the filters.  EventAttributeMap also allows
// fast access to these fields when evaluating filters.
type ObjectAttributeMapper struct {
	RootDictRec    *AttrDictionaryRec
	Ctx            *types.AppContext
	Config         MapperConfig
	objectPool     *sync.Pool
	valuesPool     *sync.Pool
	valuesPoolOnce sync.Once
}

type MapperConfig interface {
	MapScalar(interface{}) interface{}
	GetAppCtx() *types.AppContext
}

func NewObjectAttributeMapper(config MapperConfig) *ObjectAttributeMapper {
	result := ObjectAttributeMapper{Config: config, Ctx: config.GetAppCtx()}
	result.RootDictRec = &AttrDictionaryRec{dict: make(map[string]*AttrDictionaryRec), mapper: &result}
	// Global pool of *ObjectAttributeMap
	result.objectPool = &sync.Pool{
		New: func() interface{} {
			return &ObjectAttributeMap{
				Values: make([]interface{}, 0, 100), // Assume a default capacity of 100
			}
		},
	}
	return &result
}

type ObjectAttributeMap struct {
	DictRec       *AttrDictionaryRec
	Values        []interface{}
	OriginalEvent interface{} // Store original event for empty array detection
}

type PathSegment struct {
	segment    string
	quantifier string
	index      int
}

func (mapper *ObjectAttributeMapper) newPathSegment(segment string, quantifier string, isArray bool) (*PathSegment, error) {
	switch quantifier {
	case "*", "+":
		return &PathSegment{segment: segment, quantifier: quantifier, index: 0}, nil
	case "":
		if isArray {
			// ARRAY_ELEMENT issue
			return &PathSegment{segment: segment, quantifier: quantifier, index: -2}, nil
			//return nil, mapper.Ctx.Errorf("empty quantifier or mapIndex is not allowed %s", attribute)
		} else {
			return &PathSegment{segment: segment, quantifier: quantifier, index: -1}, nil
		}
	default:
		if isArray {
			index, err := strconv.Atoi(quantifier)
			if err != nil {
				return nil, err
			} else if index < 0 {
				return nil, mapper.Ctx.Errorf("invalid mapIndex %s[%d]", segment, index)
			} else {
				return &PathSegment{segment: segment, quantifier: "", index: index}, nil
			}
		} else {
			return &PathSegment{segment: segment, quantifier: "", index: -1}, nil
		}
	}
}

func (mapper *ObjectAttributeMapper) splitSegments(path string) ([]*PathSegment, error) {
	if path == "" {
		return nil, nil
	}
	var result []*PathSegment
	var array []string
	p1 := strings.Split(path, "[")

	for _, v1 := range p1 {
		p2 := strings.Split(v1, "]")
		for _, v2 := range p2 {
			array = append(array, v2)
		}
	}

	for i := 0; i < len(array); i += 2 {
		quantifier := ""
		isArray := i+1 < len(array)
		attr := array[i]
		if strings.HasPrefix(attr, ".") {
			attr = attr[1:]
		}
		if isArray {
			quantifier = array[i+1]
			attr += "[]"
		}

		// Make sure to record "" attribute after foo[0], but not the one after foo[]
		// foo[] designates the whole array under foo, while foo[0] designates one element named ""
		// while foo[0].fred designates another element named "fred"
		if attr != "" || !strings.HasSuffix(path, "[]") {
			segment, err := mapper.newPathSegment(attr, quantifier, isArray)
			if err != nil {
				return nil, err
			}
			result = append(result, segment)
		}
	}
	return result, nil
}

func (attrMap *ObjectAttributeMap) GetAttribute(attrPath string) (interface{}, error) {
	attrAddress, err := attrMap.DictRec.AttributePathToAddress(attrPath)
	if err != nil {
		return nil, err
	}
	return attrMap.GetAttributeByAddress(attrAddress, attrMap.Values)
}

// AttributeAddress (AA) is an array of integer Values, where the Values are following alternating pattern
// of AttributeIndex(ATI) and ArrayIndex(ARI):  ATI,ARI,ATI,...
// ATI references a specific element in ObjectAttributeMap.Values, while ARI >= 0 mapIndex a nested array element.
// Given "oam ObjectAttributeMap" and "aa AttributeAddress", the attribute may be accessed as follows:
// oam.Values[aa[0]][aa[1]][aa[2]]...
// Case when ARI < 0 denote a quantifier, -1 => some, -2 => all.
// The quantifier cases produce an iterator and can not be resolved to a single attribute value.

type AttributeAddress struct {
	Address []int

	// Keep the original Path string around in case we need to issue an error message
	Path string

	// ParentParameterIndex: index into the parentParameter array pointing to the parent Address value
	// The attribute Address is a concatenation of this value and the Address above
	ParentParameterIndex int

	// FullAddress is used for matching an attribute Address with the filters that use it.
	// It may not have the actual values of array indexes when used inside "for each" loops.
	FullAddress []int
}

// AttributePathToAddress
/*
AttributePathToAddress indexes a leaf attribute name for fast lookup and efficient object representation.
LeafAttribute here denotes an attribute that is either a scalar value or an array, but not a struct.
Attributes that point to nested objects are registered together with the nested objects' attributes
using flattened dotted notation.  For example, consider the following Json structure:

{
  "person": {
    "name": "John",
    "age": 25,
    "children": [
      {
        "name": "Jane",
        "age": 3
      },
      {
        "name": "Jim",
        "age": 1
      }
    ]
  }
}

Assuming that all the structure's attributes are referenced, it will be mapped to the following attributes:
   "person.name"
   "person.age"
   "person.children[]"
Note that "person" is not mapped as it is not a leaf attribute.
Neither is "person.children.name" mapped, as we stop at the first array attribute, using [] suffix to indicate
that the value is an array and not a scalar.
*/
func (dictRec *AttrDictionaryRec) AttributePathToAddress(attrPath string) ([]int, error) {
	var result []int
	segments, err := dictRec.mapper.splitSegments(attrPath)
	if err != nil {
		return nil, err
	}

	curDictRec := dictRec
	for _, s := range segments {
		attr := s.segment
		nextDictRec, ok := curDictRec.dict[attr]
		if !ok || nextDictRec.mapIndex == -1 {
			// We have not seen this segment yet. Register it.

			// Register empty root attribute to simplify matching logic
			// mapIndex = -1 signifies a non leaf attribute that doesn't have a value stored for it.
			curDictRec.dict[""] = &AttrDictionaryRec{mapIndex: -1}
			parts := strings.Split(attr, ".")
			for i := range parts[:len(parts)-1] {
				if strings.HasSuffix(parts[i], "[]") {
					panic("Should not get here?")
					//break
				} else {
					// Register empty root attribute to simplify matching logic
					// mapIndex = -1 signifies a non leaf attribute that doesn't have a value stored for it.
					parentAttr := strings.Join(parts[0:i+1], ".")
					if _, ok := curDictRec.dict[parentAttr]; !ok {
						curDictRec.dict[strings.Join(parts[0:i+1], ".")] = &AttrDictionaryRec{mapIndex: -1}
					}
				}
			}

			var newDictRecAddr []int
			if len(curDictRec.Address) > 0 {
				newDictRecAddr = ExtendAddress(curDictRec.Address, -1, curDictRec.numAttributes)
			} else {
				newDictRecAddr = []int{curDictRec.numAttributes}
			}

			var newDictRecPath string
			if curDictRec.path == "" {
				newDictRecPath = attr
			} else {
				newDictRecPath = curDictRec.path + "." + attr
			}
			if s.index != -1 {
				nextDictRec = &AttrDictionaryRec{
					mapIndex:  curDictRec.numAttributes,
					dict:      make(map[string]*AttrDictionaryRec),
					attribute: attr,
					path:      newDictRecPath,
					Address:   newDictRecAddr}
				curDictRec.dict[attr] = nextDictRec
				curDictRec.dictIndex = append(curDictRec.dictIndex, nextDictRec)
				curDictRec.numAttributes++
				// ARRAY_ELEMENT issue
				//} else if attr != "" {
			} else {
				nextDictRec =
					&AttrDictionaryRec{
						mapIndex:  curDictRec.numAttributes,
						attribute: attr,
						path:      newDictRecPath,
						Address:   newDictRecAddr} // We do not need to save address for scalars
				curDictRec.dict[attr] = nextDictRec
				curDictRec.dictIndex = append(curDictRec.dictIndex, nextDictRec)
				curDictRec.numAttributes++
			}
		}

		curDictRec = nextDictRec

		// Now generate the Address
		result = append(result, curDictRec.mapIndex)
		if s.index >= 0 {
			// this is an array
			if s.quantifier == "" {
				result = append(result, s.index)
			} else if s.quantifier == "+" {
				result = append(result, -2)
			} else if s.quantifier == "*" {
				result = append(result, -1)
			}
		}
	}
	return result, nil
}

func (dictRec *AttrDictionaryRec) AddressToDictionaryRec(address []int) *AttrDictionaryRec {
	result := dictRec
	for i := 0; i < len(address); i += 2 {
		result = result.dictIndex[address[i]]
	}
	return result
}

func (dictRec *AttrDictionaryRec) AddressToFullPath(address []int) string {
	dr := dictRec
	result := dictRec.path
	for i := 0; i < len(address); i += 2 {
		s := address[i]
		if result == "" {
			result = dr.dictIndex[s].attribute
		} else {
			result = result + "." + dr.dictIndex[s].attribute
		}
		if i+1 < len(address) {
			result = result[0:len(result)-2] + fmt.Sprintf("[%d]", address[i+1])
		}
		dr = dr.dictIndex[s]
	}
	return result
}

func (dictRec *AttrDictionaryRec) AddressToFullAddress(address []int) []int {
	if len(dictRec.Address) > 0 {
		result := make([]int, 0, len(address)+len(dictRec.Address)+1)
		result = append(result, dictRec.Address...)
		result = append(result, -1)
		result = append(result, address...)
		return result
	} else {
		return address
	}
}

func ExtendAddress(address []int, index ...int) []int {
	result := make([]int, 0, len(address)+len(index))
	result = append(result, address...)
	result = append(result, index...)
	return result
}

func AddressMatchKey(address []int) string {
	result := ""
	for i := 0; i < len(address); i += 2 {
		if i > 0 {
			result = result + "." + fmt.Sprintf("%d", address[i])
		} else {
			result = fmt.Sprintf("%d", address[i])
		}
	}
	return result
}

func GetNestedAttributeByAddress(
	v interface{}, attrAddress []int) interface{} {
	for _, s := range attrAddress {
		switch vv := v.(type) {
		case []interface{}:
			if s < 0 {
				panic("should not happen")
			}
			if s < len(vv) {
				v = vv[s]
			} else {
				// TODO: may need to log a warning here
				return nil
			}
		case map[string]interface{}:
			panic("should not happen")
		default:
			// The caller should be able to tell if this is a nil value or no value
			// as long as we map all scalars to Operand interface types
			return nil
		}
	}
	return v
}

func (attrMap *ObjectAttributeMap) GetAttributeByAddress(attrAddress []int, frame interface{}) (interface{}, error) {
	result := GetNestedAttributeByAddress(frame, attrAddress)
	if result == nil {
		return nil, attrMap.DictRec.mapper.Ctx.Errorf(
			"attribute %s not available", attrMap.DictRec.AddressToFullPath(attrAddress))
	} else {
		return result, nil
	}
}

func (mapper *ObjectAttributeMapper) NewObjectAttributeMap() *ObjectAttributeMap {
	obj := mapper.objectPool.Get().(*ObjectAttributeMap)
	obj.DictRec = mapper.RootDictRec

	// Lazily initialize valuesPool on first use (when numAttributes is known)
	// Use sync.Once to ensure thread-safe initialization
	mapper.valuesPoolOnce.Do(func() {
		numAttrs := mapper.RootDictRec.numAttributes
		mapper.valuesPool = &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, numAttrs)
			},
		}
	})

	// Get Values slice from pool and clear it
	values := mapper.valuesPool.Get().([]interface{})
	for i := range values {
		values[i] = nil
	}
	obj.Values = values
	return obj
}

// FreeObject returns a single ObjectAttributeMap to the pool
func (mapper *ObjectAttributeMapper) FreeObject(obj *ObjectAttributeMap) {
	// Return Values slice to pool (if pool is initialized)
	if mapper.valuesPool != nil && obj.Values != nil {
		mapper.valuesPool.Put(obj.Values)
	}
	obj.Values = nil
	obj.OriginalEvent = nil
	mapper.objectPool.Put(obj)
}

func (mapper *ObjectAttributeMapper) buildObjectMap(
	path string, v interface{}, values []interface{}, dictRec *AttrDictionaryRec, attrCallback func([]int), address []int) {
	kind := reflect.ValueOf(v).Kind()
	switch kind {
	case reflect.Map:
		_, ok := dictRec.dict[path]
		if ok {
			if strings.HasSuffix(path, "[]") {
				panic("not sure if this can happen")
			} else {
				// Add separator only if Path is not empty
				if path != "" {
					path += "."
				}
				for key, value := range v.(map[string]interface{}) {
					mapper.buildObjectMap(path+key, value, values, dictRec, attrCallback, address)
				}
			}
		}
	case reflect.Slice:
		attrDictRec, ok := dictRec.dict[path+"[]"]
		if ok {
			// Create new slice with its own backing array to avoid race conditions
			newAddress := append(address[:len(address):len(address)], attrDictRec.mapIndex, 0)
			for i, elem := range v.([]interface{}) {
				newAddress[len(newAddress)-1] = i
				newValues := make([]interface{}, attrDictRec.numAttributes)
				oldList := values[attrDictRec.mapIndex]
				if oldList == nil {
					values[attrDictRec.mapIndex] = []interface{}{newValues}
				} else {
					values[attrDictRec.mapIndex] = append(oldList.([]interface{}), newValues)
				}
				mapper.buildObjectMap("", elem, newValues, attrDictRec, attrCallback, newAddress)
			}
			attrCallback(newAddress)
		}
	case reflect.Int, reflect.Int64, reflect.String, reflect.Float64, reflect.Bool, reflect.Invalid:
		attrDictRec, ok := dictRec.dict[path]
		if ok && attrDictRec.mapIndex != -1 {
			// Create new slice with its own backing array to avoid race conditions
			newAddress := append(address[:len(address):len(address)], attrDictRec.mapIndex)
			values[attrDictRec.mapIndex] = mapper.Config.MapScalar(v)
			attrCallback(newAddress)
		}
	default:
		panic("Should not get here")
	}
}

func (mapper *ObjectAttributeMapper) MapObject(v interface{}, attrCallback func([]int)) *ObjectAttributeMap {
	address := make([]int, 0, 20)
	result := mapper.NewObjectAttributeMap()
	result.OriginalEvent = v // Store original event
	mapper.buildObjectMap("", v, result.Values, result.DictRec, attrCallback, address)
	return result
}

func (attrMap *ObjectAttributeMap) GetNumElementsAtAddress(address *AttributeAddress, frames []interface{}) (int, error) {
	values, err := attrMap.GetAttributeByAddress(address.Address, frames[address.ParentParameterIndex])
	if err != nil {
		// If not found in mapped Values, check the original event for empty arrays
		// This handles the case where an array exists but is empty
		if attrMap.OriginalEvent != nil {
			arrayValue := attrMap.getValueFromOriginalEvent(address.Path)
			if arrayValue != nil {
				kind := reflect.ValueOf(arrayValue).Kind()
				if kind == reflect.Slice {
					return len(arrayValue.([]interface{})), nil
				}
			}
		}
		return 0, err
	} else {
		kind := reflect.ValueOf(values).Kind()
		if kind == reflect.Slice {
			return len(values.([]interface{})), nil
		} else {
			return 0, fmt.Errorf(
				"unexpected element value kind %d for Path %s",
				kind, attrMap.DictRec.AddressToFullPath(address.Address))
		}
	}
}

// getValueFromOriginalEvent navigates the original event using a path like "items[]"
func (attrMap *ObjectAttributeMap) getValueFromOriginalEvent(path string) interface{} {
	// Strip "[]" suffix if present
	cleanPath := strings.TrimSuffix(path, "[]")
	if cleanPath == "" {
		return attrMap.OriginalEvent
	}

	// Navigate the path segments
	current := attrMap.OriginalEvent
	segments := strings.Split(cleanPath, ".")
	for _, segment := range segments {
		if current == nil {
			return nil
		}

		// Handle map access
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[segment]
		case map[interface{}]interface{}:
			current = v[segment]
		default:
			return nil
		}
	}
	return current
}
