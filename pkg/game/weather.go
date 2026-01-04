package game

import (
	"fmt"
	"math/rand"

	"rotmud/pkg/types"
)

// Time/weather constants matching ROT's merc.h
const (
	// Sunlight states
	SunDark  = 0
	SunRise  = 1
	SunLight = 2
	SunSet   = 3

	// Sky conditions
	SkyCloudless = 0
	SkyCloudy    = 1
	SkyRaining   = 2
	SkyLightning = 3

	// Time constants
	HoursPerDay   = 24
	DaysPerMonth  = 35
	MonthsPerYear = 17
)

// Day names (7 days in a week)
var dayNames = []string{
	"the Moon", "the Bull", "Deception", "Thunder",
	"Freedom", "the Great Gods", "the Sun",
}

// Month names (17 months)
var monthNames = []string{
	"Winter", "the Winter Wolf", "the Frost Giant", "the Old Forces",
	"the Grand Struggle", "the Spring", "Nature", "Futility",
	"the Dragon", "the Sun", "the Heat", "the Battle",
	"the Dark Shades", "the Shadows", "the Long Shadows",
	"the Ancient Darkness", "the Great Evil",
}

// TimeInfo tracks the current game time
type TimeInfo struct {
	Hour  int // 0-23
	Day   int // 0-34
	Month int // 0-16
	Year  int
}

// WeatherInfo tracks weather conditions
type WeatherInfo struct {
	Mmhg     int // Pressure (960-1040)
	Change   int // Pressure change rate (-12 to +12)
	Sky      int // Sky condition (cloudless, cloudy, raining, lightning)
	Sunlight int // Time of day (dark, rise, light, set)
}

// WorldTime manages game time and weather
type WorldTime struct {
	Time    TimeInfo
	Weather WeatherInfo
}

// NewWorldTime creates a new world time/weather system with defaults
func NewWorldTime() *WorldTime {
	return &WorldTime{
		Time: TimeInfo{
			Hour:  12, // Start at noon
			Day:   15,
			Month: 6, // Month of Nature
			Year:  650,
		},
		Weather: WeatherInfo{
			Mmhg:     1000, // Normal pressure
			Change:   0,
			Sky:      SkyCloudless,
			Sunlight: SunLight,
		},
	}
}

// Tick advances time by one hour and updates weather
// Returns any weather change message to broadcast
func (w *WorldTime) Tick() string {
	var messages []string

	// Advance hour
	w.Time.Hour++

	// Handle time of day transitions
	switch w.Time.Hour {
	case 5:
		w.Weather.Sunlight = SunLight
		messages = append(messages, "The day has begun.")
	case 6:
		w.Weather.Sunlight = SunRise
		messages = append(messages, "The sun rises in the east.")
	case 19:
		w.Weather.Sunlight = SunSet
		messages = append(messages, "The sun slowly disappears in the west.")
	case 20:
		w.Weather.Sunlight = SunDark
		messages = append(messages, "The night has begun.")
	case 24:
		w.Time.Hour = 0
		w.Time.Day++
	}

	// Handle day/month/year rollover
	if w.Time.Day >= DaysPerMonth {
		w.Time.Day = 0
		w.Time.Month++
	}
	if w.Time.Month >= MonthsPerYear {
		w.Time.Month = 0
		w.Time.Year++
	}

	// Update weather
	weatherMsg := w.updateWeather()
	if weatherMsg != "" {
		messages = append(messages, weatherMsg)
	}

	// Combine messages
	result := ""
	for _, msg := range messages {
		result += msg + "\r\n"
	}
	return result
}

