package main

import (
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	lib "devstats"

	yaml "gopkg.in/yaml.v2"
)

// vars contain list of InfluxDB tag/value pairs
type vars struct {
	Vars []tag `yaml:"vars"`
}

// tag contain each InfluxDB tag data
type tag struct {
	Tag     string `yaml:"tag"`
	Name    string `yaml:"name"`
	Value   string `yaml:"value"`
	Command string `yaml:"command"`
}

// Insert InfluxDB vars
func idbVars() {
	// Environment context parse
	var ctx lib.Ctx
	ctx.Init()

	// Connect to InfluxDB
	ic := lib.IDBConn(&ctx)
	defer func() { lib.FatalOnError(ic.Close()) }()

	// Get BatchPoints
	var pts lib.IDBBatchPointsN
	bp := lib.IDBBatchPoints(&ctx, &ic)
	pts.NPoints = 0
	pts.Points = &bp

	// Local or cron mode?
	dataPrefix := lib.DataDir
	if ctx.Local {
		dataPrefix = "./"
	}

	// Read vars to generate
	data, err := ioutil.ReadFile(dataPrefix + ctx.VarsYaml)
	if err != nil {
		lib.FatalOnError(err)
		return
	}
	var allVars vars
	lib.FatalOnError(yaml.Unmarshal(data, &allVars))

	// No fields value needed
	fields := map[string]interface{}{"value": 0.0}

	// Iterate vars
	for _, tag := range allVars.Vars {
		if ctx.Debug > 0 {
			lib.Printf("Tag '%s', Name '%s', Value '%s', Command '%s'\n", tag.Tag, tag.Name, tag.Value, tag.Command)
		}
		if tag.Tag == "" || tag.Name == "" || (tag.Value == "" && tag.Command == "") {
			continue
		}
		// Drop current vars
		//lib.QueryIDB(ic, &ctx, "drop series from "+tag.Tag

		if tag.Command != "" {
			cmdBytes, err := exec.Command(tag.Command).CombinedOutput()
			if err != nil {
				lib.FatalOnError(err)
				return
			}
			outString := strings.TrimSpace(string(cmdBytes))
			if outString != "" {
				tag.Value = outString
				if ctx.Debug > 0 {
					lib.Printf("Tag '%s', Name '%s', New Value '%s'\n", tag.Tag, tag.Name, tag.Value)
				}
			}
		}

		// Insert tag name/value
		lib.IDBAddPointN(
			&ctx,
			&ic,
			&pts,
			lib.IDBNewPointWithErr(
				tag.Tag,
				map[string]string{tag.Name: tag.Value},
				fields,
				lib.TimeParseAny("2014"),
			),
		)
	}

	// Write the batch
	if !ctx.SkipIDB {
		lib.FatalOnError(lib.IDBWritePointsN(&ctx, &ic, &pts))
	} else if ctx.Debug > 0 {
		lib.Printf("Skipping vars series write\n")
	}
}

func main() {
	dtStart := time.Now()
	idbVars()
	dtEnd := time.Now()
	lib.Printf("Time: %v\n", dtEnd.Sub(dtStart))
}
