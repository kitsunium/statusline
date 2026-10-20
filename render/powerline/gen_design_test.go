// Code generated from design/v1; DO NOT EDIT.
// design:digest sha256:f7116793c8d8ac76611661ce6107f6662b6eb477309df7bb89556666304dc4e6

package powerline

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

// TestPropertyCondensedMonotonic holds render/function/Condensed by property Monotonic.
func TestPropertyCondensedMonotonic(t *testing.T) {
	seed := designSeed("render/function/Condensed#Monotonic")
	useSeed(t, seed)
	propertyCondensedMonotonic(t, seed)
}

// TestPropertyIconsFromEnvOnlyNegativesSwitchOff holds render/function/IconsFromEnv by property OnlyNegativesSwitchOff.
func TestPropertyIconsFromEnvOnlyNegativesSwitchOff(t *testing.T) {
	seed := designSeed("render/function/IconsFromEnv#OnlyNegativesSwitchOff")
	useSeed(t, seed)
	propertyIconsFromEnvOnlyNegativesSwitchOff(t, seed)
}

// TestPropertyParseWidthBounded holds render/function/ParseWidth by property Bounded.
func TestPropertyParseWidthBounded(t *testing.T) {
	seed := designSeed("render/function/ParseWidth#Bounded")
	useSeed(t, seed)
	propertyParseWidthBounded(t, seed)
}

// TestPropertyRenderTwoLines holds render/function/Render by property TwoLines.
func TestPropertyRenderTwoLines(t *testing.T) {
	seed := designSeed("render/function/Render#TwoLines")
	useSeed(t, seed)
	propertyRenderTwoLines(t, seed)
}

// TestPropertyRenderLineOneFits holds render/function/Render by property LineOneFits.
func TestPropertyRenderLineOneFits(t *testing.T) {
	seed := designSeed("render/function/Render#LineOneFits")
	useSeed(t, seed)
	propertyRenderLineOneFits(t, seed)
}

// TestPropertyVisibleWidthIgnoresEscapes holds render/function/VisibleWidth by property IgnoresEscapes.
func TestPropertyVisibleWidthIgnoresEscapes(t *testing.T) {
	seed := designSeed("render/function/VisibleWidth#IgnoresEscapes")
	useSeed(t, seed)
	propertyVisibleWidthIgnoresEscapes(t, seed)
}

// TestPropertyVisibleWidthAdditive holds render/function/VisibleWidth by property Additive.
func TestPropertyVisibleWidthAdditive(t *testing.T) {
	seed := designSeed("render/function/VisibleWidth#Additive")
	useSeed(t, seed)
	propertyVisibleWidthAdditive(t, seed)
}
