package condition

import (
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/atlasgurus/rulestone/immutable"
	"github.com/atlasgurus/rulestone/objectmap"
	"github.com/atlasgurus/rulestone/types"
	"math"
	"reflect"
	"strconv"
	"time"
)

// mixUint64 is the splitmix64 finalizer.  Hashes derived from raw bit patterns
// (see FloatOperand.GetHash) carry most of their entropy in the high bits, but
// the hashmap and hashset used for interning and lookup pick a bucket with
// hash & (capacity-1).  Mixing spreads the entropy over the low bits so that
// values like 0.5, 1 and 100 do not all land in the same bucket.
func mixUint64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// conditionsEqual compares two condition slices element by element, in order.
func conditionsEqual(a, b []Condition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}

// operandsEqual compares two operand slices element by element, in order.
func operandsEqual(a, b []Operand) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equals(b[i]) {
			return false
		}
	}
	return true
}

// intsEqual compares two address slices element by element, in order.
func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type CondKind int8

const (
	AndCondKind      CondKind = 1
	OrCondKind                = 2
	NotCondKind               = 3
	CategoryCondKind          = 4
	CompareCondKind           = 5
	ExprCondKind              = 6
	ErrorCondKind             = 7
)

type Condition interface {
	immutable.SetElement
	GetOperands() []Condition
	GetKind() CondKind
}

type AndCond struct {
	Operands []Condition
	Hash     uint64
}

func FindFirstError(cond []Condition) *Condition {
	return types.FindFirstInSlice(cond, func(c Condition) bool {
		return c.GetKind() == ErrorCondKind
	})
}

func FirstError(cond []Condition) Condition {
	result := FindFirstError(cond)
	if result == nil {
		return nil
	}
	return *result
}

func NewAndCond(cond ...Condition) Condition {
	err := FirstError(cond)
	if err != nil {
		return err
	}
	return &AndCond{Operands: cond, Hash: computeCondHash(AndCondKind, cond)}
}

func computeCondHash(kind CondKind, conditions []Condition) uint64 {
	return immutable.HashInt(append([]uint64{uint64(kind)},
		types.MapSlice(conditions, func(c Condition) uint64 {
			return c.GetHash()
		})...))
}

func (c *AndCond) GetOperands() []Condition {
	return c.Operands
}

func (c *AndCond) GetKind() CondKind {
	return AndCondKind
}

func (c *AndCond) GetHash() uint64 {
	return c.Hash
}

func (c *AndCond) Equals(v immutable.SetElement) bool {
	o, ok := v.(*AndCond)
	return ok && conditionsEqual(c.Operands, o.Operands)
}

type OrCond struct {
	Operands []Condition
	Hash     uint64
}

func NewOrCond(cond ...Condition) Condition {
	err := FirstError(cond)
	if err != nil {
		return err
	}
	return &OrCond{Operands: cond, Hash: computeCondHash(OrCondKind, cond)}
}

func (c *OrCond) GetOperands() []Condition {
	return c.Operands
}

func (c *OrCond) GetKind() CondKind {
	return OrCondKind
}

func (c *OrCond) GetHash() uint64 {
	return c.Hash
}

func (c *OrCond) Equals(v immutable.SetElement) bool {
	o, ok := v.(*OrCond)
	return ok && conditionsEqual(c.Operands, o.Operands)
}

type NotCond struct {
	Operand Condition
	Hash    uint64
}

func NewNotCond(cond Condition) Condition {
	if cond.GetKind() == ErrorCondKind {
		return cond
	}
	return &NotCond{Operand: cond, Hash: computeCondHash(NotCondKind, []Condition{cond})}
}

func (c *NotCond) GetOperands() []Condition {
	return []Condition{c.Operand}
}

func (c *NotCond) GetKind() CondKind {
	return NotCondKind
}

func (c *NotCond) GetHash() uint64 {
	return c.Hash
}

func (c *NotCond) Equals(v immutable.SetElement) bool {
	o, ok := v.(*NotCond)
	return ok && c.Operand.Equals(o.Operand)
}

type CategoryCond struct {
	Cat  types.Category
	Hash uint64
}

