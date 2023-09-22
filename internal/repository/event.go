package repository

// ColorMap is a map to convert from [Notion] colors to [Google Calendar] colors
//
// [Notion]: https://developers.notion.com/reference/property-object#multi-select
// [Google Calendar]: https://developers.google.com/calendar/api/v3/reference/colors/get?hl=ja
var ColorMap = map[string]string{
	"blue":    "9",
	"brown":   "10",
	"default": "1",
	"gray":    "8",
	"green":   "2",
	"orange":  "6",
	"pink":    "4",
	"purple":  "3",
	"red":     "11",
	"yellow":  "5",
}

