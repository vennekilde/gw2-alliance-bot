package internal

import "regexp"

// World represents a server world
type World struct {
	ID   int
	Name string
}

// NormalizedWorldName returns the string representation of the world by its id
func NormalizedWorldName(worldID int) string {
	world := WorldNames[worldID]
	if worldID != world.ID {
		return ""
	}
	re := regexp.MustCompile(`[^a-zA-Z]`)
	re2 := regexp.MustCompile(`\\[.*\\]`)
	name := re.ReplaceAllString(world.Name, "")
	name = re2.ReplaceAllString(name, "")
	return name
}

// WorldNames is a hardcoded list of all world id's and its respective world representation object
var WorldNames = map[int]World{
	1001: {1001, "Anvil Rock"},
	1002: {1002, "Borlis Pass"},
	1003: {1003, "Yak's Bend"},
	1004: {1004, "Henge of Denravi"},
	1005: {1005, "Maguuma"},
	1006: {1006, "Sorrow's Furnace"},
	1007: {1007, "Gate of Madness"},
	1008: {1008, "Jade Quarry"},
	1009: {1009, "Fort Aspenwood"},
	1010: {1010, "Ehmry Bay"},
	1011: {1011, "Stormbluff Isle"},
	1012: {1012, "Darkhaven"},
	1013: {1013, "Sanctum of Rall"},
	1014: {1014, "Crystal Desert"},
	1015: {1015, "Isle of Janthir"},
	1016: {1016, "Sea of Sorrows"},
	1017: {1017, "Tarnished Coast"},
	1018: {1018, "Northern Shiverpeaks"},
	1019: {1019, "Blackgate"},
	1020: {1020, "Ferguson's Crossing"},
	1021: {1021, "Dragonbrand"},
	1022: {1022, "Kaineng"},
	1023: {1023, "Devona's Rest"},
	1024: {1024, "Eredon Terrace"},
	2001: {2001, "Fissure of Woe"},
	2002: {2002, "Desolation"},
	2003: {2003, "Gandara"},
	2004: {2004, "Blacktide"},
	2005: {2005, "Ring of Fire"},
	2006: {2006, "Underworld"},
	2007: {2007, "Far Shiverpeaks"},
	2008: {2008, "Whiteside Ridge"},
	2009: {2009, "Ruins of Surmia"},
	2010: {2010, "Seafarer's Rest"},
	2011: {2011, "Vabbi"},
	2012: {2012, "Piken Square"},
	2013: {2013, "Aurora Glade"},
	2014: {2014, "Gunnar's Hold"},
	2101: {2101, "Jade Sea [FR]"},
	2102: {2102, "Fort Ranik [FR]"},
	2103: {2103, "Augury Rock [FR]"},
	2104: {2104, "Vizunah Square [FR]"},
	2105: {2105, "Arborstone [FR]"},
	2201: {2201, "Kodash [DE]"},
	2202: {2202, "Riverside [DE]"},
	2203: {2203, "Elona Reach [DE]"},
	2204: {2204, "Abaddon's Mouth [DE]"},
	2205: {2205, "Drakkar Lake [DE]"},
	2206: {2206, "Miller's Sound [DE]"},
	2207: {2207, "Dzagonur [DE]"},
	2301: {2301, "Baruch Bay [SP]"},
}
