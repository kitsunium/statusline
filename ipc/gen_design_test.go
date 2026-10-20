// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

package ipc

import (
	"flag"
	"hash/fnv"
	"os"
	"strconv"
	"testing"
)

// designDigest is the design the properties run under: with the property's
// path, it seeds its values, so a pull request replays the same ones.
const designDigest = "sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6"

// designSeed is the seed of a property: the design's digest and its path —
// and DESIGN_SEED when it is set.
func designSeed(path string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(designDigest + "|" + path + "|" + os.Getenv("DESIGN_SEED")))
	return h.Sum64()
}

// useSeed logs a property's seed and hands it to pgregory.net/rapid, whose
// only public way in is its -rapid.seed flag: rapid.Check then draws the same
// values in every run of the same design and the same DESIGN_SEED.
func useSeed(t *testing.T, seed uint64) {
	t.Helper()
	t.Logf("property seed %d (go test -rapid.seed=%d replays it)", seed, seed)
	if f := flag.Lookup("rapid.seed"); f != nil {
		_ = f.Value.Set(strconv.FormatUint(seed, 10))
	}
}

// TestPropertyInstanceCachePathInsideCache holds status/Instance.CachePath by property InsideCache.
func TestPropertyInstanceCachePathInsideCache(t *testing.T) {
	seed := designSeed("status/Instance.CachePath#InsideCache")
	useSeed(t, seed)
	propertyInstanceCachePathInsideCache(t, seed)
}

// TestPropertyKeyHashStable holds status/Key.Hash by property Stable.
func TestPropertyKeyHashStable(t *testing.T) {
	seed := designSeed("status/Key.Hash#Stable")
	useSeed(t, seed)
	propertyKeyHashStable(t, seed)
}

// TestPropertyCompareVersionsOrder holds status/function/CompareVersions by property Order.
func TestPropertyCompareVersionsOrder(t *testing.T) {
	seed := designSeed("status/function/CompareVersions#Order")
	useSeed(t, seed)
	propertyCompareVersionsOrder(t, seed)
}

// TestPropertyCompatibleMajorOne holds status/function/Compatible by property MajorOne.
func TestPropertyCompatibleMajorOne(t *testing.T) {
	seed := designSeed("status/function/Compatible#MajorOne")
	useSeed(t, seed)
	propertyCompatibleMajorOne(t, seed)
}

// TestPropertyHereUnderRuntimeDir holds status/function/Here by property UnderRuntimeDir.
func TestPropertyHereUnderRuntimeDir(t *testing.T) {
	seed := designSeed("status/function/Here#UnderRuntimeDir")
	useSeed(t, seed)
	propertyHereUnderRuntimeDir(t, seed)
}

// TestPropertyLocateDistinct holds status/function/Locate by property Distinct.
func TestPropertyLocateDistinct(t *testing.T) {
	seed := designSeed("status/function/Locate#Distinct")
	useSeed(t, seed)
	propertyLocateDistinct(t, seed)
}

// TestPropertyReadFrameRoundtrip holds status/function/ReadFrame by property Roundtrip.
func TestPropertyReadFrameRoundtrip(t *testing.T) {
	seed := designSeed("status/function/ReadFrame#Roundtrip")
	useSeed(t, seed)
	propertyReadFrameRoundtrip(t, seed)
}

// TestPropertyWriteFrameLengthPrefixed holds status/function/WriteFrame by property LengthPrefixed.
func TestPropertyWriteFrameLengthPrefixed(t *testing.T) {
	seed := designSeed("status/function/WriteFrame#LengthPrefixed")
	useSeed(t, seed)
	propertyWriteFrameLengthPrefixed(t, seed)
}
