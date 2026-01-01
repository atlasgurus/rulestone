package types

import (
	"errors"
	"fmt"
	"github.com/atlasgurus/rulestone/immutable"
	"github.com/zyedidia/generic/hashmap"
	"github.com/zyedidia/generic/hashset"
	"sync"
)

type Category int32

const MinCategory = Category(0)
const MaxCategory = Category(1000000000)

type Mask int64

func (v Category) GetHash() uint64 {
	return uint64(v)
}

func (v Category) Equals(element immutable.SetElement) bool {
	return v == element.(Category)
}

type AndOrSet immutable.Set[immutable.Set[Category]]

func (set AndOrSet) GetHash() uint64 {
	return immutable.Set[immutable.Set[Category]](set).GetHash()
}

func (set AndOrSet) Equals(element immutable.SetElement) bool {
	return immutable.Set[immutable.Set[Category]](set).Equals(element)
}

func (set AndOrSet) Union(a AndOrSet) AndOrSet {
	return AndOrSet(immutable.Set[immutable.Set[Category]](set).Union(immutable.Set[immutable.Set[Category]](a)))
}

func (set AndOrSet) Each(fn func(key immutable.Set[Category])) {
	immutable.Set[immutable.Set[Category]](set).Each(fn)
}

func (set AndOrSet) Size() int {
	result := 0
	immutable.Set[immutable.Set[Category]](set).Each(func(key immutable.Set[Category]) {
		result++
	})
	return result
}

func (set AndOrSet) ToSlice() []immutable.Set[Category] {
	return immutable.Set[immutable.Set[Category]](set).ToSlice()
}

func (set AndOrSet) ToSlices() [][]Category {
	return MapSlice[immutable.Set[Category], []Category](
		immutable.Set[immutable.Set[Category]](set).ToSlice(), func(set immutable.Set[Category]) []Category {
			return set.ToSlice()
		})
}

func AndOrSetFromSlices(slices [][]Category) AndOrSet {
	return AndOrSet(SliceToSet(MapSlice(slices, func(c []Category) immutable.Set[Category] {
		return SliceToSet(c)
	})))
}

func (set AndOrSet) Difference(a AndOrSet) AndOrSet {
	return AndOrSet(immutable.Set[immutable.Set[Category]](set).Difference(immutable.Set[immutable.Set[Category]](a)))
}

func SliceToAndOrSet(s []Category) AndOrSet {
	return AndOrSet(SliceToSetOfSets[Category](s))
}

func SliceToSet[K immutable.SetElement](s []K) immutable.Set[K] {
	return *immutable.Of[K](s...)
}

func SliceToSetOfSets[K immutable.SetElement](s []K) immutable.Set[immutable.Set[K]] {
	var andSetSlice []immutable.Set[K]
	for _, v := range s {
		andSetSlice = append(andSetSlice, *immutable.Of[K](v))
	}
	return *immutable.Of[immutable.Set[K]](andSetSlice...)
}

func (set AndOrSet) ToHashSet() hashset.Set[immutable.Set[Category]] {
	return immutable.Set[immutable.Set[Category]](set).ToHashSet()
}

func NewHashSet[K immutable.SetElement]() *hashset.Set[K] {
	return hashset.New[K](0, func(a, b K) bool { return a.Equals(b) }, K.GetHash)
}

func NewHashMap[K immutable.SetElement, T any]() *hashmap.Map[K, T] {
	return hashmap.New[K, T](0, func(a, b K) bool { return a.Equals(b) }, K.GetHash)
}

var EmptyAndOrSet = AndOrSet(*immutable.Of[immutable.Set[Category]]())

func Reduce[T, M any](s []T, f func(M, T) M, initValue M) M {
	acc := initValue
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

func FilterSlice[T any](a []T, f func(T) bool) []T {
	n := make([]T, 0)
	for _, e := range a {
		if f(e) {
			n = append(n, e)
		}
	}
	return n
}

func FindFirstInSlice[T any](a []T, f func(T) bool) *T {
	for _, e := range a {
		if f(e) {
			return &e
		}
	}
	return nil
}

func MapSlice[T any, M any](a []T, f func(T) M) []M {
	n := make([]M, len(a))
	for i, e := range a {
		n[i] = f(e)
	}
	return n
}

func SliceToBoolMap[T comparable](s []T) map[T]bool {
	result := make(map[T]bool, len(s))
	for _, v := range s {
		result[v] = true
	}
	return result
}

type ErrorLog struct {
	mu     sync.Mutex
	errors []error
}

type AppContext struct {
	errLog ErrorLog
}

func (ctx *AppContext) LogError(err error) error {
	ctx.errLog.LogError(err)
	return err
}

func (ctx *AppContext) NewError(err string) error {
	result := errors.New(err)
	ctx.errLog.LogError(result)
	return result
}

func (errLog *ErrorLog) LogError(err error) {
	errLog.mu.Lock()
	defer errLog.mu.Unlock()
	errLog.errors = append(errLog.errors, err)
}

func (ctx *AppContext) Errorf(format string, a ...any) error {
	result := fmt.Errorf(format, a...)
	return ctx.LogError(result)
}

func (errLog *ErrorLog) PrintErrors() {
	errLog.mu.Lock()
	defer errLog.mu.Unlock()
	for _, err := range errLog.errors {
		fmt.Println(err)
	}
}

func (ctx *AppContext) PrintErrors() {
	ctx.errLog.PrintErrors()
}

func (ctx *AppContext) NumErrors() int {
	return len(ctx.errLog.errors)
}

func (ctx *AppContext) GetError(index int) error {
	return ctx.errLog.errors[index]
}

func NewAppContext() *AppContext {
	return &AppContext{}
}

const MaxIntSliceCapacity = 20

var intSlicePool = sync.Pool{
	New: func() interface{} {
		return make([]int, 0, MaxIntSliceCapacity) // Allocate a slice with a capacity of maxCapacity
	},
}

func GetIntSlice() []int {
	return intSlicePool.Get().([]int)
}

func PutIntSlice(v []int) {
	intSlicePool.Put(v[:0])
}
