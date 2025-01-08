package world

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/MrGunflame/gw2api"
	"github.com/vennekilde/gw2-alliance-bot/internal/api"
	"go.uber.org/zap"
)

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

type Team struct {
	ID                int
	Name              string
	WorldEquivalentID int
}

var TeamNames = map[int]Team{
	// EU
	12001: {12001, "Skrittsburgh", 2001},
	12002: {12002, "Fotune's Vale", 2002},
	12003: {12003, "Silent Woods", 2003},
	12004: {12004, "Ettin's Back", 2004},
	12005: {12005, "Domain of Anguish", 2005},
	12006: {12006, "Palawadan", 2006},
	12007: {12007, "Bloodstone Gulch", 2007},
	12008: {12008, "Frost Citadel", 2008},
	12009: {12009, "Dragrimmar", 2009},
	12010: {12010, "Grenth's Door", 2010},
	12011: {12011, "Mirror of Lyssa", 2011},
	12012: {12012, "Melandru's Dome", 2012},
	12013: {12013, "Kormir's Library", 2013},
	12014: {12014, "Great House Aviary", 2014},
	12015: {12015, "Bava Nisos", 2101},
	12016: {12016, "Temple of Febe", 2102},
	12017: {12017, "Gyala Hatchery", 2103},
	12018: {12018, "Grekvelnn Burrows", 2104},
	// NA
	11001: {11001, "Moogooloo", 1001},
	11002: {11002, "Rall's Rest", 1002},
	11003: {11003, "Domain of Torment", 1003},
	11004: {11004, "Yohlon Haven", 1004},
	11005: {11005, "Tombs of Drascir", 1005},
	11006: {11006, "Hall of Judgment", 1006},
	11007: {11007, "Throne of Balthazar", 1007},
	11008: {11008, "Dwayna's Temple", 1008},
	11009: {11009, "Abbaddon's Prison", 1009},
	11010: {11010, "Ruined Cathedral of Blood", 1010},
	11011: {11011, "Lutgardis Conservatory", 1011},
	11012: {11012, "Mosswood", 1012},
	11013: {11013, "Mithric Cliffs", 1013},
	11014: {11014, "Lagula's Kraal", 1014},
	11015: {11015, "De Molish Post", 1015},
}

func WorldsSorted() []World {
	worlds := make([]World, 0, len(WorldNames))
	for _, world := range WorldNames {
		worlds = append(worlds, world)
	}
	sort.Slice(worlds, func(i, j int) bool {
		return worlds[i].Name < worlds[j].Name
	})
	return worlds
}

type worldSyncError error

type LinkedWorlds map[string]api.WorldLinks

// Errors raised.
var (
	ErrWorldsNotSynced worldSyncError = errors.New("worlds are not synched")
)

type Worlds struct {
	linkedWorlds       LinkedWorlds
	lastEndTime        time.Time
	isWorldLinksSynced bool

	gw2API *gw2api.Session
}

func NewWorlds(gw2API *gw2api.Session) *Worlds {
	return &Worlds{
		gw2API: gw2API,
	}
}

func (ws *Worlds) Start() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		first := true
		for {
			zap.L().Info("synchronizing linked worlds")
			if err := ws.SynchronizeWorldLinks(ws.gw2API); err != nil {
				zap.L().Error("unable to synchronize matchup", zap.Error(err))
			} else if first {
				wg.Done()
				first = false
			}

			if !ws.lastEndTime.IsZero() {
				// Sleep until next match
				sleepUntil := time.Until(ws.lastEndTime)
				zap.L().Info("synchronizing linked worlds once matchup is over",
					zap.Duration("synchronizing timer", sleepUntil),
					zap.Time("endtime", ws.lastEndTime))
				// Sleep for at least a minute to not spam the api
				if sleepUntil < time.Minute {
					sleepUntil = time.Minute
				}
				time.Sleep(sleepUntil)
			} else {
				zap.L().Info("synchronizing linked worlds in 5 minutes")
				time.Sleep(time.Minute * 5)
			}
		}
	}()
	wg.Wait()
}

