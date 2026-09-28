// Package renderer provides status line rendering.
package renderer

import "strings"

// Color constants for terminal rendering using ANSI escape codes.
const (
	// Reset resets all terminal attributes.
	Reset string = "\033[0m"
	// Bold enables bold text.
	Bold string = "\033[1m"
	// BgBlue is the dark ground of the path segment.
	BgBlue string = "\033[48;5;236m"
	// BgGit is the periwinkle ground of the git segment. It is deliberately
	// blue rather than cyan: the context pill next to it is 152, a pale cyan,
	// and the two read as the same colour when they share a hue family.
	BgGit string = "\033[48;5;111m"
	// BgWhite is the white background color.
	BgWhite string = "\033[48;5;255m"
	// FgBlueDark is the light ink on the path ground.
	FgBlueDark string = "\033[38;5;250m"
	// FgGitInk is the deep navy ink on the git ground (8.24:1).
	FgGitInk string = "\033[38;5;17m"
	// BgHaiku is the powder blush background for Haiku.
	BgHaiku string = "\033[48;5;224m"
	// BgSonnet is the powder lavender background for Sonnet.
	BgSonnet string = "\033[48;5;189m"
	// BgOpus is the powder peach background for Opus.
	BgOpus string = "\033[48;5;223m"
	// BgFable is the mint background for Fable. The other families are all
	// warm — orange, violet, pink — so a cool hue tells Fable apart at once.
	BgFable string = "\033[48;5;194m"
	// BgModelUnknown is the neutral background for an unrecognised model. It is
	// deliberately not white: that is the OS segment's colour, and a model
	// sharing it merges with the segment before it, separator included.
	BgModelUnknown string = "\033[48;5;252m"
	// FgBlue is the path ground as a foreground, for separators.
	FgBlue string = "\033[38;5;236m"
	// FgGit is the git ground as a foreground, for separators.
	FgGit string = "\033[38;5;111m"
	// FgWhite is the white foreground color.
	FgWhite string = "\033[38;5;255m"
	// FgBlack is the black foreground color.
	FgBlack string = "\033[38;5;232m"
	// FgYellow is the yellow foreground color.
	FgYellow string = "\033[38;5;220m"
	// FgHaiku is the powder blush foreground for Haiku pill caps.
	FgHaiku string = "\033[38;5;224m"
	// FgSonnet is the powder lavender foreground for Sonnet pill caps.
	FgSonnet string = "\033[38;5;189m"
	// FgOpus is the powder peach foreground for Opus pill caps.
	FgOpus string = "\033[38;5;223m"
	// FgFable is the mint foreground for Fable pill caps.
	FgFable string = "\033[38;5;194m"
	// FgModelUnknown is the neutral foreground for an unrecognised model.
	FgModelUnknown string = "\033[38;5;252m"
	// FgModelUnknownDark is the grey ink on an unrecognised model. 5.39:1.
	FgModelUnknownDark string = "\033[38;5;239m"
	// BgGreen is the pale green background for lines added.
	BgGreen string = "\033[48;5;114m"
	// FgGreenText is the darker green for text on green background.
	FgGreenText string = "\033[38;5;22m"
	// FgGreenSep is the pale green foreground for separator.
	FgGreenSep string = "\033[38;5;114m"
	// BgRed is the pale red background for lines removed.
	BgRed string = "\033[48;5;174m"
	// FgRedText is the darker red for text on red background.
	FgRedText string = "\033[38;5;52m"
	// FgRedSep is the pale red foreground for separator.
	FgRedSep string = "\033[38;5;174m"
	// ColorGray is the gray foreground for incomplete progress.
	ColorGray string = "\033[38;5;245m"
	// FgGreenDone is the green foreground for done tasks in segmented bar.
	FgGreenDone string = "\033[38;5;34m"
	// FgYellowWip is the yellow/orange foreground for WIP indicator.
	FgYellowWip string = "\033[38;5;214m"
	// FgGrayTodo is the dark gray foreground for todo tasks.
	FgGrayTodo string = "\033[38;5;240m"
	// FgGraySep is the separator color between epics.
	FgGraySep string = "\033[38;5;245m"
	// FgCyanTask is the cyan foreground for current task indicator.
	FgCyanTask string = "\033[38;5;44m"
	// BgMCPEnabled is the pale teal background for enabled MCP servers.
	BgMCPEnabled string = "\033[48;5;116m"
	// FgMCPEnabled is the pale teal foreground for enabled MCP pill caps.
	FgMCPEnabled string = "\033[38;5;116m"
	// FgMCPEnabledText is the dark teal for text on enabled MCP background.
	FgMCPEnabledText string = "\033[38;5;23m"
	// BgMCPLabel is the dark teal ground of the MCP pill while a call is
	// in flight.
	BgMCPLabel string = "\033[48;5;23m"
	// FgMCPMuted is the gray ink of the disabled count inside the MCP pill.
	// 240 only reaches 4.31:1 on the teal; 239 clears 4.5:1 (5.05:1).
	FgMCPMuted string = "\033[38;5;239m"
	// FgMCPOnWhite is the dark teal of the MCP glyph on the OS segment's
	// white ground (6.46:1).
	FgMCPOnWhite string = "\033[38;5;23m"
	// FgMCPMutedOnWhite is the gray of the disabled count on the OS
	// segment's white ground: 241 (5.26:1); 242 barely clears 4.5 and 244
	// falls to 3.4:1.
	FgMCPMutedOnWhite string = "\033[38;5;241m"
	// StrikeMCP crosses out the disabled count; terminals without it keep
	// the gray alone.
	StrikeMCP string = "\033[9m"
	// BgWeekly is the pale gray background for weekly usage segment.
	BgWeekly string = "\033[48;5;252m"
	// FgWeekly is the pale gray foreground for weekly usage separator.
	FgWeekly string = "\033[38;5;252m"
	// FgWeeklyText is the dark gray text on weekly usage background.
	FgWeeklyText string = "\033[38;5;240m"
)