func NewCategoryCond(cat types.Category) Condition {
	return &CategoryCond{Cat: cat, Hash: immutable.HashInt([]int32{CategoryCondKind, int32(cat)})}
}

func (c *CategoryCond) GetOperands() []Condition {
	panic("GetOperands is not defined for Category condition")
}

func (c *CategoryCond) GetKind() CondKind {
	return CategoryCondKind
}

func (c *CategoryCond) GetHash() uint64 {
	return c.Hash
}

func (c *CategoryCond) Equals(v immutable.SetElement) bool {
	o, ok := v.(*CategoryCond)
	return ok && c.Cat == o.Cat
}

type ErrorCondition struct {
	Err  error
	Hash uint64
}

func NewErrorCondition(val error) Condition {
	return &ErrorCondition{
		Err:  val,
		Hash: immutable.HashInt([]uint64{uint64(ErrorCondKind), immutable.HashString(val.Error())})}
}

func (c *ErrorCondition) GetHash() uint64 {
	return c.Hash
}

func (c *ErrorCondition) Equals(v immutable.SetElement) bool {
	o, ok := v.(*ErrorCondition)
	return ok && c.Err.Error() == o.Err.Error()
}

func (c *ErrorCondition) GetOperands() []Condition {
	panic("GetOperands is not defined for ErrorCondition condition")
}

func (c *ErrorCondition) GetKind() CondKind {
	return ErrorCondKind
}

type ExprCondition struct {
	Expr string
	Hash uint64
}

func NewExprCondition(val string) Condition {
	return &ExprCondition{
		Expr: val,
		Hash: immutable.HashInt([]uint64{uint64(ExprCondKind), immutable.HashString(val)})}
}

func (c *ExprCondition) GetHash() uint64 {
	return c.Hash
}

func (c *ExprCondition) Equals(v immutable.SetElement) bool {
	o, ok := v.(*ExprCondition)
	return ok && c.Expr == o.Expr
}

func (c *ExprCondition) GetOperands() []Condition {
	panic("GetOperands is not defined for ExprCondition condition")
}

func (c *ExprCondition) GetKind() CondKind {
	return ExprCondKind
}

type CompareOp uint8

const (
	CompareEqualOp          CompareOp = 1
	CompareNotEqualOp                 = 2
	CompareGreaterOp                  = 3
	CompareGreaterOrEqualOp           = 4
	CompareLessOp                     = 5
	CompareLessOrEqualOp              = 6
	CompareContainsOp                 = 7
	CompareInvalidOp                  = 99
)

type OperandKind uint8

const (
	StringOperandKind     OperandKind = 1
	IntOperandKind                    = 2
	FloatOperandKind                  = 3
	BooleanOperandKind                = 4
	TimeOperandKind                   = 5
	AttributeOperandKind              = 6
	ExpressionOperandKind             = 7
	AddressOperandKind                = 8
	SelOperandKind                    = 9
	IndexOperandKind                  = 10
	NullOperandKind                   = 11
	ListOperandKind                   = 12
	ErrorOperandKind                  = 13
	UndefinedOperandKind              = 14
)

