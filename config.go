package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

type configData struct {
	fileFirst      bool
	hideHidden     bool
	hideSystem     bool
	hideLinks      bool
	hideMetaData   bool
	compactSizes   bool
	elideLongNames bool
	autoMore       bool
	sortAscending  bool
	sortDescending bool
	coloring       map[string]*color.Color
}

var lsConfigData configData = configData{
	fileFirst:      false,
	hideHidden:     false,
	hideSystem:     false,
	hideLinks:      false,
	hideMetaData:   false,
	compactSizes:   true,
	elideLongNames: true,
	autoMore:       true,
	sortAscending:  false,
	sortDescending: false,
	coloring:       make(map[string]*color.Color),
}

type configItems struct {
	items map[string]string
}

func (s *configItems) Set(v string) error {
	if s.items == nil {
		s.items = make(map[string]string)
	}

	items := strings.Split(v, ":")
	s.items[items[0]] = items[1]

	return nil
}

func (s *configItems) String() string {
	result := ""
	if len(s.items) != 0 {
		for key, val := range s.items {
			result += fmt.Sprintf("%s=%s ", key, val)
		}
	}
	return result
}

func constructColor(fore string, back string, bold bool) *color.Color {
	var c *color.Color = nil

	switch fore {
	case "red":
		c = color.New(color.FgRed)
	case "green":
		c = color.New(color.FgGreen)
	case "yellow":
		c = color.New(color.FgYellow)
	case "blue":
		c = color.New(color.FgBlue)
	case "magenta":
		c = color.New(color.FgMagenta)
	case "cyan":
		c = color.New(color.FgCyan)
	case "white":
		c = color.New(color.FgWhite)
	case "black":
		c = color.New(color.FgBlack)
	}

	if len(back) != 0 {
		switch back {
		case "red":
			c.Add(color.BgRed)
		case "green":
			c.Add(color.BgGreen)
		case "yellow":
			c.Add(color.BgYellow)
		case "blue":
			c.Add(color.BgBlue)
		case "magenta":
			c.Add(color.BgMagenta)
		case "cyan":
			c.Add(color.BgCyan)
		case "white":
			c.Add(color.BgWhite)
		case "black":
			c.Add(color.BgBlack)
		}
	}

	if bold {
		c.Add(color.Bold)
	}

	return c
}

func loadConfig() {
	appdata := os.Getenv("APPDATA")
	configFile := fmt.Sprintf("%s\\ls.json", appdata)

	lsConfigData.coloring["description"] = constructColor("yellow", "", false)
	lsConfigData.coloring["symlink"] = constructColor("cyan", "", true)
	lsConfigData.coloring["directories"] = constructColor("magenta", "", true)

	if _, err := os.Stat(configFile); err == nil {
		// read in the config (JSON)

		biuldColor := func(color_key string, alias string) {
			jsonKey := fmt.Sprint("color.", color_key)
			fore := viper.Get(fmt.Sprint(jsonKey, ".fore")).(string)
			back := viper.Get(fmt.Sprint(jsonKey, ".back")).(string)
			bold := viper.Get(fmt.Sprint(jsonKey, ".bold")).(bool)
			if len(alias) == 0 {
				lsConfigData.coloring[color_key] = constructColor(fore, back, bold)
			} else {
				lsConfigData.coloring[alias] = constructColor(fore, back, bold)
			}
		}

		viper.SetConfigType("json")
		viper.SetConfigFile(configFile)
		viper.ReadInConfig()

		if viper.IsSet("format.fileFirst") {
			lsConfigData.fileFirst = viper.Get("format.fileFirst").(bool)
		}

		if viper.IsSet("format.hideHidden") {
			lsConfigData.hideHidden = viper.Get("format.hideHidden").(bool)
		}

		if viper.IsSet("format.hideSystem") {
			lsConfigData.hideSystem = viper.Get("format.hideSystem").(bool)
		}

		if viper.IsSet("format.hideLinks") {
			lsConfigData.hideLinks = viper.Get("format.hideLinks").(bool)
		}

		if viper.IsSet("format.hideMetaData") {
			lsConfigData.hideMetaData = viper.Get("format.hideMetaData").(bool)
		}

		if viper.IsSet("format.compactSizes") {
			lsConfigData.compactSizes = viper.Get("format.compactSizes").(bool)
		}

		if viper.IsSet("format.elideLongNames") {
			lsConfigData.elideLongNames = viper.Get("format.elideLongNames").(bool)
		}

		if viper.IsSet("format.autoMore") {
			lsConfigData.autoMore = viper.Get("format.autoMore").(bool)
		}

		if viper.IsSet("color.description") {
			biuldColor("description", "")
		}

		if viper.IsSet("color.symlink") {
			biuldColor("symlink", "")
		}

		if viper.IsSet("color.directories") {
			biuldColor("directories", "")
		}

		if viper.IsSet("color.scm.D") {
			biuldColor("scm.D", "D")
		}

		if viper.IsSet("color.scm.R") {
			biuldColor("scm.R", "R")
		}

		if viper.IsSet("color.scm.A") {
			biuldColor("scm.A", "A")
		}

		if viper.IsSet("color.scm.M") {
			biuldColor("scm.M", "M")
		}

		if viper.IsSet("color.keys") {
			colorKeys := viper.Get("color.keys").(string)
			keys := strings.Split(colorKeys, ";")
			for _, key := range keys {
				jsonKey := fmt.Sprintf("color.%s", key)
				if viper.IsSet(jsonKey) {
					biuldColor(key, "")
				}
			}
		}
	}
}