// Model inks. Each is its ground's hue, darkened until it reads at about
// 5.4:1 — legible without sitting on the pill like a block of ink. The text,
// the quota bars and the pace cursor all take it.
//
// The 256-colour cube has nothing in these hues between a deep ink at 7:1 and
// one under 4.5:1, so the tuned inks are 24-bit. Each has a 256-colour
// fallback: the nearest cube colour (CIELAB) that still clears 4.5:1.
const (
	// inkHaikuTrue is the muted red ink on the Haiku ground. 5.44:1.
	inkHaikuTrue string = "\033[38;2;155;52;52m"
	// inkHaiku256 is its cube fallback, #870000. 7.86:1.
	inkHaiku256 string = "\033[38;5;88m"
	// inkSonnetTrue is the muted indigo ink on the Sonnet ground. 5.43:1.
	inkSonnetTrue string = "\033[38;2;66;66;192m"
	// inkSonnet256 is its cube fallback, #5f00af. 7.15:1.
	inkSonnet256 string = "\033[38;5;55m"
	// inkOpusTrue is the muted brown ink on the Opus ground. 5.42:1.
	inkOpusTrue string = "\033[38;2;117;78;39m"
	// inkOpus256 is its cube fallback, #585858. The nearer olive (58) reads
	// as dirty green on peach, so the next nearest, a neutral grey, is used.
	// 5.28:1.
	inkOpus256 string = "\033[38;5;240m"
	// inkFableTrue is the muted green ink on the Fable ground. 5.45:1.
	inkFableTrue string = "\033[38;2;38;114;38m"
	// inkFable256 is its cube fallback, #005f00. 7.27:1.
	inkFable256 string = "\033[38;5;22m"
)

// Active model inks, resolved once at startup from the terminal's colour depth.
var (
	// FgHaikuDark is the ink on the Haiku background.
	FgHaikuDark string = pickInk(inkHaikuTrue, inkHaiku256)
	// FgSonnetDark is the ink on the Sonnet background.
	FgSonnetDark string = pickInk(inkSonnetTrue, inkSonnet256)
	// FgOpusDark is the ink on the Opus background.
	FgOpusDark string = pickInk(inkOpusTrue, inkOpus256)
	// FgFableDark is the ink on the Fable background.
	FgFableDark string = pickInk(inkFableTrue, inkFable256)
)

