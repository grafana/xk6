package minhashlsh

import (
	"encoding/binary"
	"math"
	"sort"
)

const (
	integrationPrecision = 0.01
)

type hashKeyFunc func([]uint64) string

func hashKeyFuncGen(hashValueSize int) hashKeyFunc {
	return func(sig []uint64) string {
		s := make([]byte, hashValueSize*len(sig))
		buf := make([]byte, 8)
		for i, v := range sig {
			binary.LittleEndian.PutUint64(buf, v)
			copy(s[i*hashValueSize:(i+1)*hashValueSize], buf[:hashValueSize])
		}
		return string(s)
	}
}

// Compute the integral of function f, lower limit a, upper limit l, and
// precision defined as the quantize step
func integral(f func(float64) float64, a, b, precision float64) float64 {
	var area float64
	for x := a; x < b; x += precision {
		area += f(x+0.5*precision) * precision
	}
	return area
}

// Probability density function for false positive
func falsePositive(l, k int) func(float64) float64 {
	return func(j float64) float64 {
		return 1.0 - math.Pow(1.0-math.Pow(j, float64(k)), float64(l))
	}
}

// Probability density function for false negative
func falseNegative(l, k int) func(float64) float64 {
	return func(j float64) float64 {
		return 1.0 - (1.0 - math.Pow(1.0-math.Pow(j, float64(k)), float64(l)))
	}
}

// Compute the cummulative probability of false negative given threshold t
func probFalseNegative(l, k int, t, precision float64) float64 {
	return integral(falseNegative(l, k), t, 1.0, precision)
}

// Compute the cummulative probability of false positive given threshold t
func probFalsePositive(l, k int, t, precision float64) float64 {
	return integral(falsePositive(l, k), 0, t, precision)
}

// optimalKL returns the optimal K and L for Jaccard similarity search,
// and the false positive and negative probabilities.
// t is the Jaccard similarity threshold.
func optimalKL(numHash int, t float64) (optK, optL int, fp, fn float64) {
	minError := math.MaxFloat64
	for l := 1; l <= numHash; l++ {
		for k := 1; k <= numHash; k++ {
			if l*k > numHash {
				continue
			}
			currFp := probFalsePositive(l, k, t, integrationPrecision)
			currFn := probFalseNegative(l, k, t, integrationPrecision)
			currErr := currFn + currFp
			if minError > currErr {
				minError = currErr
				optK = k
				optL = l
				fp = currFp
				fn = currFn
			}
		}
	}
	return
}

// entry contains the hash key (from minhash signature) and the indexed key
type entry struct {
	hashKey string
	key     interface{}
}

// hashTable is a look-up table implemented as a slice sorted by hash keys.
// Look-up operation is implemented using binary search.
type hashTable []entry

func (h hashTable) Len() int           { return len(h) }
func (h hashTable) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h hashTable) Less(i, j int) bool { return h[i].hashKey < h[j].hashKey }

// MinhashLSH represents a MinHash LSH implemented using LSH Forest
// (http://ilpubs.stanford.edu:8090/678/1/2005-14.pdf).
// It supports query-time setting of the MinHash LSH parameters
// L (number of bands) and
// K (number of hash functions per band).
type MinhashLSH struct {
	k              int
	l              int
	hashTables     []hashTable
	hashKeyFunc    hashKeyFunc
	hashValueSize  int
	numIndexedKeys int
}

func newMinhashLSH(threshold float64, numHash, hashValueSize, initSize int) *MinhashLSH {
	k, l, _, _ := optimalKL(numHash, threshold)
	hashTables := make([]hashTable, l)
	for i := range hashTables {
		hashTables[i] = make(hashTable, 0, initSize)
	}
	return &MinhashLSH{
		k:              k,
		l:              l,
		hashValueSize:  hashValueSize,
		hashTables:     hashTables,
		hashKeyFunc:    hashKeyFuncGen(hashValueSize),
		numIndexedKeys: 0,
	}
}

// NewMinhashLSH64 uses 64-bit hash values and pre-allocation of hash tables.
func NewMinhashLSH64(numHash int, threshold float64, initSize int) *MinhashLSH {
	return newMinhashLSH(threshold, numHash, 8, initSize)
}

// NewMinhashLSH32 uses 32-bit hash values and pre-allocation of hash tables.
// MinHash signatures with 64 bit hash values will have
// their hash values trimed.
func NewMinhashLSH32(numHash int, threshold float64, initSize int) *MinhashLSH {
	return newMinhashLSH(threshold, numHash, 4, initSize)
}

// NewMinhashLSH16 uses 16-bit hash values and pre-allocation of hash tables.
// MinHash signatures with 64 or 32 bit hash values will have
// their hash values trimed.
func NewMinhashLSH16(numHash int, threshold float64, initSize int) *MinhashLSH {
	return newMinhashLSH(threshold, numHash, 2, initSize)
}

// NewMinhashLSH is the default constructor uses 32 bit hash value
// with pre-allocation of hash tables.
var NewMinhashLSH = NewMinhashLSH32

// Params returns the LSH parameters k and l
func (f *MinhashLSH) Params() (k, l int) {
	return f.k, f.l
}

func (f *MinhashLSH) hashKeys(sig []uint64) []string {
	hs := make([]string, f.l)
	for i := 0; i < f.l; i++ {
		hs[i] = f.hashKeyFunc(sig[i*f.k : (i+1)*f.k])
	}
	return hs
}

// Add a key with MinHash signature into the index.
// The key won't be searchable until Index() is called.
func (f *MinhashLSH) Add(key interface{}, sig []uint64) {
	// Generate hash keys
	hs := f.hashKeys(sig)
	// Insert keys into the hash tables by appending.
	for i := range f.hashTables {
		f.hashTables[i] = append(f.hashTables[i], entry{hs[i], key})
	}
}

// Index makes all the keys added searchable.
func (f *MinhashLSH) Index() {
	for i := range f.hashTables {
		sort.Sort(f.hashTables[i])
	}
	f.numIndexedKeys = len(f.hashTables[0])
}

// Query returns candidate keys given the query signature.
func (f *MinhashLSH) Query(sig []uint64) []interface{} {
	set := f.query(sig)
	results := make([]interface{}, 0, len(set))
	for key := range set {
		results = append(results, key)
	}
	return results
}

func (f *MinhashLSH) query(sig []uint64) map[interface{}]bool {
	// Generate hash keys.
	hashKeys := f.hashKeys(sig)
	results := make(map[interface{}]bool)
	// Query hash tables using binary search.
	for i := 0; i < f.l; i++ {
		// Only search over the indexed keys.
		hashTable := f.hashTables[i][:f.numIndexedKeys]
		hashKey := hashKeys[i]
		k := sort.Search(len(hashTable), func(x int) bool {
			return hashTable[x].hashKey >= hashKey
		})
		if k < len(hashTable) && hashTable[k].hashKey == hashKey {
			for j := k; j < len(hashTable) && hashTable[j].hashKey == hashKey; j++ {
				key := hashTable[j].key
				if _, exist := results[key]; !exist {
					results[key] = true
				}
			}
		}
	}
	return results
}
