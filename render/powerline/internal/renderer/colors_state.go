// Package renderer provides status line rendering.
package renderer

// State colours for quota pills. Backgrounds stay pale and text stays dark so
// that the whole line reads on both light and dark terminal themes, matching
// the pastel palette the other segments already use.
const (
	// BgQuota is the neutral pale background of a quota pill.
	BgQuota string = "\033[48;5;252m"
	// FgQuota is the neutral pale foreground for quota pill caps.
	FgQuota string = "\033[38;5;252m"
	// FgQuotaText is the dark neutral text on a quota pill.
	FgQuotaText string = "\033[38;5;240m"
	// FgQuotaDim is the muted text for secondary values on a quota pill.
	FgQuotaDim string = "\033[38;5;246m"

	// BgAhead is the pale green background of a comfortable quota.
	BgAhead string = "\033[48;5;151m"
	// FgAhead is the pale green foreground for comfortable pill caps.
	FgAhead string = "\033[38;5;151m"
	// FgAheadText is the dark green text on a comfortable quota.
	FgAheadText string = "\033[38;5;22m"

	// BgBehind is the pale amber background of a strained quota.
	BgBehind string = "\033[48;5;222m"
	// FgBehind is the pale amber foreground for strained pill caps.
	FgBehind string = "\033[38;5;222m"
	// FgBehindText is the dark amber text on a strained quota.
	FgBehindText string = "\033[38;5;130m"

	// BgOverrun is the pale red background of an overrunning quota.
	BgOverrun string = "\033[48;5;174m"
	// FgOverrun is the pale red foreground for overrunning pill caps.
	FgOverrun string = "\033[38;5;174m"
	// FgOverrunText is the dark red text on an overrunning quota.
	FgOverrunText string = "\033[38;5;88m"

	// FgBarFill is the filled portion of a quota bar.
	FgBarFill string = "\033[38;5;238m"
	// Dim renders secondary text at reduced intensity.
	Dim string = "\033[2m"
)

// Per-quota hues. Each bucket gets its own pale ground so the quota line reads
// as a set of distinct objects rather than one grey slab, and so a quota is
// recognised by its colour before its label is read. The hues run cool to warm
// as the window widens: conversation, then five hours, then seven days.
const (
	// BgHueSession is the pale blue ground of the five-hour quota.
	BgHueSession string = "\033[48;5;153m"
	// FgHueSession is the pale blue cap of the five-hour quota.
	FgHueSession string = "\033[38;5;153m"
	// FgHueSessionInk is the deep blue ink on the session ground.
	FgHueSessionInk string = "\033[38;5;25m"

	// BgHueWeekly is the mint ground of the seven-day quota.
	BgHueWeekly string = "\033[48;5;158m"
	// FgHueWeekly is the mint cap of the seven-day quota.
	FgHueWeekly string = "\033[38;5;158m"
	// FgHueWeeklyInk is the deep green ink on the weekly ground.
	FgHueWeeklyInk string = "\033[38;5;29m"

	// BgHueCost is the blush ground of the credit balance.
	BgHueCost string = "\033[48;5;224m"
	// FgHueCost is the blush cap of the credit balance.
	FgHueCost string = "\033[38;5;224m"
	// FgHueCostInk is the deep red ink on the credit ground.
	FgHueCostInk string = "\033[38;5;95m"
)

// Service health colours, drawn as the glyph on the white OS segment. Each
// clears the 3:1 floor for graphical objects on that ground.
const (
	// FgHealthOK marks every counted service as operational. 3.91:1.
	FgHealthOK string = "\033[38;5;29m"
	// FgHealthDegraded marks one degraded service. 3.28:1.
	FgHealthDegraded string = "\033[38;5;166m"
	// FgHealthDown marks two degraded services or one major outage. 4.65:1.
	FgHealthDown string = "\033[38;5;160m"
)

// Epic pill colours: a mauve ground with its deep plum ink, like the other
// pills. The cells are graphical objects held to 3:1 on the ground; the track
// of the cells not under way only has to stay visible.
const (
	// BgEpic is the mauve ground of an epic pill.
	BgEpic string = "\033[48;5;182m"
	// FgEpic draws the pill's rounded caps in the ground colour.
	FgEpic string = "\033[38;5;182m"
	// FgEpicInk is the plum ink of the label and the done cells. 6.66:1.
	FgEpicInk string = "\033[38;5;53m"

	// epicActiveTrue is the amber of the cell under way, #82480b. 3.82:1.
	epicActiveTrue string = "\033[38;2;130;72;11m"
	// epicActive256 is its cube fallback, #875f00. 3.00:1.
	epicActive256 string = "\033[38;5;94m"
	// epicPulseTrue is the bright frame of the cell under way, #9e5204, bold;
	// the bold is closed by the Reset that follows the cell. 3.00:1.
	epicPulseTrue string = "\033[1;38;2;158;82;4m"
	// epicPulse256 is its cube fallback: the same 94, bold. The cube has no
	// brighter amber that holds 3:1 on the mauve (130 falls to 2.47:1), so
	// only the weight pulses there.
	epicPulse256 string = "\033[1;38;5;94m"
	// epicTrackTrue is the pale track of the cells not under way, #b48cb4.
	epicTrackTrue string = "\033[38;2;180;140;180m"
	// epicTrack256 is its cube fallback, #af87af.
	epicTrack256 string = "\033[38;5;139m"
)

// Epic cell inks for the active colour depth.
var (
	// FgEpicActive is the cell under way.
	FgEpicActive string = pickInk(epicActiveTrue, epicActive256)
	// FgEpicPulse is the bright frame of the cell under way.
	FgEpicPulse string = pickInk(epicPulseTrue, epicPulse256)
	// FgEpicTrack is a cell pending or waiting on the user.
	FgEpicTrack string = pickInk(epicTrackTrue, epicTrack256)
)