func (ws *Worlds) SynchronizeWorldLinks(gw2API *gw2api.Session) error {
	matches, err := gw2API.WvWMatches()
	if err != nil {
		return err
	}

	// Sanity check before we go and reset world links before we actually have a new matchup
	if len(matches) > 0 {
		lw := createEmptyLinkedWorldsMap()
		// reset timer to avoid it not being changed by the loop
		lowestEndTime := time.Time{}
		foundWorlds := 0
		for _, match := range matches {
			zap.L().Info("matchup fetched",
				zap.Any("id", match.ID),
				zap.Any("endtime", match.EndTime),
				zap.Any("reds", match.AllWorlds["red"]),
				zap.Any("blues", match.AllWorlds["blue"]),
				zap.Any("greens", match.AllWorlds["green"]))

			// Persist world link
			lw.setWorldLinks(match.AllWorlds["red"])
			lw.setWorldLinks(match.AllWorlds["blue"])
			lw.setWorldLinks(match.AllWorlds["green"])
			// bump found world counter
			foundWorlds += len(match.AllWorlds["red"]) +
				len(match.AllWorlds["blue"]) +
				len(match.AllWorlds["green"])

			// Parse match end time
			matchEndTime, err := time.Parse(time.RFC3339, match.EndTime)
			if err != nil {
				zap.L().Error("unable to parse matchup end time", zap.Error(err))
				continue
			}

			if lowestEndTime.IsZero() || lowestEndTime.After(matchEndTime) {
				lowestEndTime = matchEndTime
			}
		}
		// Only update if we can find all worlds
		if foundWorlds >= len(WorldNames) {
			ws.setMatchupLinks(lw, lowestEndTime)
			zap.L().Info("Updated linked worlds", zap.Any("linked worlds", ws.linkedWorlds))
		} else {
			zap.L().Warn("not updating linked worlds, did not find all worlds in matchups",
				zap.Int("total worlds", len(WorldNames)),
				zap.Int("found worlds", len(lw)),
			)
		}
	}
	return nil
}

func (ws *Worlds) setMatchupLinks(lw LinkedWorlds, lowestEndTime time.Time) {
	ws.linkedWorlds = lw
	ws.lastEndTime = lowestEndTime
	ws.isWorldLinksSynced = true
}

func (lw LinkedWorlds) setWorldLinks(allWorlds []int) {
	for _, worldRefID := range allWorlds {
		links := []int{}
		for _, worldID := range allWorlds {
			if worldID != worldRefID {
				links = append(links, worldID)
			}
		}
		lw[strconv.Itoa(worldRefID)] = links
	}
}

func (ws *Worlds) IsWorldLinksSynchronized() bool {
	return ws.isWorldLinksSynced
}

func (ws *Worlds) GetWorldLinks(worldPerspective int) (links []int, err error) {
	if !ws.IsWorldLinksSynchronized() {
		return links, ErrWorldsNotSynced
	}
	return ws.linkedWorlds[strconv.Itoa(worldPerspective)], err
}

func (ws *Worlds) GetAllWorldLinks() LinkedWorlds {
	return ws.linkedWorlds
}

func createEmptyLinkedWorldsMap() LinkedWorlds {
	newLinkedWorlds := make(LinkedWorlds)
	for worldID := range WorldNames {
		newLinkedWorlds[strconv.Itoa(worldID)] = []int{}
	}
	return newLinkedWorlds
}

func matchHasWorld(match gw2api.WvWMatch, worldID int) bool {
	for _, world := range match.AllWorlds["red"] {
		if world == worldID {
			return true
		}
	}
	for _, world := range match.AllWorlds["blue"] {
		if world == worldID {
			return true
		}
	}
	for _, world := range match.AllWorlds["green"] {
		if world == worldID {
			return true
		}
	}
	return false
}
