package immutable

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/dchest/siphash"
	"github.com/zyedidia/generic/hashset"
)

const fastHash = false

type SetElement interface {
	GetHash() uint64
	Equals(element SetElement) bool
}

type Set[K SetElement] struct {
	set  *hashset.Set[K]
	hash uint64
}

func Rehash(v uint64) uint64 {
	return HashInt(v)
}

func (s Set[K]) GetHash() uint64 {
	return s.hash
}

func (s Set[K]) Equals(element SetElement) bool {
	hashEquals := s.GetHash() == element.GetHash()
	if hashEquals {
		// TODO still need to compare values, unless cryptographic hash is used
		return hashEquals
	} else {
		return false
	}
}

func (s Set[K]) Has(val K) bool {
	return s.set.Has(val)
}

func (s Set[K]) Union(a Set[K]) Set[K] {
	result := s.set.Copy()
	a.set.Each(func(key K) {
		result.Put(key)
	})
	hash := uint64(0)
	result.Each(func(key K) {
		hash ^= Rehash(key.GetHash())
	})
	return Set[K]{result, hash}
}

func (s Set[K]) Difference(a Set[K]) Set[K] {
	result := s.set.Copy()
	a.set.Each(func(key K) {
		result.Remove(key)
	})
	hash := uint64(0)
	result.Each(func(key K) {
		hash ^= Rehash(key.GetHash())
	})
	return Set[K]{result, hash}
}

func (s Set[K]) IsSuperset(a Set[K]) bool {
	result := true
	a.set.Each(func(key K) {
		if result && !s.Has(key) {
			result = false
		}
	})
	return result
}

func (s Set[K]) Each(fn func(key K)) {
	s.set.Each(fn)
}

func (s Set[K]) Size() int {
	return s.set.Size()
}

func (s Set[K]) ToSlice() []K {
	var result []K
	s.set.Each(func(key K) {
		result = append(result, key)
	})
	return result
}

// Of returns a new hashset initialized with the given 'vals'
func Of[K SetElement](vals ...K) *Set[K] {
	var hash uint64 = 0
	set := hashset.Of[K](0, func(a, b K) bool { return a.Equals(b) }, K.GetHash, vals...)

	set.Each(func(v K) {
		hash ^= Rehash(v.GetHash())
	})

	return &Set[K]{set: set, hash: hash}
}

func FromHashSet[K SetElement](set *hashset.Set[K]) *Set[K] {
	var hash uint64 = 0

	set.Each(func(v K) {
		hash ^= Rehash(v.GetHash())
	})

	return &Set[K]{set: set, hash: hash}
}

func (s Set[K]) ToHashSet() hashset.Set[K] {
	return *hashset.Of[K](0, func(a, b K) bool { return a.Equals(b) }, K.GetHash, s.ToSlice()...)
}

func hashIntFast(v any) uint64 {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, v)
	if err != nil {
		panic("Failed byte encoding of Int")
	}
	return siphash.Hash(0, 0, buf.Bytes())
}

func hashIntCrypto(v any) uint64 {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, v)
	if err != nil {
		panic("Failed byte encoding of Int")
	}
	x := sha256.Sum256(buf.Bytes())
	return binary.LittleEndian.Uint64(x[0:8])
}

func HashInt(v any) uint64 {
	if fastHash {
		return hashIntFast(v)
	} else {
		return hashIntCrypto(v)
	}
}

func ArrayIntToInt64(v []int) []int64 {
	result := make([]int64, len(v))

	for i, v := range v {
		result[i] = int64(v)
	}
	return result
}

func HashIntArray(v []int) uint64 {
	return HashInt(ArrayIntToInt64(v))
}

func hashStringFast(v string) uint64 {
	var buf bytes.Buffer
	buf.WriteString(v)
	return siphash.Hash(0, 0, buf.Bytes())
}

func hashStringCrypto(v string) uint64 {
	var buf bytes.Buffer
	buf.WriteString(v)
	x := sha256.Sum256(buf.Bytes())
	return binary.LittleEndian.Uint64(x[0:8])
}

func HashString(v string) uint64 {
	if fastHash {
		return hashStringFast(v)
	} else {
		return hashStringCrypto(v)
	}
}