// Model tracks: a pale tint of each pill, between its ground and its ink, for
// the unreached levels of the effort gauge. Like the inks, the tuned tints are
// 24-bit with a nearest 256-colour fallback.
const (
	// trackHaikuTrue is the pale tint of the Haiku pill.
	trackHaikuTrue string = "\033[38;2;215;150;150m"
	// trackHaiku256 is its cube fallback.
	trackHaiku256 string = "\033[38;5;174m"
	// trackSonnetTrue is the pale tint of the Sonnet pill.
	trackSonnetTrue string = "\033[38;2;160;160;220m"
	// trackSonnet256 is its cube fallback; 147 sits too close to the ground.
	trackSonnet256 string = "\033[38;5;146m"
	// trackOpusTrue is the pale tint of the Opus pill.
	trackOpusTrue string = "\033[38;2;205;165;125m"
	// trackOpus256 is its cube fallback.
	trackOpus256 string = "\033[38;5;180m"
	// trackFableTrue is the pale tint of the Fable pill.
	trackFableTrue string = "\033[38;2;140;195;140m"
	// trackFable256 is its cube fallback.
	trackFable256 string = "\033[38;5;108m"
	// trackModelUnknown is the pale tint of an unrecognised model's pill.
	trackModelUnknown string = "\033[38;5;247m"
)

// Active model tracks, resolved once at startup from the colour depth.
var (
	// fgHaikuTrack is the Haiku pill tint.
	fgHaikuTrack string = pickInk(trackHaikuTrue, trackHaiku256)
	// fgSonnetTrack is the Sonnet pill tint.
	fgSonnetTrack string = pickInk(trackSonnetTrue, trackSonnet256)
	// fgOpusTrack is the Opus pill tint.
	fgOpusTrack string = pickInk(trackOpusTrue, trackOpus256)
	// fgFableTrack is the Fable pill tint.
	fgFableTrack string = pickInk(trackFableTrue, trackFable256)
)

// GetModelTrack returns the pale tint of a model pill.
//
// Params:
//   - modelName: the display name of the AI model
//
// Returns:
//   - string: foreground escape of the pill tint
func GetModelTrack(modelName string) string {
	nameLower := strings.ToLower(modelName)
	// Match the families the same way GetModelColors does
	switch {
	// Haiku pill
	case strings.Contains(nameLower, "haiku"):
		return fgHaikuTrack
	// Sonnet pill
	case strings.Contains(nameLower, "sonnet"):
		return fgSonnetTrack
	// Opus pill
	case strings.Contains(nameLower, "opus"):
		return fgOpusTrack
	// Fable pill
	case strings.Contains(nameLower, "fable"):
		return fgFableTrack
	// Unrecognised model
	default:
		return trackModelUnknown
	}
}

// GetModelColors returns colors for a model pill.
//
// Params:
//   - modelName: the display name of the AI model
//
// Returns:
//   - bgColor: background color for the pill
//   - fgColor: foreground color for the pill caps
//   - textColor: darker foreground color for text on the pill
func GetModelColors(modelName string) (bgColor, fgColor, textColor string) {
	nameLower := strings.ToLower(modelName)
	// Detect model type and return appropriate colors
	switch {
	// Pink colors for Haiku model
	case strings.Contains(nameLower, "haiku"):
		// Return Haiku colors
		return BgHaiku, FgHaiku, FgHaikuDark
	// Purple colors for Sonnet model
	case strings.Contains(nameLower, "sonnet"):
		// Return Sonnet colors
		return BgSonnet, FgSonnet, FgSonnetDark
	// Orange colors for Opus model
	case strings.Contains(nameLower, "opus"):
		// Return Opus colors
		return BgOpus, FgOpus, FgOpusDark
	// Mint colors for Fable model. Without an entry of its own Fable fell back
	// to white, which is the OS segment's colour: the two segments then merged
	// into one block with an invisible separator between them.
	case strings.Contains(nameLower, "fable"):
		// Return Fable colors
		return BgFable, FgFable, FgFableDark
	// Neutral colors for unknown models
	default:
		// Return default colors
		return BgModelUnknown, FgModelUnknown, FgModelUnknownDark
	}
}