func saveConfig(config configData) error {
	viper.Set("format.fileFirst", lsConfigData.fileFirst)
	viper.Set("format.hideHidden", lsConfigData.hideHidden)
	viper.Set("format.hideSystem", lsConfigData.hideSystem)
	viper.Set("format.hideLinks", lsConfigData.hideLinks)
	viper.Set("format.hideMetaData", lsConfigData.hideMetaData)
	viper.Set("format.compactSizes", lsConfigData.compactSizes)
	viper.Set("format.autoMore", lsConfigData.autoMore)

	return viper.WriteConfig()
}

func parseCommandLine() {
	flagFileFirst := flag.Bool("F", lsConfigData.fileFirst, "List files first")
	flagHideHidden := flag.Bool("H", lsConfigData.hideHidden, "Hide hidden entries")
	flagHideSystem := flag.Bool("S", lsConfigData.hideSystem, "Hide system entries")
	flagHideLinks := flag.Bool("L", lsConfigData.hideLinks, "Hide symlink targets")
	flagHideMetaData := flag.Bool("D", lsConfigData.hideMetaData, "Hide entry metadata")
	flagExpandSizes := flag.Bool("x", !lsConfigData.compactSizes, "Expand file sizes")
	flagSortAscending := flag.Bool("m", lsConfigData.hideMetaData, "Sort by ascending modification")
	flagSortDescending := flag.Bool("M", lsConfigData.hideMetaData, "Sort by descending modification")
	var cliConfigs configItems
	flag.Var(&cliConfigs, "config", "Permanently alter a configuration value")

	flag.Parse()

	if len(cliConfigs.items) != 0 {
		for key, val := range cliConfigs.items {
			if val == "true" {
				viper.Set(key, true)
			} else if val == "false" {
				viper.Set(key, false)
			} else {
				viper.Set(key, val)
			}
		}

		err := viper.WriteConfig()
		if err != nil {
			log.Panic(err)
		}

		success := color.New(color.FgGreen).Add(color.Bold)
		success.Println("Configuration successfully updated!")
		os.Exit(0)
	}

	lsConfigData.fileFirst = *flagFileFirst
	lsConfigData.hideHidden = *flagHideHidden
	lsConfigData.hideSystem = *flagHideSystem
	lsConfigData.hideLinks = *flagHideLinks
	lsConfigData.hideMetaData = *flagHideMetaData
	lsConfigData.compactSizes = !*flagExpandSizes
	lsConfigData.sortAscending = *flagSortAscending
	lsConfigData.sortDescending = *flagSortDescending
}