type EvalOperandFunc func(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand
type OperandEvaluator interface {
	Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand
}

type Operand interface {
	immutable.SetElement
	IsConst() bool
	GetKind() OperandKind
	Greater(o Operand) bool
	Convert(to OperandKind) Operand
	OperandEvaluator
}

type ListOperand struct {
	List []Operand
}

// NewListOperand do not compute hash on creation.  We use ListOperand at runtime and need it to be fast
func NewListOperand(list []Operand) *ListOperand {
	return &ListOperand{List: list}
}

func (v *ListOperand) Convert(to OperandKind) Operand {
	panic(fmt.Errorf("unexpected conversion of ListOperand"))
}

func (v *ListOperand) IsConst() bool {
	return false
}

func (v *ListOperand) GetKind() OperandKind {
	return ListOperandKind
}

func (v *ListOperand) GetHash() uint64 {
	return immutable.HashInt(
		[]uint64{
			uint64(ListOperandKind),
			immutable.HashInt(types.MapSlice(v.List, func(o Operand) uint64 { return o.GetHash() }))})
}

func (v *ListOperand) Equals(o immutable.SetElement) bool {
	other, ok := o.(*ListOperand)
	return ok && operandsEqual(v.List, other.List)
}

func (v *ListOperand) Greater(o Operand) bool {
	panic(fmt.Errorf("greater is not supported for ListOperand"))
}

func (v *ListOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type ExprOperand struct {
	Func EvalOperandFunc
	Args []Operand
	// FuncPtr is the code pointer of Func.  Closures created from the same
	// function literal share it, which is what makes interning possible: the
	// engine builds a fresh closure per access but expects equal ones to merge.
	FuncPtr uintptr
	Hash    uint64
}

func NewExprOperand(f EvalOperandFunc, args ...Operand) *ExprOperand {
	funcPtr := reflect.ValueOf(f).Pointer()
	return &ExprOperand{
		Func:    f,
		Args:    args,
		FuncPtr: funcPtr,
		Hash: immutable.HashInt(
			[]uint64{
				uint64(ExpressionOperandKind),
				uint64(funcPtr),
				immutable.HashInt(types.MapSlice(args, func(o Operand) uint64 { return o.GetHash() }))})}
}

func (v *ExprOperand) Convert(to OperandKind) Operand {
	panic(fmt.Errorf("unexpected conversion of ExprOperand"))
}

func (v *ExprOperand) IsConst() bool {
	return false
}

func (v *ExprOperand) GetKind() OperandKind {
	return ExpressionOperandKind
}

func (v *ExprOperand) GetHash() uint64 {
	return v.Hash
}

// Equals compares the code pointer and the arguments.  Values a closure
// captures beyond its arguments are not visible here, which is the same
// assumption the hash made; the engine only ever builds closures whose
// behaviour is fully determined by Args.
func (v *ExprOperand) Equals(o immutable.SetElement) bool {
	other, ok := o.(*ExprOperand)
	return ok && v.FuncPtr == other.FuncPtr && operandsEqual(v.Args, other.Args)
}

func (v *ExprOperand) Greater(o Operand) bool {
	panic(fmt.Errorf("greater is not supported for ExprOperand"))
}

func (v *ExprOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v.Func(event, frames)
}

type IntOperand int64

func NewIntOperand(val int64) Operand {
	return IntOperand(val)
}

func (v IntOperand) Convert(to OperandKind) Operand {
	switch to {
	case TimeOperandKind:
		return NewTimeOperand(time.Unix(0, int64(v)))
	case IntOperandKind:
		return v
	case FloatOperandKind:
		return NewFloatOperand(float64(v))
	case BooleanOperandKind:
		return NewBooleanOperand(v != 0)
	case StringOperandKind:
		return NewStringOperand(strconv.Itoa(int(v)))
	case NullOperandKind:
		return NewNullOperand(nil)
	default:
		panic(fmt.Errorf("Unexpected conversion to %d ", to))
	}
}

func (v IntOperand) IsConst() bool {
	return true
}

func (v IntOperand) GetKind() OperandKind {
	return IntOperandKind
}

func (v IntOperand) GetHash() uint64 {
	return uint64(v)
}

func (v IntOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == IntOperandKind && v == o.(IntOperand)
}

func (v IntOperand) Greater(o Operand) bool {
	return v > o.(IntOperand)
}

func (v IntOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type FloatOperand float64

func (v FloatOperand) Convert(to OperandKind) Operand {
	switch to {
	case TimeOperandKind:
		return NewTimeOperand(time.Unix(0, int64(v)))
	case IntOperandKind:
		return NewIntOperand(int64(v))
	case FloatOperandKind:
		return v
	case BooleanOperandKind:
		return NewBooleanOperand(v != 0)
	case StringOperandKind:
		return NewStringOperand(strconv.FormatFloat(float64(v), 'g', -1, 64))
	case NullOperandKind:
		return NewNullOperand(nil)
	default:
		panic(fmt.Errorf("Unexpected conversion to %d ", to))
	}
}

func NewFloatOperand(val float64) Operand {
	var result Operand = FloatOperand(val)
	return result
}

func (v FloatOperand) IsConst() bool {
	return true
}

func (v FloatOperand) GetKind() OperandKind {
	return FloatOperandKind
}

// GetHash hashes the full IEEE-754 bit pattern.  Converting to uint64 instead
// would drop the fraction, giving every constant in [0,1) the same hash, and
// would be implementation-defined for negative and out-of-range values.
func (v FloatOperand) GetHash() uint64 {
	f := float64(v)
	if f == 0 {
		// -0 == +0, so the two must hash alike.
		f = 0
	}
	return mixUint64(math.Float64bits(f) ^ uint64(FloatOperandKind)<<56)
}

func (v FloatOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == FloatOperandKind && v == o.(FloatOperand)
}

func (v FloatOperand) Greater(o Operand) bool {
	return v > o.(FloatOperand)
}

func (v FloatOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type StringOperand string

func NewStringOperand(val string) Operand {
	return StringOperand(val)
}

func (v StringOperand) Convert(to OperandKind) Operand {
	switch to {
	case TimeOperandKind:
		t, err := dateparse.ParseAny(string(v))
		if err != nil {
			return NewErrorOperand(err)
		}
		return NewTimeOperand(t)
	case IntOperandKind:
		i, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return NewErrorOperand(fmt.Errorf("Err %s converting %s to IntOperand", err.Error(), string(v)))
		}
		return NewIntOperand(i)
	case FloatOperandKind:
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return NewErrorOperand(fmt.Errorf("Err %s converting %s to IntOperand", err.Error(), string(v)))
		}
		return NewFloatOperand(f)
	case BooleanOperandKind:
		return NewBooleanOperand(v != "true")
	case StringOperandKind:
		return v
	case NullOperandKind:
		return NewNullOperand(nil)
	default:
		panic(fmt.Errorf("unexpected conversion to %d ", to))
	}
}

func (v StringOperand) IsConst() bool {
	return true
}

func (v StringOperand) GetKind() OperandKind {
	return StringOperandKind
}

func (v StringOperand) GetHash() uint64 {
	return immutable.HashString(string(v))
}

func (v StringOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == StringOperandKind && v == o.(StringOperand)
}

func (v StringOperand) Greater(o Operand) bool {
	return v > o.(StringOperand)
}

func (v StringOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type TimeOperand time.Time

func NewTimeOperand(val time.Time) Operand {
	return TimeOperand(val)
}

func (v TimeOperand) Convert(to OperandKind) Operand {
	switch to {
	case IntOperandKind:
		return NewIntOperand(time.Time(v).UnixNano())
	case FloatOperandKind:
		return NewFloatOperand(float64(time.Time(v).UnixNano()))
	case BooleanOperandKind:
		panic(fmt.Errorf("unexpected conversion to %d ", to))
	case TimeOperandKind:
		return v
	case StringOperandKind:
		return NewStringOperand(time.Time(v).Format(time.RFC3339Nano))
	case NullOperandKind:
		return NewNullOperand(nil)
	default:
		panic(fmt.Errorf("unexpected conversion to %d ", to))
	}
}

func (v TimeOperand) IsConst() bool {
	return true
}

func (v TimeOperand) GetKind() OperandKind {
	return TimeOperandKind
}

func (v TimeOperand) GetHash() uint64 {
	return immutable.HashInt(time.Time(v).UnixNano())
}

func (v TimeOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == TimeOperandKind && time.Time(v).Equal(time.Time(o.(TimeOperand)))
}

func (v TimeOperand) Greater(o Operand) bool {
	return time.Time(v).After(time.Time(o.(TimeOperand)))
}

func (v TimeOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type BooleanOperand bool

var IntConst1 IntOperand = 1
var IntConst0 IntOperand = 0

var FloatConst1 FloatOperand = 1
var FloatConst0 FloatOperand = 0

func (v BooleanOperand) Convert(to OperandKind) Operand {
	switch to {
	case IntOperandKind:
		if v {
			return IntConst1
		} else {
			return IntConst0
		}
	case FloatOperandKind:
		if v {
			return FloatConst1
		} else {
			return FloatConst0
		}
	case BooleanOperandKind:
		return v
	case StringOperandKind:
		return NewStringOperand(strconv.FormatBool(bool(v)))
	case NullOperandKind:
		return NewNullOperand(nil)
	default:
		panic(fmt.Errorf("unexpected conversion to %d ", to))
	}
}

var trueBooleanOperand BooleanOperand = true
var falseBooleanOperand BooleanOperand = false

func NewBooleanOperand(val bool) BooleanOperand {
	if val {
		return trueBooleanOperand
	} else {
		return falseBooleanOperand
	}
}

func (v BooleanOperand) IsConst() bool {
	return true
}

func (v BooleanOperand) GetKind() OperandKind {
	return BooleanOperandKind
}

func (v BooleanOperand) GetHash() uint64 {
	// Use a large offset to avoid hash collisions with integers
	// BooleanOperandKind is 4, use 4 << 32 as base offset
	const boolHashOffset = uint64(BooleanOperandKind) << 32
	if v {
		return boolHashOffset + 1
	} else {
		return boolHashOffset + 0
	}
}

func (v BooleanOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == BooleanOperandKind && v == o.(BooleanOperand)
}

func (v BooleanOperand) Greater(o Operand) bool {
	return bool(v && !o.(BooleanOperand))
}

func (v BooleanOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type AttributeOperand struct {
	AttributePath string
}

func (v *AttributeOperand) Convert(to OperandKind) Operand {
	panic("Conversion of AttributeOperand is not supported")
}

func NewAttributeOperand(attributePath string) *AttributeOperand {
	return &AttributeOperand{AttributePath: attributePath}
}

func (v *AttributeOperand) Greater(o Operand) bool {
	panic("Comparison of AttributeOperand not supported")
}

func (v *AttributeOperand) IsConst() bool {
	return false
}

func (v *AttributeOperand) GetKind() OperandKind {
	return AttributeOperandKind
}

func (v *AttributeOperand) GetHash() uint64 {
	return immutable.HashString(v.AttributePath)
}

func (v *AttributeOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == AttributeOperandKind &&
		v.AttributePath == o.(*AttributeOperand).AttributePath
}

func (v *AttributeOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	panic("implement me")
}

type AddressOperand struct {
	Address        []int
	FullAddress    []int
	ParameterIndex int
	ExprOperand    *ExprOperand
	Hash           uint64
}

func NewAddressOperand(address []int, fullAddress []int, parameterIndex int, exprOperand *ExprOperand) *AddressOperand {
	return &AddressOperand{
		Address:        address,
		FullAddress:    fullAddress,
		ParameterIndex: parameterIndex,
		ExprOperand:    exprOperand,
		Hash: immutable.HashInt(
			[]uint64{uint64(AddressOperandKind),
				immutable.HashIntArray(address),
				immutable.HashIntArray(fullAddress),
				uint64(parameterIndex),
				uint64(reflect.ValueOf(exprOperand).Pointer())}),
	}
}

func (v *AddressOperand) Convert(to OperandKind) Operand {
	panic("Conversion of AddressOperand is not supported")
}

func (v *AddressOperand) Greater(o Operand) bool {
	panic("Comparison of AddressOperand not supported")
}

func (v *AddressOperand) IsConst() bool {
	return false
}

func (v *AddressOperand) GetKind() OperandKind {
	return AddressOperandKind
}

func (v *AddressOperand) GetHash() uint64 {
	return v.Hash
}

// Equals compares ExprOperand by identity rather than structurally, matching
// what the hash encodes: address operands carrying a computed expression are
// not interned, so two of them are the same operand only if they are the same
// object.
func (v *AddressOperand) Equals(o immutable.SetElement) bool {
	other, ok := o.(*AddressOperand)
	return ok &&
		v.ParameterIndex == other.ParameterIndex &&
		v.ExprOperand == other.ExprOperand &&
		intsEqual(v.Address, other.Address) &&
		intsEqual(v.FullAddress, other.FullAddress)
}

func (v *AddressOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	if v.ExprOperand == nil {
		return v
	} else {
		return v.ExprOperand.Evaluate(event, frames)
	}
}

type SelOperand struct {
	Base     Operand
	Selector string
	Hash     uint64
}

func NewSelOperand(base Operand, selector string) *SelOperand {
	baseHash := uint64(0)
	if base != nil {
		baseHash = base.GetHash()
	}
	return &SelOperand{Base: base, Selector: selector,
		Hash: immutable.HashInt([]uint64{uint64(SelOperandKind), baseHash, immutable.HashString(selector)})}
}

func (v *SelOperand) Convert(to OperandKind) Operand {
	panic("Conversion of SelOperand is not supported")
}

func (v *SelOperand) Greater(o Operand) bool {
	panic("Comparison of SelOperand not supported")
}

func (v *SelOperand) IsConst() bool {
	return false
}

func (v *SelOperand) GetKind() OperandKind {
	return SelOperandKind
}

func (v *SelOperand) GetHash() uint64 {
	return v.Hash
}

func (v *SelOperand) Equals(o immutable.SetElement) bool {
	other, ok := o.(*SelOperand)
	if !ok || v.Selector != other.Selector {
		return false
	}
	if v.Base == nil || other.Base == nil {
		return v.Base == nil && other.Base == nil
	}
	return v.Base.Equals(other.Base)
}

func (v *SelOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	panic("implement me")
}

type IndexOperand struct {
	Base      Operand
	IndexExpr Operand
	Hash      uint64
}

func NewIndexOperand(base Operand, indexExpr Operand) *IndexOperand {
	return &IndexOperand{Base: base, IndexExpr: indexExpr,
		Hash: immutable.HashInt([]uint64{uint64(IndexOperandKind), base.GetHash(), indexExpr.GetHash()})}
}

func (v *IndexOperand) Convert(to OperandKind) Operand {
	panic("Conversion of IndexOperand is not supported")
}

func (v *IndexOperand) Greater(o Operand) bool {
	panic("Comparison of IndexOperand not supported")
}

func (v *IndexOperand) IsConst() bool {
	return false
}

func (v *IndexOperand) GetKind() OperandKind {
	return IndexOperandKind
}

func (v *IndexOperand) GetHash() uint64 {
	return v.Hash
}

func (v *IndexOperand) Equals(o immutable.SetElement) bool {
	other, ok := o.(*IndexOperand)
	return ok && v.Base.Equals(other.Base) && v.IndexExpr.Equals(other.IndexExpr)
}

func (v *IndexOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	panic("implement me")
}

type ErrorOperand struct {
	Err error
}

func (v ErrorOperand) Error() string {
	return v.Err.Error()
}

func (v ErrorOperand) Convert(to OperandKind) Operand {
	// Can't convert Err to anything.  Return Err
	// Could capture a nested Err here
	return v
}

func NewErrorOperand(val error) Operand {
	return ErrorOperand{val}
}

func (v ErrorOperand) IsConst() bool {
	return true
}

func (v ErrorOperand) GetKind() OperandKind {
	return ErrorOperandKind
}

func (v ErrorOperand) GetHash() uint64 {
	return immutable.HashString(v.Error())
}

func (v ErrorOperand) Equals(o immutable.SetElement) bool {
	return false
}

func (v ErrorOperand) Greater(o Operand) bool {
	return false
}

func (v ErrorOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type NullOperand struct {
	address *AddressOperand
}

func (v NullOperand) Convert(to OperandKind) Operand {
	// Can't convert null to anything.  Return error.
	return NewErrorOperand(fmt.Errorf("invalid conversion of null operand to %d", to))
}

func NewNullOperand(val *AddressOperand) Operand {
	return NullOperand{val}
}

func (v NullOperand) IsConst() bool {
	return true
}

func (v NullOperand) GetKind() OperandKind {
	return NullOperandKind
}

func (v NullOperand) GetHash() uint64 {
	return 0
}

func (v NullOperand) Equals(o immutable.SetElement) bool {
	return o.(Operand).GetKind() == NullOperandKind
}

func (v NullOperand) Greater(o Operand) bool {
	return false
}

func (v NullOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

// UndefinedOperand represents a missing field (distinct from explicit null)
type UndefinedOperand struct {
	Source interface{} // For debugging - stores the field path or other context
}

func NewUndefinedOperand(source interface{}) Operand {
	return UndefinedOperand{Source: source}
}

func (v UndefinedOperand) Convert(to OperandKind) Operand {
	// undefined propagates through most conversions
	// Exception: conversion to boolean could be false, but we preserve undefined semantics
	switch to {
	case UndefinedOperandKind:
		return v
	case NullOperandKind:
		// undefined cannot convert to null (different types)
		return NewErrorOperand(fmt.Errorf("invalid conversion of undefined operand to null"))
	default:
		// For other types, propagate undefined
		return v
	}
}

func (v UndefinedOperand) IsConst() bool {
	return true
}

func (v UndefinedOperand) GetKind() OperandKind {
	return UndefinedOperandKind
}

func (v UndefinedOperand) GetHash() uint64 {
	// Use a unique hash different from null (0)
	return 1
}

func (v UndefinedOperand) Equals(o immutable.SetElement) bool {
	// undefined == undefined → true (for set operations)
	return o.(Operand).GetKind() == UndefinedOperandKind
}

func (v UndefinedOperand) Greater(o Operand) bool {
	// undefined is not orderable - this should not be called
	// If it is, we could panic or return false
	panic("undefined operand is not orderable")
}

func (v UndefinedOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
	return v
}

type CompareCondition struct {
	CompareOp    CompareOp
	LeftOperand  Operand
	RightOperand Operand
	Hash         uint64
}

func NewCompareCond(op CompareOp, l Operand, r Operand) *CompareCondition {
	return &CompareCondition{
		CompareOp:    op,
		LeftOperand:  l,
		RightOperand: r,
		Hash:         immutable.HashInt([]uint64{l.GetHash(), r.GetHash(), uint64(op)})}
}

func (cond *CompareCondition) GetOperands() []Condition {
	return nil
}

func (cond *CompareCondition) GetKind() CondKind {
	return CompareCondKind
}

func (cond *CompareCondition) GetHash() uint64 {
	return cond.Hash
}

// Equals compares the operator and both operands.  Comparing the stored hashes
// instead would make any hash collision a semantic identity, merging distinct
// comparisons such as x >= 0.95 and x >= 0.97 into one.
func (cond *CompareCondition) Equals(v immutable.SetElement) bool {
	o, ok := v.(*CompareCondition)
	return ok &&
		cond.CompareOp == o.CompareOp &&
		cond.LeftOperand.Equals(o.LeftOperand) &&
		cond.RightOperand.Equals(o.RightOperand)
}

type ForEach interface {
	GetElement() string
	GetPath() string
	GetCond() Condition
	GetHash() uint64
	Equals(element immutable.SetElement) bool
}

func NewInterfaceOperand(v interface{}, ctx *types.AppContext) Operand {
	switch n := v.(type) {
	case nil:
		return NewNullOperand(nil)
	case int:
		// Convert int to float64 for consistency with JSON decoding and rule parser behavior
		// All numeric literals in rules are parsed as FloatOperand, so events should match
		return NewFloatOperand(float64(n))
	case int64:
		// Convert int64 to float64 for consistency with JSON decoding and rule parser behavior
		return NewFloatOperand(float64(n))
	case string:
		return NewStringOperand(n)
	case float64:
		return NewFloatOperand(n)
	case bool:
		return NewBooleanOperand(n)
	case time.Time:
		return NewTimeOperand(n)
	case *time.Time:
		if n == nil {
			return NewNullOperand(nil)
		}
		return NewTimeOperand(*n)
	case map[string]interface{}:
		return NewErrorOperand(ctx.Errorf("scalar operand expected got map: %v", v))
	case []interface{}:
		return NewErrorOperand(ctx.Errorf("scalar operand expected got slice: %v", v))
	default:
		panic("Should not get here")
	}
}

// ReconcileOperands TODO: may need to add reconcile kind, e.g. compare, arithmetic, string, etc.
func ReconcileOperands(x, y Operand) (Operand, Operand) {
	xkind := x.GetKind()
	ykind := y.GetKind()

	// Don't convert between boolean and numeric types
	if (xkind == BooleanOperandKind && (ykind == IntOperandKind || ykind == FloatOperandKind)) ||
		(ykind == BooleanOperandKind && (xkind == IntOperandKind || xkind == FloatOperandKind)) {
		return x, y // Return without conversion
	}

	if xkind < ykind {
		return x.Convert(y.GetKind()), y
	} else if xkind > ykind {
		return x, y.Convert(x.GetKind())
	} else {
		return x, y
	}
}