// updateWeather adjusts pressure and sky conditions
func (w *WorldTime) updateWeather() string {
	// Seasonal pressure bias
	var diff int
	if w.Time.Month >= 9 && w.Time.Month <= 16 {
		// Winter months - low pressure more likely
		if w.Weather.Mmhg > 985 {
			diff = -2
		} else {
			diff = 2
		}
	} else {
		// Summer months - high pressure more likely
		if w.Weather.Mmhg > 1015 {
			diff = -2
		} else {
			diff = 2
		}
	}

	// Random pressure change
	w.Weather.Change += diff*dice(1, 4) + dice(2, 6) - dice(2, 6)

	// Clamp change rate
	if w.Weather.Change < -12 {
		w.Weather.Change = -12
	}
	if w.Weather.Change > 12 {
		w.Weather.Change = 12
	}

	// Apply change to pressure
	w.Weather.Mmhg += w.Weather.Change

	// Clamp pressure
	if w.Weather.Mmhg < 960 {
		w.Weather.Mmhg = 960
	}
	if w.Weather.Mmhg > 1040 {
		w.Weather.Mmhg = 1040
	}

	// Update sky condition based on pressure
	var message string
	switch w.Weather.Sky {
	case SkyCloudless:
		if w.Weather.Mmhg < 990 || (w.Weather.Mmhg < 1010 && rand.Intn(4) == 0) {
			message = "The sky is getting cloudy."
			w.Weather.Sky = SkyCloudy
		}

	case SkyCloudy:
		if w.Weather.Mmhg < 970 || (w.Weather.Mmhg < 990 && rand.Intn(4) == 0) {
			message = "It starts to rain."
			w.Weather.Sky = SkyRaining
		} else if w.Weather.Mmhg > 1030 && rand.Intn(4) == 0 {
			message = "The clouds disappear."
			w.Weather.Sky = SkyCloudless
		}

	case SkyRaining:
		if w.Weather.Mmhg < 970 && rand.Intn(4) == 0 {
			message = "Lightning flashes in the sky."
			w.Weather.Sky = SkyLightning
		} else if w.Weather.Mmhg > 1030 || (w.Weather.Mmhg > 1010 && rand.Intn(4) == 0) {
			message = "The rain stopped."
			w.Weather.Sky = SkyCloudy
		}

	case SkyLightning:
		if w.Weather.Mmhg > 1010 || (w.Weather.Mmhg > 990 && rand.Intn(4) == 0) {
			message = "The lightning has stopped."
			w.Weather.Sky = SkyRaining
		}
	}

	return message
}

// ControlWeather changes weather based on spell (positive = better, negative = worse)
func (w *WorldTime) ControlWeather(change int) {
	w.Weather.Change += change
	if w.Weather.Change < -12 {
		w.Weather.Change = -12
	}
	if w.Weather.Change > 12 {
		w.Weather.Change = 12
	}
}

// GetTimeString returns formatted time string
func (w *WorldTime) GetTimeString() string {
	hour12 := w.Time.Hour % 12
	if hour12 == 0 {
		hour12 = 12
	}
	suffix := "am"
	if w.Time.Hour >= 12 {
		suffix = "pm"
	}

	day := w.Time.Day + 1
	daySuffix := "th"
	if day > 4 && day < 20 {
		daySuffix = "th"
	} else if day%10 == 1 {
		daySuffix = "st"
	} else if day%10 == 2 {
		daySuffix = "nd"
	} else if day%10 == 3 {
		daySuffix = "rd"
	}

	dayName := dayNames[day%7]
	monthName := monthNames[w.Time.Month]

	return fmt.Sprintf("It is %d o'clock %s, Day of %s, %d%s the Month of %s.",
		hour12, suffix, dayName, day, daySuffix, monthName)
}

// GetWeatherString returns formatted weather string
func (w *WorldTime) GetWeatherString() string {
	skyLook := []string{
		"cloudless",
		"cloudy",
		"rainy",
		"lit by flashes of lightning",
	}

	wind := "a warm southerly breeze blows"
	if w.Weather.Change < 0 {
		wind = "a cold northern gust blows"
	}

	return fmt.Sprintf("The sky is %s and %s.", skyLook[w.Weather.Sky], wind)
}

// IsDark returns true if it's currently dark
func (w *WorldTime) IsDark() bool {
	return w.Weather.Sunlight == SunDark
}

// IsOutside returns true if the room is outdoors
func IsOutside(ch *types.Character) bool {
	if ch.InRoom == nil {
		return false
	}
	// Check if room has indoor flag
	return !ch.InRoom.Flags.Has(types.RoomIndoors)
}

// dice helper for random rolls
func dice(number, size int) int {
	if size <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < number; i++ {
		total += rand.Intn(size) + 1
	}
	return total
}
